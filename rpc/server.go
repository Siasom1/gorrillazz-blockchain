package rpc

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/ethereum/go-ethereum/common"
)

type Server struct {
	bc  *blockchain.Blockchain
	bus *events.EventBus
}

func NewServer(bc *blockchain.Blockchain, bus *events.EventBus) *Server {
	return &Server{bc: bc, bus: bus}
}

//
// ------------------------------------------------------------
// JSON RESPONSE HELPERS
// ------------------------------------------------------------
//

func writeJSON(w http.ResponseWriter, result interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"error":   err.Error(),
			"id":      1,
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  result,
		"id":      1,
	})
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, nil, err)
}

//
// ------------------------------------------------------------
// JSON-RPC HANDLER
// ------------------------------------------------------------
//

func (s *Server) HandleJSONRPC(w http.ResponseWriter, r *http.Request) {
	var req map[string]interface{}
	json.NewDecoder(r.Body).Decode(&req)

	method, _ := req["method"].(string)
	params, _ := req["params"].([]interface{})

	switch method {

	//
	// PAYMENT INTENTS
	//
	case "gorr_createPaymentIntent":
		result, err := HandleCreatePaymentIntent(s.bc, params)
		writeJSON(w, result, err)

	case "gorr_getPaymentIntent":
		result, err := HandleGetPaymentIntent(s.bc, params)
		writeJSON(w, result, err)

	case "gorr_listMerchantPayments":
		result, err := HandleListMerchantPayments(s.bc, params)
		writeJSON(w, result, err)

	//
	// NATIVE TOKEN TRANSFER
	//
	case "gorr_sendTransaction":
		result, err := HandleSendNative(s.bc, params)
		writeJSON(w, result, err)

	default:
		writeError(w, fmt.Errorf("unknown method: %s", method))
	}
}

//
// ------------------------------------------------------------
// RPC SERVER STARTUP
// ------------------------------------------------------------
//

func StartRPCServer(port int, server *Server) {
	mux := http.NewServeMux()

	// JSON-RPC
	mux.HandleFunc("/", server.HandleJSONRPC)

	// REST endpoint (admin dashboard)
	mux.HandleFunc("/payments/merchant", server.handleGetMerchantPayments)

	addr := fmt.Sprintf(":%d", port)
	fmt.Println("[RPC] Listening on", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println("[RPC] Server error:", err)
	}
}

//
// ------------------------------------------------------------
// REST ENDPOINT: LIST PAYMENTS FOR MERCHANT
// ------------------------------------------------------------
//

func (s *Server) handleGetMerchantPayments(w http.ResponseWriter, r *http.Request) {
	merchant := r.URL.Query().Get("merchant")

	if merchant == "" {
		http.Error(w, "missing merchant", http.StatusBadRequest)
		return
	}

	payments := s.bc.State.PaymentGateway.ListMerchantPayments(
		common.HexToAddress(merchant),
	)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":  true,
		"payments": payments,
	})
}

//
// ------------------------------------------------------------
// RPC METHOD: SEND NATIVE TOKEN
// ------------------------------------------------------------
//

func HandleSendNative(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) != 1 {
		return nil, errors.New("invalid params")
	}

	raw, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, errors.New("invalid param format")
	}

	fromStr := raw["from"].(string)
	toStr := raw["to"].(string)
	amountFloat := raw["amount"].(float64)

	from := common.HexToAddress(fromStr)
	to := common.HexToAddress(toStr)

	// convert float → big.Int (ether → wei)
	amountWei := new(big.Int)
	amountWei.Mul(big.NewInt(int64(amountFloat*1e6)), big.NewInt(1e12))

	txHash, err := bc.State.PaymentGateway.SendNative(from, to, amountWei)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"txHash": txHash.Hex(),
	}, nil
}

//
// ------------------------------------------------------------
// PAYMENT INTENT HANDLERS
// ------------------------------------------------------------
//

// creates payment intent
func HandleCreatePaymentIntent(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	raw := params[0].(map[string]interface{})
	merchant := common.HexToAddress(raw["merchant"].(string))
	amount := raw["amount"].(float64)
	token := raw["token"].(string)

	intent, err := bc.State.PaymentGateway.CreateIntent(merchant, amount, token)
	if err != nil {
		return nil, err
	}
	return intent, nil
}

// returns single intent
func HandleGetPaymentIntent(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	id := int(params[0].(float64))
	intent, err := bc.State.PaymentGateway.GetIntent(id)
	if err != nil {
		return nil, err
	}
	return intent, nil
}

// returns all merchant payments
func HandleListMerchantPayments(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	merchant := common.HexToAddress(params[0].(string))
	list := bc.State.PaymentGateway.ListMerchantPayments(merchant)
	return list, nil
}
