package state

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

type StateDB struct {
	db *leveldb.DB
}

func NewStateDB(path string) (*StateDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}
	return &StateDB{db: db}, nil
}

func (s *StateDB) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

func (s *StateDB) SaveAccount(acc *Account) error {
	acc.ensureBalances()

	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}

	return s.db.Put([]byte(acc.Address.Hex()), data, nil)
}

func (s *StateDB) GetAccount(addr common.Address) (*Account, error) {
	raw, err := s.db.Get([]byte(addr.Hex()), nil)
	if err == leveldb.ErrNotFound {
		return NewAccount(addr), nil
	}
	if err != nil {
		return nil, err
	}

	var acc Account
	if err := json.Unmarshal(raw, &acc); err != nil {
		return nil, err
	}

	acc.ensureBalances()
	return &acc, nil
}
