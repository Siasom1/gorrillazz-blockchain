package rpc

import (
	"encoding/hex"
	"fmt"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/types"
	"github.com/ethereum/go-ethereum/common"
)

func (e *ethRPC) SendRawTransaction(raw string) (common.Hash, error) {
	data, err := hex.DecodeString(raw[2:])
	if err != nil {
		return common.Hash{}, err
	}

	tx := new(types.Transaction)
	if err := tx.UnmarshalRLP(data); err != nil {
		return common.Hash{}, err
	}

	if err := e.bc.ApplyTransaction(tx); err != nil {
		return common.Hash{}, err
	}

	return tx.Hash(), nil
}

func HandleEthSendRawTransaction(w http.ResponseWriter, req rpcReq, eth *ethRPC) {
	if len(req.Params) < 1 {
		writeJSON(w, req.ID, nil, fmt.Errorf("missing raw tx"))
		return
	}

	raw := req.Params[0].(string)
	hash, err := eth.SendRawTransaction(raw)
	writeJSON(w, req.ID, hash.Hex(), err)
}
