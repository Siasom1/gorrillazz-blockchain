package types

import (
	"bytes"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// Used for RLP encoding/decoding
type txdata struct {
	Nonce    uint64
	To       *common.Address
	Value    *big.Int
	Gas      uint64
	GasPrice *big.Int
	Data     []byte
	V        *big.Int
	R        *big.Int
	S        *big.Int
}

func (tx *Transaction) EncodeRLP() ([]byte, error) {
	data := txdata{
		Nonce:    tx.Nonce,
		To:       tx.To,
		Value:    tx.Value,
		Gas:      tx.Gas,
		GasPrice: tx.GasPrice,
		Data:     tx.Data,
		V:        tx.V,
		R:        tx.R,
		S:        tx.S,
	}
	return rlp.EncodeToBytes(data)
}

func DecodeTx(b []byte) (*Transaction, error) {
	var data txdata
	err := rlp.Decode(bytes.NewReader(b), &data)
	if err != nil {
		return nil, err
	}
	return &Transaction{
		Nonce:    data.Nonce,
		To:       data.To,
		Value:    data.Value,
		Gas:      data.Gas,
		GasPrice: data.GasPrice,
		Data:     data.Data,
		V:        data.V,
		R:        data.R,
		S:        data.S,
	}, nil
}

func (tx *Transaction) IsSigned() bool {
	return tx.V != nil && tx.R != nil && tx.S != nil
}

func (tx *Transaction) Validate() error {
	if tx.Gas == 0 {
		return errors.New("gas cannot be 0")
	}
	if tx.GasPrice == nil {
		return errors.New("gasPrice missing")
	}
	return nil
}
