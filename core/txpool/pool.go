package txpool

import (
	"errors"
	"sync"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

type TxPool struct {
	mu      sync.RWMutex
	pending []*types.Transaction
}

func NewTxPool() *TxPool {
	return &TxPool{
		pending: []*types.Transaction{},
	}
}

func (p *TxPool) Add(tx *types.Transaction) error {
	if tx == nil {
		return errors.New("nil tx")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = append(p.pending, tx)
	return nil
}

func (p *TxPool) Pending() []*types.Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	list := make([]*types.Transaction, len(p.pending))
	copy(list, p.pending)
	return list
}

func (p *TxPool) Remove(tx *types.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()

	newList := []*types.Transaction{}
	for _, t := range p.pending {
		if t.Hash() != tx.Hash() {
			newList = append(newList, t)
		}
	}
	p.pending = newList
}
