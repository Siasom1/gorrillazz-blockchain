package types

import (
	"crypto/sha256"
	"encoding/hex"
)

type Hash [32]byte

func (h Hash) String() string {
	return "0x" + hex.EncodeToString(h[:])
}

func BytesToHash(b []byte) Hash {
	var h Hash
	copy(h[:], b)
	return h
}

func HashBytes(data []byte) Hash {
	sum := sha256.Sum256(data)
	return Hash(sum)
}
