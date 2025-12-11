package producer

import (
	"bytes"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/ethereum/go-ethereum/common"
)

const (
	paymentDataPrefix = "GORR_PAY:"

	// Treasury cut (fee) for merchant payments:
	// 250 bps = 2.5%
	treasuryFeeBps = 250
	bpsDenominator = 10000
)

type BlockProducer struct {
	chain  *blockchain.Blockchain
	logger *log.Logger
	quit   chan struct{}
	delay  time.Duration
	bus    *events.EventBus
}

func NewBlockProducer(chain *blockchain.Blockchain, logger *log.Logger, blockTime uint64, bus *events.EventBus) *BlockProducer {
	return &BlockProducer{
		chain:  chain,
		logger: logger,
		quit:   make(chan struct{}),
		delay:  time.Duration(blockTime) * time.Second,
		bus:    bus,
	}
}

func (bp *BlockProducer) Start() {
	go func() {
		ticker := time.NewTicker(bp.delay)
		for {
			select {
			case <-ticker.C:
				bp.produce()
			case <-bp.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (bp *BlockProducer) Stop() {
	close(bp.quit)
}

// ---------------------------------------------------------------
// BLOCK CREATION
// ---------------------------------------------------------------

func (bp *BlockProducer) produce() {
	head := bp.chain.Head()
	if head == nil {
		bp.logger.Error("No head block loaded in blockchain")
		return
	}

	newBlock := &types.Block{
		Header: &types.Header{
			ParentHash: head.Hash(),
			Number:     head.Header.Number + 1,
			Time:       uint64(time.Now().Unix()),
			StateRoot:  common.Hash{},
			TxRoot:     common.Hash{},
		},
		Transactions: []*types.Transaction{},
	}

	blockNum := newBlock.Header.Number
	blockTime := newBlock.Header.Time

	// Pending TXs
	txns := bp.chain.TxPool.Pending()
	receipts := []*types.Receipt{}

	for _, tx := range txns {
		if tx == nil {
			continue
		}

		from, err := tx.From()
		if err != nil {
			// fallback: gebruik Admin address
			bp.logger.Info(fmt.Sprintf("Invalid signature, using Admin as fallback sender: %v", err))
			from = bp.chain.AdminAddr
		}

		// Nonce check
		stateNonce, err := bp.chain.State.GetNonce(from)
		if err != nil {
			bp.logger.Error(fmt.Sprintf("GetNonce(%s) error: %v", from.Hex(), err))
			continue
		}
		if tx.Nonce != stateNonce {
			continue
		}

		// Detect payment intent
		intentID, isPayment := parsePaymentIntentID(tx.Data)

		var ok bool
		if isPayment {
			ok = bp.processPaymentGORR(tx, from, intentID, blockNum, blockTime)
		} else {
			ok = bp.processGORR(tx, from)
		}

		if !ok {
			continue
		}

		// Increase nonce
		if err := bp.chain.State.IncreaseNonce(from); err != nil {
			bp.logger.Error(fmt.Sprintf("IncreaseNonce error: %v", err))
			continue
		}

		// Add TX to block
		newBlock.Transactions = append(newBlock.Transactions, tx)

		// Index TX → block
		if err := bp.chain.SaveTxIndex(tx.Hash(), blockNum); err != nil {
			bp.logger.Error(fmt.Sprintf("SaveTxIndex error: %v", err))
		}

		// Build receipt
		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      blockNum,
			TransactionIndex: uint64(len(newBlock.Transactions) - 1),
			From:             from,
			To:               *tx.To,
			GasUsed:          tx.Gas,
			Status:           1,
		}
		receipts = append(receipts, receipt)

		// Remove from mempool
		bp.chain.TxPool.Remove(tx)
	}

	// Save receipts
	if err := bp.chain.SaveReceipts(blockNum, receipts); err != nil {
		bp.logger.Error(fmt.Sprintf("SaveReceipts error: %v", err))
	}

	// Update chain head
	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error(fmt.Sprintf("SetHead error: %v", err))
	}

	bp.logger.Info(fmt.Sprintf(
		"Produced block #%d | %d txs | Hash=%s",
		blockNum,
		len(newBlock.Transactions),
		newBlock.Hash().Hex(),
	))
}

// ---------------------------------------------------------------
// STANDARD GORR TRANSFER (NO FEE)
// ---------------------------------------------------------------

func (bp *BlockProducer) processGORR(tx *types.Transaction, from common.Address) bool {
	if tx.To == nil {
		bp.logger.Info("TX rejected: missing To address")
		return false
	}

	fromBal, err := bp.chain.State.GetBalance(from)
	if err != nil {
		bp.logger.Error("GetBalance(from) error: " + err.Error())
		return false
	}

	if fromBal.Cmp(tx.Value) < 0 {
		// insufficient balance
		return false
	}

	toBal, err := bp.chain.State.GetBalance(*tx.To)
	if err != nil {
		bp.logger.Error("GetBalance(to) error: " + err.Error())
		return false
	}

	// apply transfer
	newFrom := new(big.Int).Sub(fromBal, tx.Value)
	newTo := new(big.Int).Add(toBal, tx.Value)

	if err := bp.chain.State.SetBalance(from, newFrom); err != nil {
		bp.logger.Error("SetBalance(from) error: " + err.Error())
		return false
	}
	if err := bp.chain.State.SetBalance(*tx.To, newTo); err != nil {
		bp.logger.Error("SetBalance(to) error: " + err.Error())
		return false
	}

	return true
}

// ---------------------------------------------------------------
// PAYMENT WITH TREASURY FEE
// ---------------------------------------------------------------

func (bp *BlockProducer) processPaymentGORR(
	tx *types.Transaction,
	from common.Address,
	intentID uint64,
	blockNum uint64,
	blockTime uint64,
) bool {

	if tx.To == nil {
		bp.logger.Info("Payment TX rejected: missing merchant address")
		return false
	}

	if bp.chain.Payment == nil {
		bp.logger.Info("PaymentGateway is nil")
		return false
	}

	// Treasury must be known
	if bp.chain.TreasuryAddr == (common.Address{}) {
		bp.logger.Info("TreasuryAddr missing")
		return false
	}

	// Check intent
	intent, err := bp.chain.Payment.GetIntent(intentID)
	if err != nil {
		bp.logger.Info(fmt.Sprintf("Intent %d missing: %v", intentID, err))
		return false
	}

	// Check expiry
	if intent.IsExpired(blockTime) {
		bp.logger.Info(fmt.Sprintf("Intent %d expired", intentID))
		return false
	}

	if intent.Token != "GORR" {
		bp.logger.Info(fmt.Sprintf("Intent %d rejected token %s", intentID, intent.Token))
		return false
	}

	// Merchant must match tx.To
	if intent.Merchant != *tx.To {
		bp.logger.Info("Merchant mismatch")
		return false
	}

	// Balances
	fromBal, _ := bp.chain.State.GetBalance(from)
	if fromBal.Cmp(tx.Value) < 0 {
		bp.logger.Info("Insufficient balance")
		return false
	}

	// Treasury fee
	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))

	merchantAmount := new(big.Int).Sub(tx.Value, fee)

	// Load balances
	merchantBal, _ := bp.chain.State.GetBalance(*tx.To)
	treasuryBal, _ := bp.chain.State.GetBalance(bp.chain.TreasuryAddr)

	// Apply debits/credits
	newFrom := new(big.Int).Sub(fromBal, tx.Value)
	newMerchant := new(big.Int).Add(merchantBal, merchantAmount)
	newTreasury := new(big.Int).Add(treasuryBal, fee)

	bp.chain.State.SetBalance(from, newFrom)
	bp.chain.State.SetBalance(*tx.To, newMerchant)
	bp.chain.State.SetBalance(bp.chain.TreasuryAddr, newTreasury)

	// Mark intent paid
	if err := bp.chain.Payment.MarkPaidFromTx(
		intentID,
		from,
		*tx.To,
		tx.Value,
		tx.Hash(),
		blockNum,
		blockTime,
	); err != nil {
		bp.logger.Info(fmt.Sprintf("MarkPaidFromTx failed: %v", err))
	}

	bp.logger.Info(fmt.Sprintf(
		"Intent %d PAID | gross=%s fee=%s net=%s",
		intentID,
		tx.Value.String(),
		fee.String(),
		merchantAmount.String(),
	))

	return true
}

// ---------------------------------------------------------------
// HELPERS
// ---------------------------------------------------------------

func parsePaymentIntentID(data []byte) (uint64, bool) {
	if len(data) == 0 {
		return 0, false
	}

	if !bytes.HasPrefix(data, []byte(paymentDataPrefix)) {
		return 0, false
	}

	value := string(data[len(paymentDataPrefix):])
	id, err := strconv.ParseUint(value, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
