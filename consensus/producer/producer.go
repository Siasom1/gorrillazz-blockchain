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
	paymentDataPrefix = "GORR_PAY:" // tx.Data = "GORR_PAY:<intentID>"
	// Fee in basispunten → 250 = 2.5%
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

// ----------------------------------------------------------------
// BLOCK CREATION
// ----------------------------------------------------------------

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

	txns := bp.chain.TxPool.Pending()
	receipts := []*types.Receipt{}

	for _, tx := range txns {
		if tx == nil {
			continue
		}

		from, err := tx.From()
		if err != nil {
			// DEV fallback: gebruik admin als sender zodat je chain niet vastloopt
			bp.logger.Info(fmt.Sprintf("Invalid TX signature (%v); using AdminAddr as fallback sender", err))
			if bp.chain.AdminAddr == (common.Address{}) {
				bp.logger.Error("AdminAddr is zero address; cannot fallback sender")
				continue
			}
			from = bp.chain.AdminAddr
		}

		// NONCE check
		stateNonce, err := bp.chain.State.GetNonce(from)
		if err != nil {
			bp.logger.Error(fmt.Sprintf("GetNonce error for %s: %v", from.Hex(), err))
			continue
		}
		if tx.Nonce != stateNonce {
			// Nonce mismatch → skip (later opnieuw proberen)
			continue
		}

		// Detecteer payment intent in tx.Data
		intentID, isPayment := parsePaymentIntentID(tx.Data)
		var ok bool

		if isPayment {
			ok = bp.processPaymentGORR(tx, from, intentID, blockNum, blockTime)
		} else {
			ok = bp.processGORR(tx, from)
		}

		if !ok {
			// Onvoldoende saldo / intent ongeldig
			continue
		}

		// Nonce verhogen pas ná succesvolle verwerking
		if err := bp.chain.State.IncreaseNonce(from); err != nil {
			bp.logger.Error(fmt.Sprintf("IncreaseNonce error: %v", err))
			continue
		}

		// In block opnemen
		newBlock.Transactions = append(newBlock.Transactions, tx)

		// Tx indexeren voor eth_getTransactionReceipt / eth_getTransactionByHash
		if err := bp.chain.SaveTxIndex(tx.Hash(), blockNum); err != nil {
			bp.logger.Error(fmt.Sprintf("SaveTxIndex error: %v", err))
		}

		// Receipt opslaan in geheugen (later opslaan naar disk)
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

		// Uit de txpool halen
		bp.chain.TxPool.Remove(tx)
	}

	// HEAD updaten + block opslaan
	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error(fmt.Sprintf("SetHead error: %v", err))
	}

	// Receipts opslaan
	if err := bp.chain.SaveReceipts(newBlock.Header.Number, receipts); err != nil {
		bp.logger.Error(fmt.Sprintf("SaveReceipts error: %v", err))
	}

	bp.logger.Info(fmt.Sprintf(
		"Produced block #%d | %d txs | Hash=%s",
		newBlock.Header.Number,
		len(newBlock.Transactions),
		newBlock.Hash().Hex(),
	))
}

// ----------------------------------------------------------------
// Normale GORR transfer (zonder fee / payment intent)
// ----------------------------------------------------------------

func (bp *BlockProducer) processGORR(tx *types.Transaction, from common.Address) bool {
	if tx.To == nil {
		bp.logger.Info("TX rejected: nil To address")
		return false
	}

	fromBal, err := bp.chain.State.GetBalance(from)
	if err != nil {
		bp.logger.Error(fmt.Sprintf("GetBalance(from) error: %v", err))
		return false
	}

	if fromBal.Cmp(tx.Value) < 0 {
		// Onvoldoende saldo
		return false
	}

	toBal, err := bp.chain.State.GetBalance(*tx.To)
	if err != nil {
		bp.logger.Error(fmt.Sprintf("GetBalance(to) error: %v", err))
		return false
	}

	newFrom := new(big.Int).Sub(fromBal, tx.Value)
	newTo := new(big.Int).Add(toBal, tx.Value)

	if err := bp.chain.State.SetBalance(from, newFrom); err != nil {
		bp.logger.Error(fmt.Sprintf("SetBalance(from) error: %v", err))
		return false
	}
	if err := bp.chain.State.SetBalance(*tx.To, newTo); err != nil {
		bp.logger.Error(fmt.Sprintf("SetBalance(to) error: %v", err))
		return false
	}

	return true
}

// ----------------------------------------------------------------
// Payment GORR transfer (met treasury fee + PaymentGateway)
// ----------------------------------------------------------------

func (bp *BlockProducer) processPaymentGORR(
	tx *types.Transaction,
	from common.Address,
	intentID uint64,
	blockNum uint64,
	blockTime uint64,
) bool {
	if tx.To == nil {
		bp.logger.Info("Payment TX rejected: nil To address")
		return false
	}

	if bp.chain.Payment == nil {
		bp.logger.Info("Payment TX rejected: PaymentGateway is nil")
		return false
	}

	if bp.chain.TreasuryAddr == (common.Address{}) {
		bp.logger.Info("Payment TX rejected: TreasuryAddr is zero address")
		return false
	}

	// 1) Intent ophalen
	intent, err := bp.chain.Payment.GetIntent(intentID)
	if err != nil {
		bp.logger.Info(fmt.Sprintf("Payment intent %d not found: %v", intentID, err))
		return false
	}

	// Voor nu: alleen GORR-payments via native Value
	if intent.Token != "GORR" {
		bp.logger.Info(fmt.Sprintf("Payment intent %d has unsupported token %s (only GORR supported for now)", intentID, intent.Token))
		return false
	}

	// Merchant moet overeenkomen met tx.To
	if intent.Merchant != *tx.To {
		bp.logger.Info(fmt.Sprintf("Payment intent %d merchant mismatch", intentID))
		return false
	}

	// 2) Balances & fee berekenen
	fromBal, err := bp.chain.State.GetBalance(from)
	if err != nil {
		bp.logger.Error(fmt.Sprintf("GetBalance(from) error: %v", err))
		return false
	}

	if fromBal.Cmp(tx.Value) < 0 {
		bp.logger.Info("Payment TX rejected: insufficient balance")
		return false
	}

	// Fee = value * treasuryFeeBps / 10000
	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))

	merchantAmount := new(big.Int).Sub(tx.Value, fee)

	merchantBal, err := bp.chain.State.GetBalance(*tx.To)
	if err != nil {
		bp.logger.Error(fmt.Sprintf("GetBalance(merchant) error: %v", err))
		return false
	}

	treasuryBal, err := bp.chain.State.GetBalance(bp.chain.TreasuryAddr)
	if err != nil {
		bp.logger.Error(fmt.Sprintf("GetBalance(treasury) error: %v", err))
		return false
	}

	newFrom := new(big.Int).Sub(fromBal, tx.Value)
	newMerchant := new(big.Int).Add(merchantBal, merchantAmount)
	newTreasury := new(big.Int).Add(treasuryBal, fee)

	// 3) Balances wegschrijven
	if err := bp.chain.State.SetBalance(from, newFrom); err != nil {
		bp.logger.Error(fmt.Sprintf("SetBalance(from) error: %v", err))
		return false
	}
	if err := bp.chain.State.SetBalance(*tx.To, newMerchant); err != nil {
		bp.logger.Error(fmt.Sprintf("SetBalance(merchant) error: %v", err))
		return false
	}
	if err := bp.chain.State.SetBalance(bp.chain.TreasuryAddr, newTreasury); err != nil {
		bp.logger.Error(fmt.Sprintf("SetBalance(treasury) error: %v", err))
		return false
	}

	// 4) PaymentGateway updaten (on-chain settlement registratie)
	if err := bp.chain.Payment.MarkPaidFromTx(
		intentID,
		from,
		*tx.To,
		tx.Value,
		tx.Hash(),
		blockNum,
		blockTime,
	); err != nil {
		// Funds zijn al verplaatst, intent niet gemarkeerd -> in echte productie:
		// alert / compensating action. Voor nu loggen we.
		bp.logger.Info(fmt.Sprintf("MarkPaidFromTx failed for intent %d: %v", intentID, err))
	}

	bp.logger.Info(fmt.Sprintf(
		"Payment intent %d PAID via tx %s | gross=%s, fee=%s, net=%s",
		intentID,
		tx.Hash().Hex(),
		tx.Value.String(),
		fee.String(),
		merchantAmount.String(),
	))

	return true
}

// ----------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------

// parsePaymentIntentID verwacht tx.Data als ASCII "GORR_PAY:<id>"
// en geeft (id, true) terug als het matcht.
// Zo niet, dan (0, false).
func parsePaymentIntentID(data []byte) (uint64, bool) {
	if len(data) == 0 {
		return 0, false
	}
	if !bytes.HasPrefix(data, []byte(paymentDataPrefix)) {
		return 0, false
	}

	idStr := string(data[len(paymentDataPrefix):])
	if idStr == "" {
		return 0, false
	}

	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
