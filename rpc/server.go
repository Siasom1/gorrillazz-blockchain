package rpc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strconv"

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

// ------------------------------------------------------------
// JSON RESPONSE HELPERS
// ------------------------------------------------------------

type rpcReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

func writeResult(w http.ResponseWriter, id interface{}, result interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  result,
		"id":      id,
	})
}

func writeRPCError(w http.ResponseWriter, id interface{}, code int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    code,
			"message": msg,
		},
		"id": id,
	})
}

// writeJSON writes either a successful result or an RPC error using the provided id.
func writeJSON(w http.ResponseWriter, result interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -32000,
				"message": err.Error(),
			},
			"id": nil,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  result,
		"id":      1,
	})
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, nil, err)
}

// ------------------------------------------------------------
// START SERVER (JSON-RPC + REST)
// ------------------------------------------------------------

func StartRPCServer(port int, server *Server) {
	mux := http.NewServeMux()

	// JSON-RPC endpoint
	mux.HandleFunc("/", server.HandleJSONRPC)

	// REST endpoint for Admin dashboard
	mux.HandleFunc("/payments/merchant", server.handleGetMerchantPayments)

	addr := fmt.Sprintf(":%d", port)
	fmt.Println("[RPC] Listening on", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println("[RPC] Server error:", err)
	}
}

// ------------------------------------------------------------
// REST: /payments/merchant?merchant=0x...
// ------------------------------------------------------------

func (s *Server) handleGetMerchantPayments(w http.ResponseWriter, r *http.Request) {
	merchant := r.URL.Query().Get("merchant")
	if merchant == "" {
		http.Error(w, "missing merchant", http.StatusBadRequest)
		return
	}
	payments := s.bc.Payment.ListMerchantPayments(common.HexToAddress(merchant))
	writeResult(w, nil, payments)
}

func (s *Server) HandleJSONRPC(w http.ResponseWriter, r *http.Request) {
	var req struct {
		JSONRPC string        `json:"jsonrpc"`
		Method  string        `json:"method"`
		Params  []interface{} `json:"params"`
		ID      interface{}   `json:"id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, err)
		return
	}

	switch req.Method {

	case "gorr_getSystemWallets":
		result, err := HandleGetSystemWallets(s.bc, req.Params)
		writeJSON(w, result, err)

	case "gorr_getBalance":
		result, err := HandleGetBalance(s.bc, req.Params)
		writeJSON(w, result, err)

	case "gorr_getUSDCcBalance":
		result, err := HandleGetUSDCcBalance(s.bc, req.Params)
		writeJSON(w, result, err)

	case "gorr_sendTransaction":
		result, err := HandleSendNative(s.bc, req.Params)
		writeJSON(w, result, err)

	default:
		writeError(w, fmt.Errorf("method not found: %s", req.Method))
	}
}

// ------------------------------------------------------------
// PAYMENT INTENT METHODS (matchen met CreateIntent signature)
// ------------------------------------------------------------

func HandleCreatePaymentIntent(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing params")
	}

	raw, ok := params[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid params[0], expected object")
	}

	merchantStr, _ := raw["merchant"].(string)
	if merchantStr == "" {
		return nil, fmt.Errorf("missing merchant")
	}
	merchant := common.HexToAddress(merchantStr)

	token, _ := raw["token"].(string)
	if token == "" {
		token = "GORR"
	}

	// amount kan number of string zijn
	amount, err := parseBigInt(raw["amount"])
	if err != nil {
		return nil, fmt.Errorf("invalid amount: %w", err)
	}

	// expiry optional: seconds
	expiry := uint64(900) // default 15 min
	if v, exists := raw["expiry"]; exists {
		exp, err := parseUint64(v)
		if err != nil {
			return nil, fmt.Errorf("invalid expiry: %w", err)
		}
		expiry = exp
	}

	intent, blockNumber, err := bc.Payment.CreateIntent(merchant, amount, token, expiry)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"intent":      intent,
		"blockNumber": blockNumber,
	}, nil
}

func HandleGetPaymentIntent(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing id param")
	}
	id, err := parseUint64(params[0])
	if err != nil {
		return nil, fmt.Errorf("invalid id: %w", err)
	}

	intent, err := bc.Payment.GetIntent(id)
	if err != nil {
		return nil, err
	}
	return intent, nil
}

func HandleListMerchantPayments(bc *blockchain.Blockchain, params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, fmt.Errorf("missing merchant param")
	}
	merchantStr, ok := params[0].(string)
	if !ok || merchantStr == "" {
		return nil, fmt.Errorf("invalid merchant param")
	}
	return bc.Payment.ListMerchantPayments(common.HexToAddress(merchantStr)), nil
}

// ------------------------------------------------------------
// HELPERS: parse *big.Int / uint64 from JSON decoded types
// ------------------------------------------------------------

// parseBigInt accepteert:
// - JSON number (float64) -> wordt int64 afgerond
// - string ("12345")
func parseBigInt(v interface{}) (*big.Int, error) {
	switch t := v.(type) {
	case float64:
		// Let op: JSON numbers worden float64, dus dit is whole-number only
		return big.NewInt(int64(t)), nil
	case string:
		if t == "" {
			return nil, fmt.Errorf("empty string")
		}
		n := new(big.Int)
		if _, ok := n.SetString(t, 10); !ok {
			return nil, fmt.Errorf("cannot parse %q", t)
		}
		return n, nil
	case json.Number:
		n := new(big.Int)
		if _, ok := n.SetString(t.String(), 10); !ok {
			return nil, fmt.Errorf("cannot parse %q", t.String())
		}
		return n, nil
	default:
		return nil, fmt.Errorf("unsupported type %T", v)
	}
}

func parseUint64(v interface{}) (uint64, error) {
	switch t := v.(type) {
	case float64:
		if t < 0 {
			return 0, fmt.Errorf("negative")
		}
		return uint64(t), nil
	case string:
		if t == "" {
			return 0, fmt.Errorf("empty string")
		}
		return strconv.ParseUint(t, 10, 64)
	case json.Number:
		return strconv.ParseUint(t.String(), 10, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", v)
	}
}
