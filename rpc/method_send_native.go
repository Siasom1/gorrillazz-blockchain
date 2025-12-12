package rpc

import (
	"errors"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
)

func HandleSendNative(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {

	if len(params) == 0 {
		return nil, errors.New("missing params")
	}

	obj, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid tx object")
	}

	from := common.HexToAddress(obj["from"].(string))
	to := common.HexToAddress(obj["to"].(string))

	amountFloat := obj["amount"].(float64)
	amountWei := new(big.Int).Mul(big.NewInt(int64(amountFloat*1e6)), big.NewInt(1e12))

	// ♻️ BELANGRIJK: PaymentGateway zit in bc.Gateway
	txHash, err := bc.Gateway.SendNative(from, to, amountWei)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"txHash": txHash.Hex(),
	}, nil
}
