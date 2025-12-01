// core/types/genesis.go
package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

// NewGenesisBlock maakt block #0 met lege roots en geen transacties.
// Alloc van balances doen we in de State (LevelDB), niet in dit block zelf.
func NewGenesisBlock() *Block {
	header := &Header{
		ParentHash: common.Hash{}, // geen parent
		Number:     0,             // genesis block
		Time:       uint64(time.Now().Unix()),
		StateRoot:  common.Hash{}, // voorlopig 0, echte root later
		TxRoot:     common.Hash{}, // geen tx's
	}

	return &Block{
		Header:       header,
		Transactions: []*Transaction{},
	}
}
