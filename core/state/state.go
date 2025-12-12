package state

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/common"
)

//
// --------------------------------------------------
// State structure (balances, USDCc, metadata)
// --------------------------------------------------

type AccountState struct {
	Balance      *big.Int `json:"balance"`     // GORR
	USDCcBalance *big.Int `json:"usdcBalance"` // USDCc
}

type State struct {
	path     string
	Accounts map[string]*AccountState
}

//
// --------------------------------------------------
// Constructor
// --------------------------------------------------

func NewState(path string) (*State, error) {
	s := &State{
		path:     path,
		Accounts: map[string]*AccountState{},
	}

	if _, err := os.Stat(path); err == nil {
		// Load existing state
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if len(data) > 0 {
			if err := json.Unmarshal(data, s); err != nil {
				return nil, err
			}
		}
	}

	return s, nil
}

//
// --------------------------------------------------
// Internal helpers
// --------------------------------------------------

func (s *State) getOrCreate(addr common.Address) *AccountState {
	key := addr.Hex()

	if acc, ok := s.Accounts[key]; ok {
		return acc
	}

	// create fresh
	acc := &AccountState{
		Balance:      big.NewInt(0),
		USDCcBalance: big.NewInt(0),
	}
	s.Accounts[key] = acc
	return acc
}

//
// --------------------------------------------------
// GORR balance methods
// --------------------------------------------------

func (s *State) GetBalance(addr common.Address) *big.Int {
	return new(big.Int).Set(s.getOrCreate(addr).Balance)
}

func (s *State) SetBalance(addr common.Address, amount *big.Int) {
	s.getOrCreate(addr).Balance = new(big.Int).Set(amount)
	s.Commit()
}

func (s *State) AddBalance(addr common.Address, amount *big.Int) {
	acc := s.getOrCreate(addr)
	acc.Balance = new(big.Int).Add(acc.Balance, amount)
	s.Commit()
}

func (s *State) SubBalance(addr common.Address, amount *big.Int) error {
	acc := s.getOrCreate(addr)
	if acc.Balance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient GORR balance")
	}
	acc.Balance = new(big.Int).Sub(acc.Balance, amount)
	s.Commit()
	return nil
}

//
// --------------------------------------------------
// USDCc balance methods
// --------------------------------------------------

func (s *State) GetUSDCcBalance(addr common.Address) *big.Int {
	return new(big.Int).Set(s.getOrCreate(addr).USDCcBalance)
}

func (s *State) SetUSDCcBalance(addr common.Address, amount *big.Int) {
	acc := s.getOrCreate(addr)
	acc.USDCcBalance = new(big.Int).Set(amount)
	s.Commit()
}

func (s *State) AddUSDCc(addr common.Address, amount *big.Int) {
	acc := s.getOrCreate(addr)
	acc.USDCcBalance = new(big.Int).Add(acc.USDCcBalance, amount)
	s.Commit()
}

func (s *State) SubUSDCc(addr common.Address, amount *big.Int) error {
	acc := s.getOrCreate(addr)
	if acc.USDCcBalance.Cmp(amount) < 0 {
		return fmt.Errorf("insufficient USDCc balance")
	}
	acc.USDCcBalance = new(big.Int).Sub(acc.USDCcBalance, amount)
	s.Commit()
	return nil
}

//
// --------------------------------------------------
// Commit — save entire state to JSON
// --------------------------------------------------

func (s *State) Commit() error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return err
	}

	return nil
}

//
// --------------------------------------------------
// Debug tooling
// --------------------------------------------------

func (s *State) Dump() {
	fmt.Println("---- STATE DUMP ----")
	for addr, acc := range s.Accounts {
		fmt.Println(addr, "| GORR:", acc.Balance.String(), "| USDCc:", acc.USDCcBalance.String())
	}
	fmt.Println("---------------------")
}
