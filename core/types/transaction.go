package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Transaction struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *common.Address
	Value    *big.Int
	Data     []byte

	// ECDSA signature (legacy tx)
	V *big.Int
	R *big.Int
	S *big.Int

	// Derived / internal
	From  common.Address
	Token TokenType
	Hash  common.Hash
}
