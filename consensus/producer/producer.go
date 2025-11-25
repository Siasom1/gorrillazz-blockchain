package producer

import (
	"fmt"
	"math/big"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/Siasom1/gorrillazz-chain/log"
)

type BlockProducer struct {
	chain    *blockchain.Blockchain
	logger   *log.Logger
	quit     chan struct{}
	interval time.Duration
	events   *events.EventBus
}

func NewBlockProducer(chain *blockchain.Blockchain, logger *log.Logger, blockTimeSeconds uint64, bus *events.EventBus) *BlockProducer {
	return &BlockProducer{
		chain:    chain,
		logger:   logger,
		quit:     make(chan struct{}),
		interval: time.Duration(blockTimeSeconds) * time.Second,
		events:   bus,
	}
}

func (bp *BlockProducer) Start() {
	bp.logger.Info("Starting block producer...")

	go func() {
		ticker := time.NewTicker(bp.interval)
		for {
			select {
			case <-ticker.C:
				bp.produceBlock()
			case <-bp.quit:
				ticker.Stop()
				return
			}
		}
	}()
}

func (bp *BlockProducer) Stop() {
	bp.logger.Info("Stopping block producer...")
	close(bp.quit)
}

// ---------------------------------------------------------
// Produce a new block
// ---------------------------------------------------------
func (bp *BlockProducer) produceBlock() {
	head := bp.chain.Head()

	newBlock := &types.Block{
		Header: &types.Header{
			ParentHash: head.Hash(),
			Number:     head.Header.Number + 1,
			Time:       uint64(time.Now().Unix()),
			StateRoot:  head.Header.StateRoot,
			TxRoot:     head.Header.TxRoot,
		},
		Transactions: []*types.Transaction{},
	}

	// A list of receipts for this block
	receipts := []*types.Receipt{}

	// ---------------------------------------------------------
	// 1. PROCESS PENDING TRANSACTIONS
	// ---------------------------------------------------------
	pending := bp.chain.TxPool.Pending()

	for _, tx := range pending {
		// Publish tx event (mempool)
		if bp.events != nil {
			bp.events.PublishTx(tx)
		}

		// Recover the sender
		from, err := tx.From()
		if err != nil {
			bp.logger.Error("Invalid tx signature: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		// Check sender balance
		senderBal, _ := bp.chain.State.GetBalance(from)
		if senderBal.Cmp(tx.Value) < 0 {
			bp.logger.Error("TX rejected: insufficient funds")
			bp.chain.TxPool.Remove(tx)
			continue
		}

		// Decrease sender balance
		newSenderBal := new(big.Int).Sub(senderBal, tx.Value)
		bp.chain.State.SetBalance(from, newSenderBal)

		// Increase receiver balance
		receiverBal, _ := bp.chain.State.GetBalance(*tx.To)
		newReceiverBal := new(big.Int).Add(receiverBal, tx.Value)
		bp.chain.State.SetBalance(*tx.To, newReceiverBal)

		// Increase nonce
		bp.chain.State.IncreaseNonce(from)

		// Transaction index inside the new block
		txIndex := uint64(len(newBlock.Transactions))

		// Build receipt
		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      newBlock.Header.Number,
			TransactionIndex: txIndex,
			From:             from,
			To:               *tx.To,
			GasUsed:          tx.Gas,
			Status:           1, // success
		}

		// Save TX → Block index
		if err := bp.chain.SaveTxIndex(tx.Hash(), newBlock.Header.Number); err != nil {
			bp.logger.Error("Failed to save tx index: " + err.Error())
		}

		// Add tx to block
		newBlock.Transactions = append(newBlock.Transactions, tx)

		// Add receipt
		receipts = append(receipts, receipt)

		// Remove from pool
		bp.chain.TxPool.Remove(tx)
	}

	// ---------------------------------------------------------
	// 2. SAVE NEW BLOCK
	// ---------------------------------------------------------
	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error("Failed to save block: " + err.Error())
		return
	}

	// ---------------------------------------------------------
	// 3. SAVE RECEIPTS FOR THIS BLOCK
	// ---------------------------------------------------------
	if err := bp.chain.SaveReceipts(newBlock.Header.Number, receipts); err != nil {
		bp.logger.Error("Failed to save receipts: " + err.Error())
	}

	// ---------------------------------------------------------
	// 4. PUBLISH BLOCK EVENT
	// ---------------------------------------------------------
	if bp.events != nil {
		bp.events.PublishBlock(newBlock)
	}

	// ---------------------------------------------------------
	// LOG
	// ---------------------------------------------------------
	bp.logger.Info(
		fmt.Sprintf(
			"Produced block #%d | %d txs | Hash=%s",
			newBlock.Header.Number,
			len(newBlock.Transactions),
			newBlock.Hash().Hex(),
		),
	)
}
