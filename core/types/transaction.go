package types

import (
	"crypto/sha256"
	"encoding/binary"
	"io"

	"github.com/ethereum/go-ethereum/common"
)

type Transaction struct {
	// core
	From   common.Address
	To     common.Address
	Token  TokenType
	Amount uint64
	Fee    uint64
	Nonce  uint64
	Memo   string

	// cached hash
	hash common.Hash
}

func (tx *Transaction) Hash() common.Hash {
	if tx.hash != (common.Hash{}) {
		return tx.hash
	}

	h := sha256.New()
	h.Write([]byte{0x01})
	h.Write(tx.From.Bytes())
	h.Write(tx.To.Bytes())
	writeU64(h, uint64(tx.Token))
	writeU64(h, tx.Amount)
	writeU64(h, tx.Fee)
	writeU64(h, tx.Nonce)
	writeBytes(h, []byte(tx.Memo))

	tx.hash = common.BytesToHash(h.Sum(nil))
	return tx.hash
}

func writeBytes(w io.Writer, b []byte) {
	binary.Write(w, binary.BigEndian, uint16(len(b)))
	w.Write(b)
}

func writeString(w io.Writer, s string) {
	writeBytes(w, []byte(s))
}

func writeU64(w io.Writer, v uint64) {
	binary.Write(w, binary.BigEndian, v)
}
