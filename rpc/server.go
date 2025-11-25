package rpc

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
)

type RPCServer struct {
	addr     string
	handlers map[string]RPCHandler
	server   *http.Server
}

type RPCHandler func(params []interface{}) (interface{}, error)

type request struct {
	JSONRPC string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type response struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   interface{} `json:"error,omitempty"`
	ID      int         `json:"id"`
}

func NewRPCServer(host string, handlers map[string]RPCHandler) *RPCServer {
	return &RPCServer{
		addr:     host,
		handlers: handlers,
	}
}

func (s *RPCServer) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleRequest)

	s.server = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	go func() {
		log.Println("[RPC] Started JSON-RPC server at", s.addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Println("[RPC] Error:", err)
		}
	}()

	return nil
}

func (s *RPCServer) Stop() {
	if s.server != nil {
		s.server.Close()
	}
}

func (s *RPCServer) handleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}

	var req request
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	handler, ok := s.handlers[req.Method]
	if !ok {
		writeError(w, req.ID, "method not found")
		return
	}

	result, err := handler(req.Params)
	if err != nil {
		writeError(w, req.ID, err.Error())
		return
	}

	writeResult(w, req.ID, result)
}

func writeResult(w http.ResponseWriter, id int, result interface{}) {
	resp := response{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
	bytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}

func writeError(w http.ResponseWriter, id int, err string) {
	resp := response{
		JSONRPC: "2.0",
		Error:   err,
		ID:      id,
	}
	bytes, _ := json.Marshal(resp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(bytes)
}
