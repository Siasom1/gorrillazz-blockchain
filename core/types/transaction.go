package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type Transaction struct {
	Nonce    uint64
	To       *common.Address
	Value    *big.Int
	Gas      uint64
	GasPrice *big.Int
	Data     []byte
	V, R, S  *big.Int

	Sender common.Address
}

func (tx *Transaction) Hash() common.Hash {
	return crypto.Keccak256Hash(tx.Serialize())
}
