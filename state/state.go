package state

import (
	// "fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type EscrowAccount struct {
	Balance     *big.Int
	DailyLimit  *big.Int
	PayoutToday *big.Int
}

type State struct {
	db     *StateDB
	Paused bool

	totalSupply map[string]*big.Int
	fees        map[string]*big.Int
}

func NewState(path string) (*State, error) {
	db, err := NewStateDB(path)
	if err != nil {
		return nil, err
	}
	return &State{
		db:          db,
		totalSupply: make(map[string]*big.Int),
		fees:        make(map[string]*big.Int),
		// escrow:      make(map[common.Address]*EscrowAccount),
	}, nil
}

func (s *State) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// --- GORR ---
func (s *State) GetBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if acc.Balances["GORR"] == nil {
		acc.Balances["GORR"] = big.NewInt(0)
	}
	return new(big.Int).Set(acc.Balances["GORR"]), nil
}

func (s *State) SetBalance(addr common.Address, amount *big.Int) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Balances["GORR"] = new(big.Int).Set(amount)
	return s.db.SaveAccount(acc)
}

// --- USDCc ---
func (s *State) GetUSDCcBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
	}
	if acc.Balances["USDCc"] == nil {
		acc.Balances["USDCc"] = big.NewInt(0)
	}
	return new(big.Int).Set(acc.Balances["USDCc"]), nil
}

func (s *State) SetUSDCcBalance(addr common.Address, amount *big.Int) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Balances["USDCc"] = new(big.Int).Set(amount)
	return s.db.SaveAccount(acc)
}

// --- NONCE ---
func (s *State) GetNonce(addr common.Address) (uint64, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return 0, err
	}
	return acc.Nonce, nil
}

func (s *State) IncNonce(addr common.Address) error {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return err
	}
	acc.Nonce++
	return s.db.SaveAccount(acc)
}

// --- TOTAL SUPPLY ---
func (s *State) AddSupply(token string, amount *big.Int) {
	if amount == nil || amount.Sign() <= 0 {
		return
	}
	if s.totalSupply[token] == nil {
		s.totalSupply[token] = big.NewInt(0)
	}
	s.totalSupply[token].Add(s.totalSupply[token], amount)
}

func (s *State) SubSupply(token string, amount *big.Int) error {
	if s.totalSupply[token] == nil {
		return nil
	}
	if s.totalSupply[token].Cmp(amount) < 0 {
		return nil
	}
	s.totalSupply[token].Sub(s.totalSupply[token], amount)
	return nil
}

func (s *State) GetTotalSupply(token string) *big.Int {
	if s.totalSupply[token] == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(s.totalSupply[token])
}

// --- FEES ---
func (s *State) AddFee(token string, amount *big.Int) {
	if s.fees[token] == nil {
		s.fees[token] = big.NewInt(0)
	}
	s.fees[token].Add(s.fees[token], amount)
}

func (s *State) GetFees(token string) *big.Int {
	if s.fees[token] == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(s.fees[token])
}

// // --- ESCROW ---
// func (s *State) GetEscrow(addr common.Address) *EscrowAccount {
// 	if s.escrow[addr] == nil {
// 		s.escrow[addr] = &EscrowAccount{
// 			Balance:     big.NewInt(0),
// 			DailyLimit:  big.NewInt(0),
// 			PayoutToday: big.NewInt(0),
// 		}
// 	}
// 	return s.escrow[addr]
// }

// func (s *State) SetEscrowLimit(addr common.Address, limit *big.Int) {
// 	e := s.GetEscrow(addr)
// 	e.DailyLimit = new(big.Int).Set(limit)
// 	e.PayoutToday = big.NewInt(0)
// }

// func (s *State) DepositEscrow(addr common.Address, amount *big.Int) {
// 	e := s.GetEscrow(addr)
// 	e.Balance.Add(e.Balance, amount)
// }

// func (s *State) WithdrawEscrow(addr common.Address, amount *big.Int) error {
// 	e := s.GetEscrow(addr)
// 	if new(big.Int).Add(e.PayoutToday, amount).Cmp(e.DailyLimit) > 0 {
// 		return fmt.Errorf("withdraw exceeds daily limit")
// 	}
// 	if e.Balance.Cmp(amount) < 0 {
// 		return fmt.Errorf("insufficient escrow balance")
// 	}
// 	e.Balance.Sub(e.Balance, amount)
// 	e.PayoutToday.Add(e.PayoutToday, amount)
// 	return nil
// }
