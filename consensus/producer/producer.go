package producer

import (
	"bytes"
	"fmt"
	"go/token"
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


func calculateFee(tx *types.Transaction) *big.Int {
    if tx.GasPrice == nil {
        return big.NewInt(0)
    }
    fee := new(big.Int).Mul(tx.GasPrice, new(big.Int).SetUint64(tx.Gas))
    return fee
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
			StateRoot: common.Hash{},
			TxRoot: common.Hash{},
			Hash: comnmon.Hash{},
			
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
			from := tx.From
		}

		stateNonce, err := bp.chain.State.GetNonce(from)
		if err != {
			bp.logger.Error(fmt.Sprintf("GetNonce error for %s: %v", from.Hex(), err))
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

				// Increase nonce pas na succesvolle verwerking
		if err := bp.chain.State.IncNonce(from); err != nil {
			bp.logger.Error(fmt.Sprintf("IncNonce error: %v", err))
			continue
		}


		newBlock.Transactions = append(newBlock.Transactions, tx)

		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      blockNum,
			TransactionIndex: uint64(len(newBlock.Transactions) - 1),
			From:             from,
			To: 			  *tx. To,
			GasUsed: ,		   tx.Gas,
			Status: 		   1,
		}
		receipts = append(receipts, receipt)
		// if tx.To != nil {
		// 	receipt.To = *tx.To
		// }
		// receipt.Status = 1
		// receipts = append(receipts, receipt)

		bp.chain.TxPool.Remove(tx.Hash)
	}

	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error(fmt.Sprintf("SetHead error: %v", err))
	}
	if err := bp.chain.SaveReceipts(newBlock.Header.Number, receipts); err != nil {
		bp.logger.Error(fmt.Sprintf("SaveReceipts error: %v", err))
	}

	bp.logger.Info(fmt.Sprintf("Produced block #%d | %d txs | Hash=%s", newBlock.Header.Number, len(newBlock.Transactions), newBlock.Hash().Hex()))
}

func (bp *BlockProducer) processTokenTransfer(tx *types.Transaction, from common.Address) bool {
	if tx.To == nil {
		bp.logger.Info("TX rejected: nil To address")
		return false
	}

	token := tx.Token
	var fromBal, toBal *big.Int
	var err error

	switch token {
	case types.TokenGORR:
	fromBal, err := bp.chain.State.GetBalance(from)
	if err != nil {
	bp.logger.Error(fmt.Sprintf("GetBalance(from) error: %v", err))
		return false
	}
	toBal, err := bp.chain.State.GetBalance(*tx.To)
	if err != nil {
	bp.logger.Error(fmt.Sprintf("GetBalance(to) error: %v", err))
		return false
	}

	case types.TokenUSDCc:
		fromBal, err = bp.chain.State.GetUSDCcBalance(from)
		if err != nil {
			bp.logger.Error(fmt.Sprintf("GetUSDCcBalance(from) error: %v", err))
			return false
		}
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}

	toBal, err = bp.chain.State.GetUSDCcBalance(*tx.To)
		if err != nil {
			bp.logger.Error(fmt.Sprintf("GetUSDCcBalance(to) error: %v", err))
			return false
		}
	default:
		bp.logger.Info("TX rejected: unknown token")
		return false
	}

	if fromBal.Cmp(tx.Value) < 0 {
		bp.logger.Info("TX rejected: insufficient balance")
		return false
	}

	// Bereken fee
	fee := new(big.Int).Mul(tx.Value, big.NewInt(treasuryFeeBps))
	fee.Div(fee, big.NewInt(bpsDenominator))
	transferAmount := new(big.Int).Sub(tx.Value, fee)

	// Update balances
	switch token {
	case types.TokenGORR:
		bp.chain.State.SetBalance(from, new(big.Int).Sub(fromBal, tx.Value))
		bp.chain.State.SetBalance(*tx.To, new(big.Int).Add(toBal, transferAmount))
	case types.TokenUSDCc:
		bp.chain.State.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, tx.Value))
		bp.chain.State.SetUSDCcBalance(*tx.To, new(big.Int).Add(toBal, transferAmount))
	}

	// Treasury update
	if fee.Sign() > 0 && bp.chain.TreasuryAddr != (common.Address{}) {
		if token == types.TokenGORR {
			treasuryBal, _ := bp.chain.State.GetBalance(bp.chain.TreasuryAddr)
			bp.chain.State.SetBalance(bp.chain.TreasuryAddr, new(big.Int).Add(treasuryBal, fee))
		} else if token == types.TokenUSDCc {
			treasuryBal, _ := bp.chain.State.GetUSDCcBalance(bp.chain.TreasuryAddr)
			bp.chain.State.SetUSDCcBalance(bp.chain.TreasuryAddr, new(big.Int).Add(treasuryBal, fee))
		}
	}

	bp.logger.Info(fmt.Sprintf("%s transfer from %s to %s | value=%s | fee=%s", token, from.Hex(), tx.To.Hex(), tx.Value.String(), fee.String()))
	return true
}

func (bp *BlockProducer) applyTransaction(tx *types.Transaction, st *state.State) error {
    // --- bestaande tx processing ---
    err := bp.chain.ApplyTransaction(tx)
    if err != nil {
        return err
    }

    // --- fee settlement (bestaand) ---
    fee := calculateFee(tx)
    if fee.Sign() > 0 {
        st.AddCollectedFee("GORR", fee)
    }

    // --- Q4A: admin padding refill ---
    adminAddr := bp.chain.AdminAddress
    bal, err := st.GetBalance(adminAddr)
    if err != nil {
        return err
    }

    if bal.Sign() < 0 {
        // refill amount to PAD
        pad := new(big.Int).SetUint64(10_000_000_000) // 10B GORR
        refill := new(big.Int).Sub(pad, bal)          // pad - (-bal)

        // mint from treasury (infinite)
        st.AddSupply("GORR", refill)

        // credit admin
        newBal := new(big.Int).Add(bal, refill)
        _ = st.SetBalance(adminAddr, newBal)
    }

    return nil
}


// -------------------------
// Payment GORR transfer (met fee, voorlopig optioneel)
// -------------------------

func (bp *BlockProducer) processPaymentGORR(tx *types.Transaction, from common.Address, intentID, blockNum, blockTime uint64) bool {
	// Voor nu nog niet actief, kan later volwaardig worden geïmplementeerd
	return false
}

// -------------------------
// Helpers
// -------------------------

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

func (b *Block) Hash() common.Hash {
	if b == nil || b.Header == nil {
		return common.Hash{}
	}
	return b.Header.Hash()
}

func (bp *BlockProducer) produce() {
	head := bp.chain.Head()
	if head == nil {
		bp.logger.Error("No head block loaded")
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

	txns := bp.chain.TxPool.Pending()
	receipts := []*types.Receipt{}

	for _, tx := range txns {
		if tx == nil {
			continue
		}

		from := tx.From // geen tx.From() meer

		// nonce
		stateNonce, err := bp.chain.State.GetNonce(from)
		if err != nil || tx.Nonce != stateNonce {
			continue
		}

		ok := false
		switch tx.Token {
		case types.TokenGORR:
			ok = bp.processGORR(tx, from, newBlock)
		case types.TokenUSDCc:
			ok = bp.processUSDCc(tx, from, newBlock)
		default:
			bp.logger.Info(fmt.Sprintf("Unknown token %s", tx.Token))
		}
		if !ok {
			continue
		}

		// nonce++
		_ = bp.chain.State.IncNonce(from)

		newBlock.Transactions = append(newBlock.Transactions, tx)

		receipts = append(receipts, &types.Receipt{
			TxHash:           tx.Hash,
			BlockHash:        newBlock.Hash(),
			BlockNumber:      newBlock.Header.Number,
			TransactionIndex: uint64(len(newBlock.Transactions) - 1),
			From:             from,
			To:               *tx.To,
			Status:           1,
		})

		// remove via hash
		bp.chain.TxPool.Remove(tx.Hash)
	}

	_ = bp.chain.SetHead(newBlock)
	_ = bp.chain.SaveReceipts(newBlock.Header.Number, receipts)

	bp.logger.Info(
		fmt.Sprintf(
			"Produced block #%d | %d txs | Hash=%s",
			newBlock.Header.Number,
			len(newBlock.Transactions),
			newBlock.Hash().Hex(),
		),

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

	return true
}


func (bp *BlockProducer) processUSDCc(tx *types.Transaction, from common.Address, b *types.Block) bool {
	if tx.To == nil {
		return false
	}

	fromBal, _ := bp.chain.State.GetUSDCcBalance(from)
	if fromBal.Cmp(tx.Value) < 0 {
		return false
	}

	fee := new(big.Int).Mul(tx.Value, big.NewInt(250))
	fee.Div(fee, big.NewInt(10000))
	net := new(big.Int).Sub(tx.Value, fee)

	toBal, _ := bp.chain.State.GetUSDCcBalance(*tx.To)
	treasuryBal, _ := bp.chain.State.GetUSDCcBalance(bp.chain.TreasuryAddr)

	bp.chain.State.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, tx.Value))
	bp.chain.State.SetUSDCcBalance(*tx.To, new(big.Int).Add(toBal, net))
	bp.chain.State.SetUSDCcBalance(bp.chain.TreasuryAddr, new(big.Int).Add(treasuryBal, fee))

	return true
}

	)
}
