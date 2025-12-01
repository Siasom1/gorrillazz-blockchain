package state

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

// StateDB is de low-level LevelDB wrapper.
type StateDB struct {
	db *leveldb.DB
}

// NewStateDB opent de LevelDB op de gegeven path.
func NewStateDB(path string) (*StateDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &StateDB{db: db}, nil
}

// Close sluit de DB.
func (s *StateDB) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// SaveAccount slaat een account op onder key = address.Hex().
func (s *StateDB) SaveAccount(acc *Account) error {
	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(acc.Address.Hex()), data, nil)
}

// GetAccount laadt een account, of maakt een nieuwe lege als hij niet bestaat.
func (s *StateDB) GetAccount(addr common.Address) (*Account, error) {
	data, err := s.db.Get([]byte(addr.Hex()), nil)
	if err == leveldb.ErrNotFound {
		// nieuw lege account
		return NewAccount(addr), nil
	}
	if err != nil {
		return nil, err
	}

	var acc Account
	if err := json.Unmarshal(data, &acc); err != nil {
		return nil, err
	}

	if acc.Balance == nil {
		acc.Balance = big.NewInt(0)
	}

	return &acc, nil
}
