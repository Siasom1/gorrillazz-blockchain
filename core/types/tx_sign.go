package types

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// RecoverSender recovers the sender address from a legacy (type 0) tx
func RecoverSender(tx *Transaction, chainID *big.Int) (common.Address, error) {
	// Legacy signing payload (EIP-155)
	var sigData []interface{}

	if chainID != nil {
		sigData = []interface{}{
			tx.Nonce,
			tx.GasPrice,
			tx.Gas,
			tx.To,
			tx.Value,
			tx.Data,
			chainID,
			uint(0),
			uint(0),
		}
	} else {
		sigData = []interface{}{
			tx.Nonce,
			tx.GasPrice,
			tx.Gas,
			tx.To,
			tx.Value,
			tx.Data,
		}
	}

	encoded, err := rlp.EncodeToBytes(sigData)
	if err != nil {
		return common.Address{}, err
	}

	hash := crypto.Keccak256Hash(encoded)

	// --- recover V ---
	v := tx.V.Uint64()
	var recoveryID byte

	if chainID != nil {
		recoveryID = byte((v - 35 - 2*chainID.Uint64()) % 2)
	} else {
		recoveryID = byte(v - 27)
	}

	// --- build signature ---
	sig := make([]byte, 65)

	rb := tx.R.Bytes()
	sb := tx.S.Bytes()

	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = recoveryID

	pub, err := crypto.Ecrecover(hash.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	key, err := crypto.UnmarshalPubkey(pub)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*key), nil
}
