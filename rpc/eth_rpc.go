package rpc

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/ethereum/go-ethereum/common"
	gethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
)

//
// ------------------------------------------------------------
// Ethereum JSON-RPC (LEGACY, MINIMAL) — D.5.2
// ------------------------------------------------------------
// ✅ Accept legacy signed tx (RLP), recover sender, apply state transfer,
// and return txHash + receipt immediately (dev-mode).
//

type ethRPC struct {
	bc      *blockchain.Blockchain
	chainID uint64

	mu     sync.Mutex
	nonces map[common.Address]uint64
	txs    map[common.Hash]*ethTxRecord
}

type ethTxRecord struct {
	Hash        common.Hash
	From        common.Address
	To          *common.Address
	ValueWei    *big.Int
	Nonce       uint64
	BlockNumber uint64
	Time        uint64
	Status      uint64 // 1 success, 0 fail
}

func newEthRPC(bc *blockchain.Blockchain) *ethRPC {
	return &ethRPC{
		bc:      bc,
		chainID: bc.NetworkID(),
		nonces:  make(map[common.Address]uint64),
		txs:     make(map[common.Hash]*ethTxRecord),
	}
}

//
// ------------------------------------------------------------
// ETH RPC ROUTER
// ------------------------------------------------------------
// Server.go calls: HandleEthRPC(w, req, s.eth)
//

func HandleEthRPC(w http.ResponseWriter, req rpcReq, eth *ethRPC) {
	switch req.Method {

	case "eth_chainId":
		writeJSON(w, req.ID, fmt.Sprintf("0x%x", eth.chainID), nil)

	case "net_version":
		writeJSON(w, req.ID, fmt.Sprintf("%d", eth.chainID), nil)

	case "web3_clientVersion":
		writeJSON(w, req.ID, "Gorrillazz/v0.5.2", nil)

	case "eth_blockNumber":
		head := eth.bc.Head()
		if head == nil || head.Header == nil {
			writeJSON(w, req.ID, "0x0", nil)
			return
		}
		writeJSON(w, req.ID, fmt.Sprintf("0x%x", head.Header.Number), nil)

	case "eth_getBalance":
		// params: [address, "latest"]
		if len(req.Params) < 1 {
			writeJSON(w, req.ID, nil, fmt.Errorf("missing address"))
			return
		}
		addrHex, ok := req.Params[0].(string)
		if !ok {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid address"))
			return
		}
		addr := common.HexToAddress(addrHex)

		// Your State.GetBalance returns 2 values in your project.
		bal, _ := eth.bc.State.GetBalance(addr)
		if bal == nil {
			bal = big.NewInt(0)
		}
		writeJSON(w, req.ID, "0x"+bal.Text(16), nil)

	case "eth_getTransactionCount":
		// params: [address, "latest" | "pending"]
		if len(req.Params) < 1 {
			writeJSON(w, req.ID, nil, fmt.Errorf("missing address"))
			return
		}

		addrHex, ok := req.Params[0].(string)
		if !ok {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid address"))
			return
		}

		addr := common.HexToAddress(addrHex)

		eth.mu.Lock()
		nonce := eth.nonces[addr]
		eth.mu.Unlock()

		writeJSON(w, req.ID, fmt.Sprintf("0x%x", nonce), nil)

	case "eth_gasPrice":
		// dev: free tx
		writeJSON(w, req.ID, "0x0", nil)

	case "eth_estimateGas":
		// legacy transfer gas
		writeJSON(w, req.ID, "0x5208", nil)

	case "eth_sendRawTransaction":
		// params: ["0x...rawRLP..."]
		if len(req.Params) < 1 {
			writeJSON(w, req.ID, nil, fmt.Errorf("missing raw tx"))
			return
		}
		rawHex, ok := req.Params[0].(string)
		if !ok {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid raw tx"))
			return
		}

		tx, from, err := decodeAndRecoverLegacyTx(rawHex, eth.chainID)
		if err != nil {
			writeJSON(w, req.ID, nil, err)
			return
		}

		// nonce check (dev)
		eth.mu.Lock()
		expected := eth.nonces[from]
		eth.mu.Unlock()

		if tx.Nonce() != expected {
			writeJSON(w, req.ID, nil, fmt.Errorf("bad nonce: got %d want %d", tx.Nonce(), expected))
			return
		}

		// basic checks
		to := tx.To()
		if to == nil || *to == (common.Address{}) {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid to address"))
			return
		}

		value := tx.Value()
		if value == nil || value.Sign() <= 0 {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid amount"))
			return
		}

		// Apply state (GORR native in wei)
		if err := eth.bc.State.SubBalance(from, value); err != nil {
			writeJSON(w, req.ID, nil, err)
			return
		}
		eth.bc.State.AddBalance(*to, value)

		// store record + bump nonce
		txHash := tx.Hash()
		head := eth.bc.Head()
		var bn uint64
		if head != nil && head.Header != nil {
			bn = head.Header.Number
		}

		rec := &ethTxRecord{
			Hash:        txHash,
			From:        from,
			To:          to,
			ValueWei:    new(big.Int).Set(value),
			Nonce:       tx.Nonce(),
			BlockNumber: bn, // dev: treat as mined "now"
			Time:        uint64(time.Now().Unix()),
			Status:      1,
		}

		eth.mu.Lock()
		eth.txs[txHash] = rec
		eth.nonces[from] = expected + 1
		eth.mu.Unlock()

		// Return tx hash
		writeJSON(w, req.ID, txHash.Hex(), nil)

	case "eth_getTransactionByHash":
		if len(req.Params) < 1 {
			writeJSON(w, req.ID, nil, fmt.Errorf("missing tx hash"))
			return
		}
		hx, ok := req.Params[0].(string)
		if !ok {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid tx hash"))
			return
		}
		h := common.HexToHash(hx)

		eth.mu.Lock()
		rec := eth.txs[h]
		eth.mu.Unlock()

		if rec == nil {
			writeJSON(w, req.ID, nil, nil) // JSON-RPC expects null when not found
			return
		}

		toHex := "0x"
		if rec.To != nil {
			toHex = rec.To.Hex()
		}

		// Minimal tx object for MetaMask/dev tooling
		writeJSON(w, req.ID, map[string]interface{}{
			"hash":             rec.Hash.Hex(),
			"nonce":            fmt.Sprintf("0x%x", rec.Nonce),
			"from":             rec.From.Hex(),
			"to":               toHex,
			"value":            "0x" + rec.ValueWei.Text(16),
			"blockNumber":      fmt.Sprintf("0x%x", rec.BlockNumber),
			"transactionIndex": "0x0",
		}, nil)

	case "eth_getTransactionReceipt":
		if len(req.Params) < 1 {
			writeJSON(w, req.ID, nil, fmt.Errorf("missing tx hash"))
			return
		}
		hx, ok := req.Params[0].(string)
		if !ok {
			writeJSON(w, req.ID, nil, fmt.Errorf("invalid tx hash"))
			return
		}
		h := common.HexToHash(hx)

		eth.mu.Lock()
		rec := eth.txs[h]
		eth.mu.Unlock()

		if rec == nil {
			writeJSON(w, req.ID, nil, nil)
			return
		}

		toHex := "0x"
		if rec.To != nil {
			toHex = rec.To.Hex()
		}

		// Minimal receipt
		writeJSON(w, req.ID, map[string]interface{}{
			"transactionHash":   rec.Hash.Hex(),
			"transactionIndex":  "0x0",
			"blockNumber":       fmt.Sprintf("0x%x", rec.BlockNumber),
			"blockHash":         "0x" + strings.Repeat("0", 64), // dev placeholder
			"from":              rec.From.Hex(),
			"to":                toHex,
			"cumulativeGasUsed": "0x0",
			"gasUsed":           "0x0",
			"effectiveGasPrice": "0x0",
			"status":            fmt.Sprintf("0x%x", rec.Status),
			"logs":              []interface{}{},
			"logsBloom":         "0x" + strings.Repeat("0", 512),
		}, nil)

	default:
		writeJSON(w, req.ID, nil, fmt.Errorf("unsupported eth method: %s", req.Method))
	}
}

//
// ------------------------------------------------------------
// Helpers
// ------------------------------------------------------------

func decodeAndRecoverLegacyTx(rawHex string, chainID uint64) (*gethtypes.Transaction, common.Address, error) {
	rawHex = strings.TrimPrefix(rawHex, "0x")
	b, err := hex.DecodeString(rawHex)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("invalid hex: %w", err)
	}

	// Legacy tx RLP decode
	var tx gethtypes.Transaction
	if err := rlp.DecodeBytes(b, &tx); err != nil {
		return nil, common.Address{}, fmt.Errorf("rlp decode failed: %w", err)
	}

	// EIP-155 signer (legacy + chainId)
	signer := gethtypes.NewEIP155Signer(new(big.Int).SetUint64(chainID))

	from, err := gethtypes.Sender(signer, &tx)
	if err != nil {
		return nil, common.Address{}, fmt.Errorf("signature recover failed: %w", err)
	}

	// ChainId sanity check (MetaMask always sets it)
	if tx.ChainId() != nil && tx.ChainId().Uint64() != chainID {
		return nil, common.Address{}, fmt.Errorf("wrong chainId: got %d want %d", tx.ChainId().Uint64(), chainID)
	}

	return &tx, from, nil
}
