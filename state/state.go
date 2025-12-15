package state

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type State struct {
	db     *StateDB
	Paused bool

	// --- D.3 Admin accounting ---
	totalSupply map[string]*big.Int
	fees        map[string]*big.Int
}

type Fees struct {
	MerchantFeeBps uint64 // bv 100 = 1%
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
	}, nil
}

func (s *State) Close() error {
	if s.db != nil {
		s.db.Close()
	}
	return nil
}

// ---------------- GORR ----------------

func (s *State) GetBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
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

// ---------------- USDCc ----------------

func (s *State) GetUSDCcBalance(addr common.Address) (*big.Int, error) {
	acc, err := s.db.GetAccount(addr)
	if err != nil {
		return nil, err
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

// ---------------- NONCE ----------------

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
	return s.db.SaveAccount(acc)
}

// ---------------- TOTAL SUPPLY ----------------

func (s *State) AddSupply(token string, amount *big.Int) {
	if s.totalSupply[token] == nil {
		s.totalSupply[token] = big.NewInt(0)
	}
	s.totalSupply[token].Add(s.totalSupply[token], amount)
}

func (s *State) SubSupply(token string, amount *big.Int) error {
	if s.totalSupply[token] == nil {
		return nil // supply onbekend = permissive
	}
	if s.totalSupply[token].Cmp(amount) < 0 {
		return nil // geen hard fail, admin mag burnen
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

// ---------------- FEES ----------------

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

// ---------------- FEES (ADMIN) --------------

// ---------------- FEES / CONFIG ----------------

func (s *State) GetMerchantFeeBps() uint64 {
	if s.db.Meta == nil {
		return 0
	}
	return s.db.Meta.MerchantFeeBps
}

func (s *State) SetMerchantFeeBps(bps uint64) {
	if s.db.Meta == nil {
		return
	}
	s.db.Meta.MerchantFeeBps = bps
}

// ---------------- FEES (COLLECTED) ----------------

func (s *State) GetCollectedFees(token string) *big.Int {
	if s.db.Meta == nil || s.db.Meta.Fees == nil {
		return big.NewInt(0)
	}
	if s.db.Meta.Fees[token] == nil {
		return big.NewInt(0)
	}
	return new(big.Int).Set(s.db.Meta.Fees[token])
}

func (s *State) AddCollectedFee(token string, amount *big.Int) {
	if s.db.Meta == nil {
		return
	}
	if s.db.Meta.Fees == nil {
		s.db.Meta.Fees = make(map[string]*big.Int)
	}
	if s.db.Meta.Fees[token] == nil {
		s.db.Meta.Fees[token] = big.NewInt(0)
	}
	s.db.Meta.Fees[token].Add(s.db.Meta.Fees[token], amount)
	_ = s.db.SaveMeta()
}

func (s *State) SubCollectedFee(token string, amount *big.Int) error {
	if amount == nil || amount.Sign() <= 0 {
		return nil
	}
	if s.db == nil || s.db.Meta == nil || s.db.Meta.Fees == nil || s.db.Meta.Fees[token] == nil {
		return nil
	}
	// clamp (geen crash als admin teveel vraagt)
	if s.db.Meta.Fees[token].Cmp(amount) < 0 {
		s.db.Meta.Fees[token].SetInt64(0)
	} else {
		s.db.Meta.Fees[token].Sub(s.db.Meta.Fees[token], amount)
	}
	return s.db.SaveMeta()
}
