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
	treasuryFeeBps    = 250 // 2.5%
	bpsDenominator    = 10000
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
			if bp.chain.AdminAddr == (common.Address{}) {
				continue
			}
			from = bp.chain.AdminAddr
		}

		stateNonce, err := bp.chain.State.GetNonce(from)
		if err != nil || tx.Nonce != stateNonce {
			continue
		}

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

		if err := bp.chain.State.IncreaseNonce(from); err != nil {
			bp.logger.Error(fmt.Sprintf("IncreaseNonce error: %v", err))
			continue
		}

		newBlock.Transactions = append(newBlock.Transactions, tx)

		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      blockNum,
			TransactionIndex: uint64(len(newBlock.Transactions) - 1),
			From:             from,
		}
		if tx.To != nil {
			receipt.To = *tx.To
		}
		receipt.Status = 1
		receipts = append(receipts, receipt)

		bp.chain.TxPool.Remove(tx)
	}

	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error(fmt.Sprintf("SetHead error: %v", err))
	}
	if err := bp.chain.SaveReceipts(newBlock.Header.Number, receipts); err != nil {
		bp.logger.Error(fmt.Sprintf("SaveReceipts error: %v", err))
	}

	bp.logger.Info(fmt.Sprintf("Produced block #%d | %d txs | Hash=%s", newBlock.Header.Number, len(newBlock.Transactions), newBlock.Hash().Hex()))
}

func (bp *BlockProducer) processGORR(tx *types.Transaction, from common.Address) bool {
	if tx.To == nil {
		return false
	}

	fromBal, err := bp.chain.State.GetBalance(from)
	if err != nil {
		return false
	}
	toBal, err := bp.chain.State.GetBalance(*tx.To)
	if err != nil {
		return false
	}
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}

	bp.chain.State.SetBalance(from, new(big.Int).Sub(fromBal, tx.Value))
	bp.chain.State.SetBalance(*tx.To, new(big.Int).Add(toBal, tx.Value))
	return true
}

func (bp *BlockProducer) processPaymentGORR(tx *types.Transaction, from common.Address, intentID, blockNum, blockTime uint64) bool {
	if tx.To == nil || bp.chain.Payment == nil || bp.chain.TreasuryAddr == (common.Address{}) {
		return false
	}

	intent, err := bp.chain.Payment.GetIntent(intentID)
	if err != nil {
		return false
	}
	if intent.Token != "GORR" || intent.Merchant != *tx.To {
		return false
	}

	fromBal, _ := bp.chain.State.GetBalance(from)
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}

	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))
	merchantAmount := new(big.Int).Sub(tx.Value, fee)

	merchantBal, _ := bp.chain.State.GetBalance(*tx.To)
	treasuryBal, _ := bp.chain.State.GetBalance(bp.chain.TreasuryAddr)

	bp.chain.State.SetBalance(from, new(big.Int).Sub(fromBal, tx.Value))
	bp.chain.State.SetBalance(*tx.To, new(big.Int).Add(merchantBal, merchantAmount))
	bp.chain.State.SetBalance(bp.chain.TreasuryAddr, new(big.Int).Add(treasuryBal, fee))

	bp.chain.Payment.MarkPaidFromTx(intentID, from, *tx.To, tx.Value, tx.Hash(), blockNum, blockTime)
	return true
}

func parsePaymentIntentID(data []byte) (uint64, bool) {
	if len(data) == 0 || !bytes.HasPrefix(data, []byte(paymentDataPrefix)) {
		return 0, false
	}
	idStr := string(data[len(paymentDataPrefix):])
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		return 0, false
	}
	return id, true
}
