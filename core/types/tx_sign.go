package types

import (
	"crypto/ecdsa"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/rlp"
)

// Hash of a transaction before signature
func (tx *Transaction) Hash() common.Hash {
	// Build RLP without signature fields
	data := txdata{
		Nonce:    tx.Nonce,
		To:       tx.To,
		Value:    tx.Value,
		Gas:      tx.Gas,
		GasPrice: tx.GasPrice,
		Data:     tx.Data,
	}

	encoded, _ := rlp.EncodeToBytes(data)
	return crypto.Keccak256Hash(encoded)
}

// Recover sender from signature
func (tx *Transaction) From() (common.Address, error) {
	if tx.from != nil {
		return *tx.from, nil
	}
	if !tx.IsSigned() {
		return common.Address{}, errors.New("tx not signed")
	}

	// Build message hash
	h := tx.Hash()

	// Recreate signature bytes
	sig := make([]byte, 65)
	copy(sig[32-len(tx.R.Bytes()):32], tx.R.Bytes())
	copy(sig[64-len(tx.S.Bytes()):64], tx.S.Bytes())
	sig[64] = byte(tx.V.Uint64() - 27)

	pub, err := crypto.SigToPub(h.Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	addr := crypto.PubkeyToAddress(*pub)
	tx.from = &addr

	return addr, nil
}

// Sign transaction using a private key
func (tx *Transaction) Sign(priv *ecdsa.PrivateKey) error {
	hash := tx.Hash()
	sig, err := crypto.Sign(hash.Bytes(), priv)
	if err != nil {
		return err
	}

	tx.R = new(big.Int).SetBytes(sig[:32])
	tx.S = new(big.Int).SetBytes(sig[32:64])
	tx.V = big.NewInt(int64(sig[64] + 27))

	return nil
}
