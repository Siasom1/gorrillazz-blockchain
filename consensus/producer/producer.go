package producer

import (
	"bytes"
	"math/big"
	"strconv"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/Siasom1/gorrillazz-chain/node/payment_gateway"
	"github.com/ethereum/go-ethereum/common"
)

const (
	paymentDataPrefix = "GORR_PAY:"
	treasuryFeeBps    = 250
	bpsDenominator    = 10000
)

type BlockProducer struct {
	chain  *blockchain.Blockchain
	logger *log.Logger
	quit   chan struct{}
	delay  time.Duration
	bus    *events.EventBus
	paygw  *payment_gateway.PaymentGateway
}

func NewBlockProducer(chain *blockchain.Blockchain, logger *log.Logger, blockTime uint64, bus *events.EventBus, paygw *payment_gateway.PaymentGateway) *BlockProducer {
	return &BlockProducer{
		chain:  chain,
		logger: logger,
		quit:   make(chan struct{}),
		delay:  time.Duration(blockTime) * time.Second,
		bus:    bus,
		paygw:  paygw,
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

// ----------------------
// Process GORR
// ----------------------
func (bp *BlockProducer) processGORR(tx *types.Transaction, from common.Address, b *types.Block) bool {
	if tx.To == nil {
		return false
	}
	fromBal, _ := bp.chain.State.GetBalance(from)
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}
	toBal, _ := bp.chain.State.GetBalance(*tx.To)
	bp.chain.State.SetBalance(from, new(big.Int).Sub(fromBal, tx.Value))
	bp.chain.State.SetBalance(*tx.To, new(big.Int).Add(toBal, tx.Value))

	// Treasury fee
	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))
	if fee.Sign() > 0 {
		tBal, _ := bp.chain.State.GetBalance(bp.chain.TreasuryAddr)
		bp.chain.State.SetBalance(bp.chain.TreasuryAddr, new(big.Int).Add(tBal, fee))
	}

	// Payment Gateway integration
	if id, ok := parsePaymentIntentID(tx.Data); ok {
		_ = bp.paygw.MarkPaidFromTx(id, from, *tx.To, tx.Value, tx.Hash, b.Header.Number, b.Header.Time)
	}

	return true
}

// ----------------------
// Process USDCc
// ----------------------
func (bp *BlockProducer) processUSDCc(tx *types.Transaction, from common.Address, b *types.Block) bool {
	if tx.To == nil {
		return false
	}
	fromBal, _ := bp.chain.State.GetUSDCcBalance(from)
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}

	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))
	net := new(big.Int).Sub(tx.Value, fee)

	toBal, _ := bp.chain.State.GetUSDCcBalance(*tx.To)
	treasuryBal, _ := bp.chain.State.GetUSDCcBalance(bp.chain.TreasuryAddr)

	bp.chain.State.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, tx.Value))
	bp.chain.State.SetUSDCcBalance(*tx.To, new(big.Int).Add(toBal, net))
	bp.chain.State.SetUSDCcBalance(bp.chain.TreasuryAddr, new(big.Int).Add(treasuryBal, fee))

	// Payment Gateway integration
	if id, ok := parsePaymentIntentID(tx.Data); ok {
		_ = bp.paygw.MarkPaidFromTx(id, from, *tx.To, tx.Value, tx.Hash, b.Header.Number, b.Header.Time)
	}

	return true
}

// ----------------------
// Payment Intent
// ----------------------
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
