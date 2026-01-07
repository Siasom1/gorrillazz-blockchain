package events

import (
	"sync"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

type EventBus struct {
	mu sync.RWMutex

	Blocks   chan interface{}
	Txs      chan interface{}
	Payments chan interface{}

	blockSubscribers []chan *types.Block
	txSubscribers    []chan *types.Transaction
}

func NewEventBus() *EventBus {
	return &EventBus{
		Blocks:   make(chan interface{}, 100),
		Txs:      make(chan interface{}, 100),
		Payments: make(chan interface{}, 100),
	}
}

// ---------- EMIT HELPERS ----------

func (b *EventBus) EmitBlock(block interface{}) {
	select {
	case b.Blocks <- block:
	default:
	}
}

func (b *EventBus) EmitTx(tx interface{}) {
	select {
	case b.Txs <- tx:
	default:
	}
}

func (b *EventBus) EmitPayment(p interface{}) {
	select {
	case b.Payments <- p:
	default:
	}
}

// ---------- SUBSCRIBE HELPERS ----------

func (bus *EventBus) SubscribeBlocks(ch chan *types.Block) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.blockSubscribers = append(bus.blockSubscribers, ch)
}

func (bus *EventBus) SubscribeTxs(ch chan *types.Transaction) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	bus.txSubscribers = append(bus.txSubscribers, ch)
}
