package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

type legacyTx struct {
	Nonce    uint64
	GasPrice *big.Int
	Gas      uint64
	To       *[20]byte
	Value    *big.Int
	Data     []byte
	V, R, S  *big.Int
}

func DecodeLegacyTx(raw []byte) (*Transaction, error) {
	var lt legacyTx
	if err := rlp.DecodeBytes(raw, &lt); err != nil {
		return nil, err
	}

	var toAddr *common.Address
	if lt.To != nil {
		addr := common.BytesToAddress(lt.To[:])
		toAddr = &addr
	}

	return &Transaction{
		To:       toAddr,
		Value:    lt.Value,
		Data:     lt.Data,
		Nonce:    lt.Nonce,
		Gas:      lt.Gas,
		GasPrice: lt.GasPrice,
		Token:    TokenGORR, // 🔥 default native coin
		V:        lt.V,
		R:        lt.R,
		S:        lt.S,
	}, nil
}
