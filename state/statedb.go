package state

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

type StateDB struct {
	db *leveldb.DB
}

// ------------------------------------------------------------
// Open / Close
// ------------------------------------------------------------
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

// ------------------------------------------------------------
// Load / Save Accounts
// ------------------------------------------------------------
func (s *StateDB) SaveAccount(acc *Account) error {
	data, err := json.Marshal(acc)
	if err != nil {
		return err
	}
	return s.db.Put([]byte(acc.Address.Hex()), data, nil)
}

func (s *StateDB) GetAccount(addr common.Address) (*Account, error) {
	bytes, err := s.db.Get([]byte(addr.Hex()), nil)
	if err == leveldb.ErrNotFound {
		return NewAccount(addr), nil
	}
	if err != nil {
		return nil, err
	}

	var acc Account
	if err := json.Unmarshal(bytes, &acc); err != nil {
		return nil, err
	}

	return &acc, nil
}

// ------------------------------------------------------------
// Compute StateRoot (simple hash)
// ------------------------------------------------------------
func (s *StateDB) RootHash() (common.Hash, error) {
	iter := s.db.NewIterator(nil, nil)
	defer iter.Release()

	combined := []byte{}

	for iter.Next() {
		key := iter.Key()
		val := iter.Value()
		combined = append(combined, key...)
		combined = append(combined, val...)
	}

	if err := iter.Error(); err != nil {
		return common.Hash{}, err
	}

	return common.BytesToHash(combined), nil
}
