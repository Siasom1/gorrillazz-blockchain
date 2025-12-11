package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type rlpTx struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *common.Address // NOTE: correct position
	Value    *big.Int
	Data     []byte
	V        *big.Int
	R        *big.Int
	S        *big.Int
}

// Serialize encodes the tx in exact Ethereum RLP format
func (tx *Transaction) Serialize() []byte {
	obj := rlpTx{
		Nonce:    tx.Nonce,
		GasPrice: tx.GasPrice,
		Gas:      tx.Gas,
		To:       tx.To,
		Value:    tx.Value,
		Data:     tx.Data,
		V:        tx.V,
		R:        tx.R,
		S:        tx.S,
	}

	out, _ := rlp.EncodeToBytes(obj)
	return out
}

// DecodeTx decodes Ethereum-style RLP into our Transaction struct
func DecodeTx(data []byte) (*Transaction, error) {
	var decoded rlpTx
	err := rlp.DecodeBytes(data, &decoded)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Nonce:    decoded.Nonce,
		GasPrice: decoded.GasPrice,
		Gas:      decoded.Gas,
		To:       decoded.To,
		Value:    decoded.Value,
		Data:     decoded.Data,
		V:        decoded.V,
		R:        decoded.R,
		S:        decoded.S,
	}, nil
}
