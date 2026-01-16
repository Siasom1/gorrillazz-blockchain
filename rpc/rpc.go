package rpc

import (
	"encoding/json"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/txpool"
	"github.com/ethereum/go-ethereum/common"
)

type RPCServer struct {
	Chain  *blockchain.Blockchain
	TxPool *txpool.TxPool
}

func NewRPCServer(chain *blockchain.Blockchain, pool *txpool.TxPool) *RPCServer {
	return &RPCServer{
		Chain:  chain,
		TxPool: pool,
	}
}

type RPCRequest struct {
	Jsonrpc string        `json:"jsonrpc"`
	Method  string        `json:"method"`
	Params  []interface{} `json:"params"`
	ID      int           `json:"id"`
}

type RPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   string      `json:"error,omitempty"`
	ID      int         `json:"id"`
}

func (s *RPCServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req RPCRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: "invalid request", ID: 0})
		return
	}

	switch req.Method {
	case "eth_getBalance":
		params := req.Params
		if len(params) < 1 {
			writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: "missing params", ID: req.ID})
			return
		}
		addr := common.HexToAddress(params[0].(string))
		bal, _ := s.Chain.State.GetBalance(addr)
		writeJSON(w, RPCResponse{Jsonrpc: "2.0", Result: bal.String(), ID: req.ID})

	case "eth_sendRawTransaction":
		params := req.Params
		if len(params) < 1 {
			writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: "missing raw tx", ID: req.ID})
			return
		}
		raw := params[0].(string)
		tx, err := s.Chain.ParseRawTransaction(raw)
		if err != nil {
			writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: err.Error(), ID: req.ID})
			return
		}
		if err := s.TxPool.Add(tx); err != nil {
			writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: err.Error(), ID: req.ID})
			return
		}
		writeJSON(w, RPCResponse{Jsonrpc: "2.0", Result: tx.Hash.Hex(), ID: req.ID})

	default:
		writeJSON(w, RPCResponse{Jsonrpc: "2.0", Error: "method not found", ID: req.ID})
	}
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
