package state

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// AddBalance: native balance verhogen
func (s *State) AddBalance(addr common.Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("invalid amount")
	}

	cur, err := s.GetBalance(addr)
	if err != nil {
		return err
	}
	if cur == nil {
		cur = big.NewInt(0)
	}

	next := new(big.Int).Add(cur, amount)
	s.SetBalance(addr, next)
	return nil
}

// SubBalance: native balance verlagen (met underflow check)
func (s *State) SubBalance(addr common.Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("invalid amount")
	}

	cur, err := s.GetBalance(addr)
	if err != nil {
		return err
	}
	if cur == nil {
		cur = big.NewInt(0)
	}

	if cur.Cmp(amount) < 0 {
		return errors.New("insufficient balance")
	}

	next := new(big.Int).Sub(cur, amount)
	s.SetBalance(addr, next)
	return nil
}

// AddUSDCc: USDCc balance verhogen
func (s *State) AddUSDCc(addr common.Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("invalid amount")
	}

	cur, err := s.GetUSDCcBalance(addr)
	if err != nil {
		return err
	}
	if cur == nil {
		cur = big.NewInt(0)
	}

	next := new(big.Int).Add(cur, amount)
	s.SetUSDCcBalance(addr, next)
	return nil
}

// (optioneel maar handig) SubUSDCc
func (s *State) SubUSDCc(addr common.Address, amount *big.Int) error {
	if amount == nil || amount.Sign() < 0 {
		return errors.New("invalid amount")
	}

	cur, err := s.GetUSDCcBalance(addr)
	if err != nil {
		return err
	}
	if cur == nil {
		cur = big.NewInt(0)
	}

	if cur.Cmp(amount) < 0 {
		return errors.New("insufficient USDCc balance")
	}

	next := new(big.Int).Sub(cur, amount)
	s.SetUSDCcBalance(addr, next)
	return nil
}
