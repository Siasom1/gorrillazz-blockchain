package events

import (
	"sync"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

type EventBus struct {
	mu        sync.RWMutex
	blockSubs []chan *types.Block
	txSubs    []chan *types.Transaction
}

func NewEventBus() *EventBus {
	return &EventBus{
		blockSubs: make([]chan *types.Block, 0),
		txSubs:    make([]chan *types.Transaction, 0),
	}
}

// -------------------- Blocks --------------------

func (b *EventBus) SubscribeBlocks() <-chan *types.Block {
	ch := make(chan *types.Block, 16)

	b.mu.Lock()
	b.blockSubs = append(b.blockSubs, ch)
	b.mu.Unlock()

	return ch
}

func (b *EventBus) PublishBlock(block *types.Block) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.blockSubs {
		// non-blocking send
		select {
		case ch <- block:
		default:
		}
	}
}

// -------------------- Transactions --------------------

func (b *EventBus) SubscribeTxs() <-chan *types.Transaction {
	ch := make(chan *types.Transaction, 64)

	b.mu.Lock()
	b.txSubs = append(b.txSubs, ch)
	b.mu.Unlock()

	return ch
}

func (b *EventBus) PublishTx(tx *types.Transaction) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.txSubs {
		select {
		case ch <- tx:
		default:
		}
	}
}
