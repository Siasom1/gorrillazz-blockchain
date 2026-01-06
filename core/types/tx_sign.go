package types

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// SignLegacy signs a transaction using private key
func SignLegacy(tx *Transaction, privHex string, chainID *big.Int) error {
	priv, err := crypto.HexToECDSA(privHex)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}

	hash := tx.Hash()

	sig, err := crypto.Sign(hash.Bytes(), priv)
	if err != nil {
		return fmt.Errorf("sign failed: %w", err)
	}

	r := new(big.Int).SetBytes(sig[0:32])
	s := new(big.Int).SetBytes(sig[32:64])
	v := big.NewInt(int64(sig[64]))

	if chainID != nil {
		v.Add(v, new(big.Int).Mul(chainID, big.NewInt(2)))
		v.Add(v, big.NewInt(35))
	} else {
		v.Add(v, big.NewInt(27))
	}

	tx.R = r
	tx.S = s
	tx.V = v

	return nil
}

// Recover sender address
func RecoverSender(tx *Transaction) (common.Address, error) {
	if tx.R == nil || tx.S == nil || tx.V == nil {
		return common.Address{}, fmt.Errorf("transaction not signed")
	}

	recoveryID := byte((tx.V.Uint64() - 35) % 2)
	sig := make([]byte, 65)

	rb := tx.R.Bytes()
	sb := tx.S.Bytes()

	copy(sig[32-len(rb):32], rb)
	copy(sig[64-len(sb):64], sb)
	sig[64] = recoveryID

	pub, err := crypto.Ecrecover(tx.Hash().Bytes(), sig)
	if err != nil {
		return common.Address{}, err
	}

	key, err := crypto.UnmarshalPubkey(pub)
	if err != nil {
		return common.Address{}, err
	}

	return crypto.PubkeyToAddress(*key), nil
}
