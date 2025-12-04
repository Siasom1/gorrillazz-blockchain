package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type rlpTx struct {
	Nonce    uint64
	To       *common.Address
	Value    *big.Int
	Asset    string
	Amount   *big.Int
	Gas      uint64
	GasPrice *big.Int
	V        *big.Int
	R        *big.Int
	S        *big.Int
}

func (tx *Transaction) Serialize() []byte {
	obj := rlpTx{
		Nonce:    tx.Nonce,
		To:       tx.To,
		Value:    tx.Value,
		Asset:    tx.Asset,
		Amount:   tx.Amount,
		Gas:      tx.Gas,
		GasPrice: tx.GasPrice,
		V:        tx.V,
		R:        tx.R,
		S:        tx.S,
	}
	bytes, _ := rlp.EncodeToBytes(obj)
	return bytes
}

func DecodeTx(data []byte) (*Transaction, error) {
	var rt rlpTx
	err := rlp.DecodeBytes(data, &rt)
	if err != nil {
		return nil, err
	}

	return &Transaction{
		Nonce:    rt.Nonce,
		To:       rt.To,
		Value:    rt.Value,
		Asset:    rt.Asset,
		Amount:   rt.Amount,
		Gas:      rt.Gas,
		GasPrice: rt.GasPrice,
		V:        rt.V,
		R:        rt.R,
		S:        rt.S,
	}, nil
}
