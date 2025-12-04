package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Multi-asset account for native GORR + USDCc.
type Account struct {
	Address  common.Address      `json:"address"`
	Balances map[string]*big.Int `json:"balances"`
	Nonce    uint64              `json:"nonce"`
}

// Create an empty account with GORR + USDCc = 0
func NewAccount(addr common.Address) *Account {
	return &Account{
		Address:  addr,
		Balances: newDefaultBalances(),
		Nonce:    0,
	}
}

// Default assets map
func newDefaultBalances() map[string]*big.Int {
	return map[string]*big.Int{
		"GORR":  big.NewInt(0),
		"USDCc": big.NewInt(0),
	}
}

// Ensure all balances exist
func (a *Account) ensureBalances() {
	if a.Balances == nil {
		a.Balances = newDefaultBalances()
		return
	}
	if a.Balances["GORR"] == nil {
		a.Balances["GORR"] = big.NewInt(0)
	}
	if a.Balances["USDCc"] == nil {
		a.Balances["USDCc"] = big.NewInt(0)
	}
}
