package types

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ComputeTxHash(raw []byte) common.Hash {
	return crypto.Keccak256Hash(raw)
}
