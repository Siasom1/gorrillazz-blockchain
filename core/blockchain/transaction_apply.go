package blockchain

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

func (bc *Blockchain) applyTransaction(tx *types.Transaction) error {
	// Nonce check
	currentNonce := bc.State.GetNonce(tx.From)
	if tx.Nonce != currentNonce {
		return fmt.Errorf("invalid nonce: expected %d got %d", currentNonce, tx.Nonce)
	}

	// Token switch
	switch tx.Token {
	case types.TokenGORR:
		if err := bc.State.SubBalance(tx.From, tx.Value); err != nil {
			return err
		}
		bc.State.AddBalance(*tx.To, tx.Value)

	case types.TokenUSDCc:
		if err := bc.State.SubUSDCc(tx.From, tx.Value); err != nil {
			return err
		}
		bc.State.AddUSDCc(*tx.To, tx.Value)

	default:
		return fmt.Errorf("unknown token")
	}

	// Increment nonce
	bc.State.IncNonce(tx.From)

	return nil
}
