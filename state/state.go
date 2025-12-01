package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// State is een hogere-level wrapper rond StateDB met makkelijke helpers.
type State struct {
	db *StateDB
}

// NewState opent StateDB en wrapped deze in State.
func NewState(path string) (*State, error) {
	db, err := NewStateDB(path)
	if err != nil {
		return nil, err
	}
	return &State{db: db}, nil
}

func (s *State) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// ------------------- READ ---------------------

func (s *State) GetBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if acc.Balance == nil {
		acc.Balance = big.NewInt(0)
	}
	return acc.Balance, nil
}

func (s *State) GetNonce(addr common.Address) (uint64, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

// ------------------- WRITE ---------------------

func (s *State) SetBalance(addr common.Address, amount *big.Int) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}

	if amount == nil {
		amount = big.NewInt(0)
	}

	// kopie zodat callers de pointer niet per ongeluk muteren
	acc.Balance = new(big.Int).Set(amount)
	return s.db.SaveAccount(acc)
}

func (s *State) IncreaseNonce(addr common.Address) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Nonce++
	return s.db.SaveAccount(acc)
}
