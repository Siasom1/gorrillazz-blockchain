package rpc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/ethereum/go-ethereum/common"
)

//
// ------------------------------------------------------------
// SERVER
// ------------------------------------------------------------
//

type Server struct {
	bc  *blockchain.Blockchain
	bus *events.EventBus
	eth *ethRPC
}

func NewServer(bc *blockchain.Blockchain, bus *events.EventBus) *Server {
	return &Server{
		bc:  bc,
		bus: bus,
		eth: newEthRPC(bc),
	}
}

//
// ------------------------------------------------------------
// JSON-RPC TYPES
// ------------------------------------------------------------
//

type rpcReq struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

//
// ------------------------------------------------------------
// JSON-RPC RESPONSE HELPERS
// ------------------------------------------------------------
//

func writeJSON(w http.ResponseWriter, id interface{}, result interface{}, err error) {
	w.Header().Set("Content-Type", "application/json")

	if err != nil {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0",
			"error": map[string]interface{}{
				"code":    -32000,
				"message": err.Error(),
			},
			"id": id,
		})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"jsonrpc": "2.0",
		"result":  result,
		"id":      id,
	})
}

//
// ------------------------------------------------------------
// SERVER START (JSON-RPC + REST + WS)
// ------------------------------------------------------------
//

func StartRPCServer(port int, server *Server) {
	mux := http.NewServeMux()

	// JSON-RPC
	mux.HandleFunc("/", server.HandleJSONRPC)

	// REST
	mux.HandleFunc("/payments/merchant", server.handleGetMerchantPayments)

	// WebSocket (D.4.2)
	mux.HandleFunc("/ws", WSHandler(server.bus))

	addr := fmt.Sprintf(":%d", port)
	fmt.Println("[RPC] Listening on", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		fmt.Println("[RPC] Server error:", err)
	}
}

//
// ------------------------------------------------------------
// REST: /payments/merchant?merchant=0x...
// ------------------------------------------------------------
//

func (s *Server) handleGetMerchantPayments(w http.ResponseWriter, r *http.Request) {
	merchant := r.URL.Query().Get("merchant")
	if merchant == "" {
		http.Error(w, "missing merchant", http.StatusBadRequest)
		return
	}

	payments := s.bc.Payment.ListMerchantPayments(common.HexToAddress(merchant))
	writeJSON(w, nil, payments, nil)
}

//
// ------------------------------------------------------------
// JSON-RPC ROUTER
// ------------------------------------------------------------
//

func (s *Server) HandleJSONRPC(w http.ResponseWriter, r *http.Request) {
	var req rpcReq

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, nil, nil, err)
		return
	}

	// ------------------------------------------------------------
	// Ethereum JSON-RPC routing (D.5.1)
	// ------------------------------------------------------------

	// ------------------------------------------------------------
	// Ethereum JSON-RPC routing (D.5.1)
	// ------------------------------------------------------------
	if len(req.Method) >= 4 &&
		(req.Method[:4] == "eth_" ||
			req.Method == "net_version" ||
			req.Method == "web3_clientVersion") {

		HandleEthRPC(w, req, s.eth)
		return
	}

	switch req.Method {

	// -------- SYSTEM --------

	case "gorr_getSystemWallets":
		res, err := HandleGetSystemWallets(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	// -------- BALANCES --------

	case "gorr_getBalance":
		res, err := HandleGetBalance(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	case "gorr_getUSDCcBalance":
		res, err := HandleGetUSDCcBalance(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	// -------- TRANSFERS --------

	case "gorr_sendTransaction":
		res, err := HandleSendNative(s.bc, req.Params)
		if err == nil {
			s.bus.EmitTx(res)
		}
		writeJSON(w, req.ID, res, err)

	case "gorr_sendUSDCc":
		res, err := HandleSendUSDCc(s.bc, req.Params)
		if err == nil {
			s.bus.EmitTx(res)
		}
		writeJSON(w, req.ID, res, err)

	// -------- ADMIN --------

	case "gorr_adminMint":
		res, err := HandleAdminMint(s.bc, req.Params)
		if err == nil {
			s.bus.EmitTx(res)
		}
		writeJSON(w, req.ID, res, err)

	case "gorr_adminBurn":
		res, err := HandleAdminBurn(s.bc, req.Params)
		if err == nil {
			s.bus.EmitTx(res)
		}
		writeJSON(w, req.ID, res, err)

	case "gorr_adminSetFees":
		res, err := HandleSetFees(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	case "gorr_adminPauseTransfers":
		res, err := HandlePauseTransfers(s.bc, req.Params)
		if err == nil {
			s.bus.EmitBlock(map[string]interface{}{
				"type":   "admin.pause",
				"paused": res,
			})
		}
		writeJSON(w, req.ID, res, err)

	case "gorr_adminForceTransfer":
		res, err := HandleAdminForceTransfer(s.bc, req.Params)
		if err == nil {
			s.bus.EmitBlock(map[string]interface{}{
				"type":   "admin.pause",
				"paused": res,
			})
		}
		writeJSON(w, req.ID, res, err)

	case "gorr_adminMintToTreasury":
		res, err := HandleAdminMintToTreasury(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	case "gorr_adminWithdrawFees":
		res, err := HandleAdminWithdrawFees(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	case "gorr_adminStats":
		res, err := HandleAdminStats(s.bc, req.Params)
		writeJSON(w, req.ID, res, err)

	// -------- FALLBACK --------

	default:
		writeJSON(
			w,
			req.ID,
			nil,
			fmt.Errorf("method not found: %s", req.Method),
		)
	}
}

//
// ------------------------------------------------------------
// HELPERS (used by methods.go)
// ------------------------------------------------------------
//

// parseBigInt:
// - float64 (JSON number)
// - string ("123")

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
