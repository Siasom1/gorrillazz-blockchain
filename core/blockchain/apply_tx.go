package blockchain

import (
	"fmt"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

func (bc *Blockchain) applyTransaction(tx *types.Transaction) error {
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

	return nil
}
