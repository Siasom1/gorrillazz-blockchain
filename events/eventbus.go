package events

import "sync"

type Callback func(event interface{})

type EventBus struct {
	mu   sync.RWMutex
	subs map[string][]Callback
}

func NewEventBus() *EventBus {
	return &EventBus{
		subs: make(map[string][]Callback),
	}
}

func (b *EventBus) Subscribe(event string, cb Callback) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subs[event] = append(b.subs[event], cb)
}

func (b *EventBus) Emit(event string, payload interface{}) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if cbs, ok := b.subs[event]; ok {
		for _, cb := range cbs {
			go cb(payload)
		}
	}
}
