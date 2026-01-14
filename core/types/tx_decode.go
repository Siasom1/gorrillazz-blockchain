package types

import (
	"encoding/binary"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

func DecodeTransaction(r io.Reader) (*Transaction, error) {
	tx := new(Transaction)

	from, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	tx.From = common.BytesToAddress(from)

	to, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	tx.To = common.BytesToAddress(to)

	if tx.Amount, err = readU64(r); err != nil {
		return nil, err
	}
	if tx.Nonce, err = readU64(r); err != nil {
		return nil, err
	}

	tokenRaw, err := readU64(r)
	if err != nil {
		return nil, err
	}
	tx.Token = TokenType(tokenRaw)

	if tx.Fee, err = readU64(r); err != nil {
		return nil, err
	}

	memo, err := readBytes(r)
	if err != nil {
		return nil, err
	}
	tx.Memo = string(memo)

	return tx, nil
}

func readBytes(r io.Reader) ([]byte, error) {
	var l uint16
	if err := binary.Read(r, binary.BigEndian, &l); err != nil {
		return nil, err
	}
	b := make([]byte, l)
	_, err := io.ReadFull(r, b)
	return b, err
}

func readU64(r io.Reader) (uint64, error) {
	var v uint64
	err := binary.Read(r, binary.BigEndian, &v)
	return v, err
}
