package rpc

import (
	"encoding/json"
	"math/big"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
)

type createIntentParams struct {
	Merchant string `json:"merchant"`
	Amount   string `json:"amount"`
	Token    string `json:"token"`
}

func handleCreatePaymentIntent(bc *blockchain.Blockchain, raw json.RawMessage) (interface{}, interface{}) {
	var params []interface{}
	json.Unmarshal(raw, &params)

	merchant := common.HexToAddress(params[0].(string))
	amount := new(big.Int)
	amount.SetString(params[1].(string), 10)
	token := params[2].(string)

	intent, id, err := bc.Payment.CreateIntent(merchant, amount, token, uint64(bc.Head().Header.Time))
	if err != nil {
		return nil, err.Error()
	}

	return map[string]interface{}{"id": id, "intent": intent}, nil
}

func handleGetPaymentIntent(bc *blockchain.Blockchain, raw json.RawMessage) (interface{}, interface{}) {
	var params []uint64
	json.Unmarshal(raw, &params)

	intent, err := bc.Payment.GetIntent(params[0])
	if err != nil {
		return nil, err.Error()
	}
	return intent, nil
}

func handleListMerchantPayments(bc *blockchain.Blockchain, raw json.RawMessage) (interface{}, interface{}) {
	var params []string
	json.Unmarshal(raw, &params)

	merchant := common.HexToAddress(params[0])

	out := bc.Payment.ListMerchantPayments(merchant)
	return out, nil
}
