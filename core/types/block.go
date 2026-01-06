package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Block struct {
	Header *Header
	Txs    []*Transaction
}

type Header struct {
	Number uint64
	Time   uint64
	Hash   common.Hash
	Parent common.Hash
}

// Compute deterministic block hash
func (b *Block) ComputeHash() common.Hash {
	data, _ := json.Marshal(b)
	return crypto.Keccak256Hash(data)
}
