package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (b *Block) ComputeHash() common.Hash {
	data, _ := json.Marshal(b.Header)
	return crypto.Keccak256Hash(data)
}
