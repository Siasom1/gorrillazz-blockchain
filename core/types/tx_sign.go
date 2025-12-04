package types

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// From returns the sender address by recovering the public key
// from the V,R,S signature.
func (tx *Transaction) From() (common.Address, error) {
	if tx.V == nil || tx.R == nil || tx.S == nil {
		return common.Address{}, errors.New("missing signature (V/R/S)")
	}

	// Build the sig:
	// Ethereum expects: [R || S || V]
	sig := make([]byte, 65)

	rb := tx.R.Bytes()
	sb := tx.S.Bytes()

	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = byte(tx.V.Uint64() - 27) // normalize V (Ethereum rules)

	// Hash of the transaction:
	h := tx.Hash()

	pubKey, err := crypto.Ecrecover(h.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	pub, err := crypto.UnmarshalPubkey(pubKey)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*pub), nil
}
