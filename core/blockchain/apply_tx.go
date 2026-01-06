package blockchain

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

func (bc *Blockchain) applyTransaction(tx *types.Transaction) error {
	nonce, _ := bc.State.GetNonce(tx.From)

	if tx.Nonce != nonce {
		return fmt.Errorf("invalid nonce")
	}

	switch tx.Token {
	case "GORR":
		if err := bc.State.SubBalance(tx.From, tx.Value); err != nil {
			return err
		}
		bc.State.AddBalance(*tx.To, tx.Value)

	case "USDCc":
		if err := bc.State.SubUSDCc(tx.From, tx.Value); err != nil {
			return err
		}
		bc.State.AddUSDCc(*tx.To, tx.Value)
	}

	bc.State.IncNonce(tx.From)
	return nil
}
