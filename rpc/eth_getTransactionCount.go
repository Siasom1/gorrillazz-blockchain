package rpc

import (
	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
)

func HandleEthGetTransactionCount(bc *blockchain.Blockchain, params []interface{}) (uint64, error) {
	addrStr := params[0].(string)
	addr := bc.State.HexToAddress(addrStr)
	return bc.State.GetNonce(addr), nil
}
