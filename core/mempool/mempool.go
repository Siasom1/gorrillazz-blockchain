package mempool

import (
	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

type Mempool struct {
	txs map[common.Hash]*types.Transaction
}

func New() *Mempool {
	return &Mempool{
		txs: make(map[common.Hash]*types.Transaction),
	}
}

func (m *Mempool) Add(tx *types.Transaction) {
	m.txs[tx.Hash] = tx
}
