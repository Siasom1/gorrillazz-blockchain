package rpc

import (
	"encoding/json"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
)

func Dispatch(bc *blockchain.Blockchain, bus *events.EventBus, method string, raw json.RawMessage) (interface{}, interface{}) {
	switch method {

	case "gorr_createPaymentIntent":
		return handleCreatePaymentIntent(bc, raw)

	case "gorr_getPaymentIntent":
		return handleGetPaymentIntent(bc, raw)

	case "gorr_listMerchantPayments":
		return handleListMerchantPayments(bc, raw)

	default:
		return nil, map[string]interface{}{
			"code":    -32601,
			"message": "Method not found",
		}
	}
}
