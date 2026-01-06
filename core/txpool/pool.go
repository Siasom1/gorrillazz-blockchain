package txpool

import (
	"sync"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

type TxPool struct {
	mu  sync.Mutex
	txs map[common.Hash]*types.Transaction
}

func NewTxPool() *TxPool {
	return &TxPool{
		txs: make(map[common.Hash]*types.Transaction),
	}
}

func (p *TxPool) Add(tx *types.Transaction) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// ✅ Hash is a FIELD, not a method
	if _, exists := p.txs[tx.Hash]; exists {
		return
	}

	p.txs[tx.Hash] = tx
}

func (p *TxPool) Remove(hash common.Hash) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.txs, hash)
}

func (p *TxPool) Pending() []*types.Transaction {
	p.mu.Lock()
	defer p.mu.Unlock()

	out := make([]*types.Transaction, 0, len(p.txs))
	for _, tx := range p.txs {
		out = append(out, tx)
	}
	return out
}
