package types

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// From recovers the sender from an EIP-155 signed transaction.
func (tx *Transaction) From() (common.Address, error) {
	if tx.V == nil || tx.R == nil || tx.S == nil {
		return common.Address{}, errors.New("missing signature")
	}

	// Compute recovery id (EIP-155)
	// V = 27/28 (no chainid) OR V = 35 + 2·chainId or 36 + 2·chainId
	v := tx.V.Uint64()
	recoveryID := byte((v - 35) % 2) // always 0 or 1

	// Build 65-byte signature [R || S || V]
	sig := make([]byte, 65)

	rb := tx.R.Bytes()
	sb := tx.S.Bytes()

	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = recoveryID

	// Hash per yellowpaper (Keccak256 of RLP of signing data)
	h := tx.Hash()

	pub, err := crypto.Ecrecover(h.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	key, err := crypto.UnmarshalPubkey(pub)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*key), nil
}
