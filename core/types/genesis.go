package types

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
)

func DefaultGenesis() *Block {
	h := &Header{
		Number:     0,
		Time:       uint64(time.Now().Unix()),
		ParentHash: common.Hash{},
		StateRoot:  common.Hash{},
		TxRoot:     common.Hash{},
	}

	b := &Block{
		Header:       h,
		Transactions: []*Transaction{},
	}

	b.Header.Hash = b.ComputeHash()
	return b
}
