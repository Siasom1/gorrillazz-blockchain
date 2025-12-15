package events

import "sync"

type EventBus struct {
	mu sync.RWMutex

	Blocks   chan interface{}
	Txs      chan interface{}
	Payments chan interface{}
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
