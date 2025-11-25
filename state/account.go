package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type Account struct {
	Address common.Address `json:"address"`
	Balance *big.Int       `json:"balance"`
	Nonce   uint64         `json:"nonce"`
}

func NewAccount(addr common.Address) *Account {
	return &Account{
		Address: addr,
		Balance: big.NewInt(0),
		Nonce:   0,
	}
}
