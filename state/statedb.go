package state

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/syndtr/goleveldb/leveldb"
)

type Meta struct {
	MerchantFeeBps uint64              `json:"merchantFeeBps"`
	Fees           map[string]*big.Int `json:"fees"`
	TotalSupply    map[string]*big.Int `json:"totalSupply"`
}

type StateDB struct {
	db   *leveldb.DB
	Meta *Meta
}

func NewStateDB(path string) (*StateDB, error) {
	db, err := leveldb.OpenFile(path, nil)
	if err != nil {
		return nil, err
	}

	s := &StateDB{db: db}
	if err := s.loadMeta(); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *StateDB) Close() {
	if s.db != nil {
		s.db.Close()
	}
}

// ---------------- META ----------------

func (s *StateDB) loadMeta() error {
	raw, err := s.db.Get([]byte("_meta"), nil)
	if err == leveldb.ErrNotFound {
		s.Meta = &Meta{
			Fees:        make(map[string]*big.Int),
			TotalSupply: make(map[string]*big.Int),
		}
		return s.SaveMeta()
	}
	if err != nil {
		return err
	}

	var m Meta
	if err := json.Unmarshal(raw, &m); err != nil {
		return err
	}

	if m.Fees == nil {
		m.Fees = make(map[string]*big.Int)
	}
	if m.TotalSupply == nil {
		m.TotalSupply = make(map[string]*big.Int)
	}

	s.Meta = &m
	return nil
}

func (s *StateDB) SaveMeta() error {
	data, err := json.Marshal(s.Meta)
	if err != nil {
		return err
	}
	return s.db.Put([]byte("_meta"), data, nil)
}

// ---------------- ACCOUNTS ----------------

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
