package types

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Transaction represents a native multi-asset transaction
type Transaction struct {
	Nonce    uint64          `json:"nonce"`
	To       *common.Address `json:"to"`
	Value    *big.Int        `json:"value"`  // GORR transfers
	Asset    string          `json:"asset"`  // "GORR" or "USDCc"
	Amount   *big.Int        `json:"amount"` // multi-asset amount
	Gas      uint64          `json:"gas"`
	GasPrice *big.Int        `json:"gasPrice"`
	V, R, S  *big.Int        `json:"vrs"`
}

// Validate basic fields
func (tx *Transaction) Validate() error {
	if tx.To == nil {
		return errors.New("missing 'to' address")
	}
	if tx.Asset != "GORR" && tx.Asset != "USDCc" {
		return errors.New("unknown asset")
	}
	if tx.Amount == nil {
		return errors.New("missing amount")
	}
	if tx.Amount.Sign() < 0 {
		return errors.New("amount negative")
	}
	return nil
}

// Hash computes the transaction hash
func (tx *Transaction) Hash() common.Hash {
	enc := tx.Serialize()
	return crypto.Keccak256Hash(enc)
}
