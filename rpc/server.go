package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
)

// RPCHandler is a function that handles a single RPC method.
type RPCHandler func(params []interface{}) (interface{}, error)

type RPCServer struct {
	handlers map[string]RPCHandler
}

// StartRPCServer starts the JSON-RPC server on a given port.
func StartRPCServer(port int, chain *blockchain.Blockchain) {
	handlers := NewHandlers(chain)

	log.Println("--------- RPC HANDLERS REGISTERED ---------")
	for name := range handlers {
		log.Println("RPC method available:", name)
	}
	log.Println("-------------------------------------------")

	s := &RPCServer{
		handlers: handlers,
	}

	http.HandleFunc("/", s.handle)

	addr := fmt.Sprintf(":%d", port)
	log.Printf("[RPC] Started JSON-RPC server at %s\n", addr)
	go http.ListenAndServe(addr, nil)
}

type rpcRequest struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      interface{}   `json:"id"`
}

type rpcResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

func (s *RPCServer) handle(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeRPCError(w, nil, "invalid body")
		return
	}

	var req rpcRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeRPCError(w, nil, "invalid JSON")
		return
	}

	// LOGGING: see what the client calls
	log.Println("[RPC] Incoming method:", req.Method)

	handler, ok := s.handlers[req.Method]
	if !ok {
		log.Println("[RPC] Unknown method:", req.Method)
		writeRPCError(w, req.ID, "method not found")
		return
	}

	result, err := handler(req.Params)
	if err != nil {
		writeRPCError(w, req.ID, err.Error())
		return
	}

	resp := rpcResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}

	writeJSON(w, resp)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeRPCError(w http.ResponseWriter, id interface{}, msg string) {
	resp := rpcResponse{
		JSONRPC: "2.0",
		Error:   msg,
		ID:      id,
	}
	writeJSON(w, resp)
}
