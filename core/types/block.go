package types

import "github.com/ethereum/go-ethereum/common"

type Block struct {
	Header       *Header
	Transactions []*Transaction
}

type Header struct {
	Number     uint64
	Time       uint64
	ParentHash common.Hash
	StateRoot  common.Hash
	TxRoot     common.Hash
	Hash       common.Hash
}
