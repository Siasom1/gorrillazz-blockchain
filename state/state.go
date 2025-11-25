package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type State struct {
	db *StateDB
}

func NewState(path string) (*State, error) {
	db, err := NewStateDB(path)
	if err != nil {
		return nil, err
	}
	return &State{db: db}, nil
}

func (s *State) GetBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	return new(big.Int).Set(acc.Balance), nil
}

func (s *State) SetBalance(addr common.Address, amount *big.Int) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Balance = new(big.Int).Set(amount)
	return s.db.SaveAccount(acc)
}

func (s *State) GetNonce(addr common.Address) (uint64, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

func (s *State) IncreaseNonce(addr common.Address) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Nonce++
	return s.db.SaveAccount(acc)
}

func (s *State) RootHash() (common.Hash, error) {
	return s.db.RootHash()
}
