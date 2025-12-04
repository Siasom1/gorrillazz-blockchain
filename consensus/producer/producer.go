package producer

import (
	"math/big"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
	"github.com/ethereum/go-ethereum/common"
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

	newBlock := &types.Block{
		Header: &types.Header{
			ParentHash: head.Hash(),
			Number:     head.Header.Number + 1,
			Time:       uint64(time.Now().Unix()),
			StateRoot:  common.Hash{},
			TxRoot:     common.Hash{},
		},
	}

	txns := bp.chain.TxPool.Pending()
	receipts := []*types.Receipt{}

	for _, tx := range txns {
		from, err := tx.From()
		if err != nil {
			bp.logger.Error("Invalid TX signature")
			continue
		}

		// NONCE check
		stateNonce, _ := bp.chain.State.GetNonce(from)
		if tx.Nonce != stateNonce {
			continue
		}

		// PROCESS ASSET TRANSFER
		switch tx.Asset {
		case "GORR":
			bp.processGORR(tx, from)
		case "USDCc":
			bp.processUSDCc(tx, from)
		default:
			bp.logger.Error("Unknown asset: " + tx.Asset)
			continue
		}

		// Increase nonce
		bp.chain.State.IncreaseNonce(from)

		// Add to block
		newBlock.Transactions = append(newBlock.Transactions, tx)

		// Receipt
		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      newBlock.Header.Number,
			TransactionIndex: uint64(len(newBlock.Transactions) - 1),
			From:             from,
			To:               *tx.To,
			GasUsed:          tx.Gas,
			Status:           1,
		}
		receipts = append(receipts, receipt)

		// Remove from pool
		bp.chain.TxPool.Remove(tx)
	}

	bp.chain.SetHead(newBlock)
	bp.chain.SaveReceipts(newBlock.Header.Number, receipts)
}

func (bp *BlockProducer) processGORR(tx *types.Transaction, from common.Address) {
	fromBal, _ := bp.chain.State.GetBalance(from)
	if fromBal.Cmp(tx.Amount) < 0 {
		return
	}
	toBal, _ := bp.chain.State.GetBalance(*tx.To)

	bp.chain.State.SetBalance(from, new(big.Int).Sub(fromBal, tx.Amount))
	bp.chain.State.SetBalance(*tx.To, new(big.Int).Add(toBal, tx.Amount))
}

func (bp *BlockProducer) processUSDCc(tx *types.Transaction, from common.Address) {
	fromBal, _ := bp.chain.State.GetUSDCcBalance(from)
	if fromBal.Cmp(tx.Amount) < 0 {
		return
	}
	toBal, _ := bp.chain.State.GetUSDCcBalance(*tx.To)

	bp.chain.State.SetUSDCcBalance(from, new(big.Int).Sub(fromBal, tx.Amount))
	bp.chain.State.SetUSDCcBalance(*tx.To, new(big.Int).Add(toBal, tx.Amount))
}
