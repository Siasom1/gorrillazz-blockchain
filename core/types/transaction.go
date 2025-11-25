package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Ethereum-like transaction
type Transaction struct {
	Nonce    uint64          `json:"nonce"`
	To       *common.Address `json:"to"`
	Value    *big.Int        `json:"value"`
	Gas      uint64          `json:"gas"`
	GasPrice *big.Int        `json:"gasPrice"`
	Data     []byte          `json:"data"`

	// Signature parts
	V *big.Int `json:"v"`
	R *big.Int `json:"r"`
	S *big.Int `json:"s"`

	// Cached sender
	from *common.Address
}
