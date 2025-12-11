package rpc

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
)

type RPCHandler func(params []interface{}) (interface{}, error)

type RPCServer struct {
	handlers map[string]RPCHandler
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

func StartRPCServer(port int, chain *blockchain.Blockchain) {
	s := &RPCServer{
		handlers: NewHandlers(chain),
	}

	http.HandleFunc("/", s.handle)

	addr := fmt.Sprintf(":%d", port)
	log.Println("[RPC] Started JSON-RPC server at", addr)

	go http.ListenAndServe(addr, nil)
}

func (s *RPCServer) handle(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeRPCError(w, nil, "invalid body")
		return
	}

	log.Println("[RPC] RAW REQUEST:", string(bodyBytes))

	// Detect batch
	if len(bodyBytes) > 0 && bodyBytes[0] == '[' {
		var batch []rpcRequest
		if err := json.Unmarshal(bodyBytes, &batch); err != nil {
			writeRPCError(w, nil, "invalid batch JSON")
			return
		}

		responses := make([]rpcResponse, 0, len(batch))
		for _, req := range batch {
			log.Println("[RPC] Incoming method:", req.Method)
			responses = append(responses, s.call(req))
		}

		writeJSON(w, responses)
		return
	}

	// single request
	var req rpcRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		writeRPCError(w, nil, "invalid JSON")
		return
	}

	log.Println("[RPC] Incoming method:", req.Method)
	resp := s.call(req)
	writeJSON(w, resp)
}

func (s *RPCServer) call(req rpcRequest) rpcResponse {
	handler, ok := s.handlers[req.Method]
	if !ok {
		return rpcResponse{
			JSONRPC: "2.0",
			Error:   "method not found",
			ID:      req.ID,
		}
	}

	result, err := handler(req.Params)
	if err != nil {
		return rpcResponse{
			JSONRPC: "2.0",
			Error:   err.Error(),
			ID:      req.ID,
		}
	}

	return rpcResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      req.ID,
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func writeRPCError(w http.ResponseWriter, id interface{}, msg string) {
	writeJSON(w, rpcResponse{
		JSONRPC: "2.0",
		Error:   msg,
		ID:      id,
	})
}
