package txpool

import (
	"sync"

	"github.com/Siasom1/gorrillazz-chain/common"
	"github.com/Siasom1/gorrillazz-chain/core/types"
)

type TxPool struct {
	mu   sync.RWMutex
	pool map[common.Hash]*types.Transaction
}

func NewTxPool() *TxPool {
	return &TxPool{
		pool: make(map[common.Hash]*types.Transaction),
	}
}

func (tp *TxPool) Add(tx *types.Transaction) {
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// !!! Belangrijk: Hash() is functie, niet field
	h := tx.Hash()
	tp.pool[h] = tx
}

func (tp *TxPool) Get(hash common.Hash) (*types.Transaction, bool) {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	tx, ok := tp.pool[hash]
	return tx, ok
}

func (tp *TxPool) Pending() []*types.Transaction {
	tp.mu.RLock()
	defer tp.mu.RUnlock()

	pending := make([]*types.Transaction, 0, len(tp.pool))
	for _, tx := range tp.pool {
		pending = append(pending, tx)
	}
	return pending
}

func (tp *TxPool) Remove(hash common.Hash) {
	tp.mu.Lock()
	defer tp.mu.Unlock()
	delete(tp.pool, hash)
}

func (tp *TxPool) Count() int {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return len(tp.pool)
}
