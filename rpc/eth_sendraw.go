package rpc

import (
	"encoding/hex"

	"github.com/Siasom1/gorrillazz-chain/core/types"
)

func (e *ethRPC) SendRawTransaction(rawHex string) (string, error) {
	raw, err := hex.DecodeString(rawHex[2:])
	if err != nil {
		return "", err
	}

	tx, err := DecodeLegacyTx(raw)
	if err != nil {
		return "", err
	}

	from, err := types.RecoverSender(tx)
	if err != nil {
		return "", err
	}
	tx.From = from

	if err := e.bc.ProcessTransaction(tx); err != nil {
		return "", err
	}

	return tx.Hash().Hex(), nil
}
