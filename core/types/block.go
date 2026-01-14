package types

import (
	"crypto/sha256"
	"encoding/binary"

	"github.com/ethereum/go-ethereum/common"
)

type Block struct {
	Header       *Header
	Transactions []*Transaction
}

type Header struct {
	Number     uint64
	Time       uint64
	ParentHash common.Hash
	StateRoot  common.Hash
	TxRoot     common.Hash

	Hash common.Hash // runtime cached
}

func (b *Block) Hash() common.Hash {
	if b.Header.Hash != (common.Hash{}) {
		return b.Header.Hash
	}

	h := sha256.New()
	h.Write([]byte{0x02})

	writeUint64(h, b.Header.Number)
	writeUint64(h, b.Header.Time)
	h.Write(b.Header.ParentHash.Bytes())
	h.Write(b.Header.StateRoot.Bytes())
	h.Write(b.Header.TxRoot.Bytes())

	for _, tx := range b.Transactions {
		h.Write(tx.Hash().Bytes())
	}

	b.Header.Hash = common.BytesToHash(h.Sum(nil))
	return b.Header.Hash
}

// helper
func writeUint64(h interface{ Write([]byte) (int, error) }, v uint64) {
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], v)
	h.Write(buf[:])
}
