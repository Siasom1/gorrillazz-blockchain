package state

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type Account struct {
	Balance *big.Int `json:"balance"`
	Nonce   uint64   `json:"nonce"`
}

type StateDB struct {
	mu       sync.RWMutex
	dataDir  string
	accounts map[string]*Account
}

func NewStateDB(dataDir string) (*StateDB, error) {
	db := &StateDB{
		dataDir:  dataDir,
		accounts: make(map[string]*Account),
	}

	if err := db.load(); err != nil {
		return nil, err
	}

	return db, nil
}

// ------------------------------------------------------------
// Load & Save State
// ------------------------------------------------------------

func (s *StateDB) load() error {
	path := filepath.Join(s.dataDir, "state.json")

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // new chain
	}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(bytes, &s.accounts)
}

func (s *StateDB) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := filepath.Join(s.dataDir, "state.json")
	bytes, err := json.MarshalIndent(s.accounts, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, bytes, 0o644)
}

// ------------------------------------------------------------
// Account Operations
// ------------------------------------------------------------

func (s *StateDB) GetOrCreate(addr common.Address) *Account {
	key := addr.Hex()

	s.mu.Lock()
	defer s.mu.Unlock()

	acc, ok := s.accounts[key]
	if !ok {
		acc = &Account{
			Balance: big.NewInt(0),
			Nonce:   0,
		}
		s.accounts[key] = acc
	}

	return acc
}

func (s *StateDB) GetBalance(addr common.Address) (*big.Int, error) {
	acc := s.GetOrCreate(addr)
	return new(big.Int).Set(acc.Balance), nil
}

func (s *StateDB) SetBalance(addr common.Address, amount *big.Int) {
	acc := s.GetOrCreate(addr)
	acc.Balance = new(big.Int).Set(amount)
}

func (s *StateDB) GetNonce(addr common.Address) (uint64, error) {
	acc := s.GetOrCreate(addr)
	return acc.Nonce, nil
}

func (s *StateDB) IncreaseNonce(addr common.Address) {
	acc := s.GetOrCreate(addr)
	acc.Nonce++
}

// ------------------------------------------------------------
// State Root (simple hash of state.json)
// ------------------------------------------------------------

func (s *StateDB) RootHash() common.Hash {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// serialize whole state
	bytes, _ := json.Marshal(s.accounts)
	return common.BytesToHash(bytes)
}
