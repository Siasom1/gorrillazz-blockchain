package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Account is een simpele account-representatie in de state.
type Account struct {
	Address common.Address `json:"address"`
	Balance *big.Int       `json:"balance"`
	Nonce   uint64         `json:"nonce"`
}

// NewAccount maakt een nieuwe lege account.
func NewAccount(addr common.Address) *Account {
	return &Account{
		Address: addr,
		Balance: big.NewInt(0),
		Nonce:   0,
	}
}
