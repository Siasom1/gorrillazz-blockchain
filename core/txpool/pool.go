package txpool

import (
	"errors"
	"sync"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

var (
	ErrPoolFull  = errors.New("txpool: pool full")
	ErrTxExists  = errors.New("txpool: tx already exists")
	ErrInvalidTx = errors.New("txpool: invalid transaction")
)

type TxPool struct {
	mu      sync.RWMutex
	pending map[string]*types.Transaction
	maxSize int
	timeout time.Duration
}

func New(maxSize int, timeout time.Duration) *TxPool {
	return &TxPool{
		pending: make(map[string]*types.Transaction),
		maxSize: maxSize,
		timeout: timeout,
	}
}

func (p *TxPool) Add(tx *types.Transaction) error {
	if tx == nil {
		return ErrInvalidTx
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(p.pending) >= p.maxSize {
		return ErrPoolFull
	}

	// HASH FIX → always convert to string for map key
	h := tx.Hash().String()

	if _, exists := p.pending[h]; exists {
		return ErrTxExists
	}

	p.pending[h] = tx
	return nil
}

func (p *TxPool) Remove(tx *types.Transaction) {
	if tx == nil {
		return
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.pending, tx.Hash().String())
}

func (p *TxPool) Pending() []*types.Transaction {
	p.mu.RLock()
	defer p.mu.RUnlock()

	out := make([]*types.Transaction, 0, len(p.pending))
	for _, tx := range p.pending {
		out = append(out, tx)
	}
	return out
}

func (p *TxPool) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.pending = make(map[string]*types.Transaction)
}
