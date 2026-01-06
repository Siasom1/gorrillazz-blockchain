package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	// "github.com/ethereum/go-ethereum/crypto"
)

type TokenType string

const (
	TokenGORR  TokenType = "GORR"
	TokenUSDCc TokenType = "USDCc"
)

type Transaction struct {
	From     common.Address
	To       *common.Address
	Value    *big.Int
	Data     []byte
	Nonce    uint64
	Gas      uint64
	GasPrice *big.Int
	Token    TokenType

	V, R, S *big.Int
	HashVal common.Hash
}

// // Compute tx hash (cached)
// func (tx *Transaction) Hash() common.Hash {
// 	if tx.HashVal == (common.Hash{}) {
// 		data := []byte{}
// 		if tx.To != nil {
// 			data = append(data, tx.To.Bytes()...)
// 		}
// 		if tx.Value != nil {
// 			data = append(data, tx.Value.Bytes()...)
// 		}
// 		tx.HashVal = crypto.Keccak256Hash(data)
// 	}
// 	return tx.HashVal
// }
