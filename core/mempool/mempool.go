package mempool

import (
	"fmt"
	"sync"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

type Mempool struct {
	mu     sync.Mutex
	pool   map[common.Hash]*types.Transaction
	nonces map[common.Address]uint64
}

func NewMempool() *Mempool {
	return &Mempool{
		pool:   make(map[common.Hash]*types.Transaction),
		nonces: make(map[common.Address]uint64),
	}
}

func (m *Mempool) NextNonce(addr common.Address) uint64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nonces[addr]
}

func (m *Mempool) Add(tx *types.Transaction) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	expected := m.nonces[tx.From]
	if tx.Nonce != expected {
		return fmt.Errorf("bad nonce: expected %d got %d", expected, tx.Nonce)
	}

	m.pool[tx.Hash] = tx
	m.nonces[tx.From]++

	return nil
}
