package blockchain

import (
	"fmt"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

func (bc *Blockchain) applyTransaction(tx *types.Transaction) error {
	switch tx.Token {
	case types.TokenGORR:
		bal, err := bc.State.GetBalance(tx.From)
		if err != nil {
			return fmt.Errorf("failed to get sender GORR balance: %v", err)
		}
		if bal.Cmp(tx.Value) < 0 {
			return fmt.Errorf("insufficient GORR balance")
		}
		if err := bc.State.SetBalance(tx.From, new(big.Int).Sub(bal, tx.Value)); err != nil {
			return fmt.Errorf("failed to subtract GORR: %v", err)
		}
		if tx.To != nil {
			recvBal, err := bc.State.GetBalance(*tx.To)
			if err != nil {
				return fmt.Errorf("failed to get recipient GORR balance: %v", err)
			}
			if err := bc.State.SetBalance(*tx.To, new(big.Int).Add(recvBal, tx.Value)); err != nil {
				return fmt.Errorf("failed to add GORR: %v", err)
			}
		}

	case types.TokenUSDCc:
		bal, err := bc.State.GetUSDCcBalance(tx.From)
		if err != nil {
			return fmt.Errorf("failed to get sender USDCc balance: %v", err)
		}
		if bal.Cmp(tx.Value) < 0 {
			return fmt.Errorf("insufficient USDCc balance")
		}
		if err := bc.State.SetUSDCcBalance(tx.From, new(big.Int).Sub(bal, tx.Value)); err != nil {
			return fmt.Errorf("failed to subtract USDCc: %v", err)
		}
		if tx.To != nil {
			recvBal, err := bc.State.GetUSDCcBalance(*tx.To)
			if err != nil {
				return fmt.Errorf("failed to get recipient USDCc balance: %v", err)
			}
			if err := bc.State.SetUSDCcBalance(*tx.To, new(big.Int).Add(recvBal, tx.Value)); err != nil {
				return fmt.Errorf("failed to add USDCc: %v", err)
			}
		}

	default:
		return fmt.Errorf("unsupported token type: %s", tx.Token)
	}

	if err := bc.State.IncNonce(tx.From); err != nil {
		return fmt.Errorf("failed to increment nonce: %v", err)
	}

	return nil
}
