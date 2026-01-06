package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func (tx *Transaction) Hash() common.Hash {
	b, _ := json.Marshal(tx)
	return crypto.Keccak256Hash(b)
}
