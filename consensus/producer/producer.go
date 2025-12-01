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

	receipts := []*types.Receipt{}

	pending := bp.chain.TxPool.Pending()

	for _, tx := range pending {
		// Eventbus: tx in mempool
		if bp.events != nil {
			bp.events.PublishTx(tx)
		}

		from, err := tx.From()
		if err != nil {
			bp.logger.Error("Invalid tx signature: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		// Sender balance
		senderBal, err := bp.chain.State.GetBalance(from)
		if err != nil {
			bp.logger.Error("Failed to load sender balance: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		if senderBal.Cmp(tx.Value) < 0 {
			bp.logger.Error("TX rejected: insufficient funds")
			bp.chain.TxPool.Remove(tx)
			continue
		}

		// Update balances
		newSenderBal := new(big.Int).Sub(senderBal, tx.Value)
		if err := bp.chain.State.SetBalance(from, newSenderBal); err != nil {
			bp.logger.Error("Failed to update sender balance: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		receiverBal, err := bp.chain.State.GetBalance(*tx.To)
		if err != nil {
			bp.logger.Error("Failed to load receiver balance: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		newReceiverBal := new(big.Int).Add(receiverBal, tx.Value)
		if err := bp.chain.State.SetBalance(*tx.To, newReceiverBal); err != nil {
			bp.logger.Error("Failed to update receiver balance: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		// Nonce
		if err := bp.chain.State.IncreaseNonce(from); err != nil {
			bp.logger.Error("Failed to increase nonce: " + err.Error())
			bp.chain.TxPool.Remove(tx)
			continue
		}

		txIndex := uint64(len(newBlock.Transactions))

		receipt := &types.Receipt{
			TxHash:           tx.Hash(),
			BlockHash:        newBlock.Hash(),
			BlockNumber:      newBlock.Header.Number,
			TransactionIndex: txIndex,
			From:             from,
			To:               *tx.To,
			GasUsed:          tx.Gas,
			Status:           1,
		}

		if err := bp.chain.SaveTxIndex(tx.Hash(), newBlock.Header.Number); err != nil {
			bp.logger.Error("Failed to save tx index: " + err.Error())
		}

		newBlock.Transactions = append(newBlock.Transactions, tx)
		receipts = append(receipts, receipt)

		bp.chain.TxPool.Remove(tx)
	}

	// Save new head
	if err := bp.chain.SetHead(newBlock); err != nil {
		bp.logger.Error("Failed to save block: " + err.Error())
		return
	}

	// Save receipts
	if err := bp.chain.SaveReceipts(newBlock.Header.Number, receipts); err != nil {
		bp.logger.Error("Failed to save receipts: " + err.Error())
	}

	// Eventbus: new block
	if bp.events != nil {
		bp.events.PublishBlock(newBlock)
	}

	bp.logger.Info(
		fmt.Sprintf(
			"Produced block #%d | %d txs | Hash=%s",
			newBlock.Header.Number,
			len(newBlock.Transactions),
			newBlock.Hash().Hex(),
		),
	)
}
