package rpc

import (
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
)

func HandleEthGetBalance(bc *blockchain.Blockchain, params []interface{}) (*big.Int, error) {
	if len(params) < 1 {
		return nil, nil
	}
	addrStr := params[0].(string)
	addr := common.HexToAddress(addrStr)
	return bc.State.GetBalance(addr), nil
}

func HandleEthGetTransactionCount(bc *blockchain.Blockchain, params []interface{}) (uint64, error) {
	if len(params) < 1 {
		return 0, nil
	}
	addrStr := params[0].(string)
	addr := common.HexToAddress(addrStr)
	return bc.State.GetNonce(addr), nil
}
