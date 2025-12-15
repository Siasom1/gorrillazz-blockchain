package events

import "sync"

// EventBus is een simpele pub/sub bus
type EventBus struct {
	mu sync.RWMutex

	txSubs    []chan interface{}
	blockSubs []chan interface{}
}

// NewEventBus maakt een nieuwe bus
func NewEventBus() *EventBus {
	return &EventBus{
		txSubs:    make([]chan interface{}, 0),
		blockSubs: make([]chan interface{}, 0),
	}
}

// --------------------
// SUBSCRIBE
// --------------------

func (b *EventBus) SubscribeTxs() <-chan interface{} {
	ch := make(chan interface{}, 16)

	b.mu.Lock()
	b.txSubs = append(b.txSubs, ch)
	b.mu.Unlock()

	return ch
}

func (b *EventBus) SubscribeBlocks() <-chan interface{} {
	ch := make(chan interface{}, 16)

	b.mu.Lock()
	b.blockSubs = append(b.blockSubs, ch)
	b.mu.Unlock()

	return ch
}

// --------------------
// PUBLISH
// --------------------

func (b *EventBus) PublishTx(event interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.txSubs {
		select {
		case ch <- event:
		default:
			// drop if slow consumer
		}
	}
}

func (b *EventBus) PublishBlock(event interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, ch := range b.blockSubs {
		select {
		case ch <- event:
		default:
		}
	}
}
