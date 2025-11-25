package rpc

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/core/types"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type RPCHandlers struct {
	Chain *blockchain.Blockchain
}

// ---------------------------------------------------------------
// REGISTER METHODS
// ---------------------------------------------------------------
func NewHandlers(chain *blockchain.Blockchain) map[string]RPCHandler {
	h := &RPCHandlers{Chain: chain}

	return map[string]RPCHandler{
		// ---------------- GORR RPC ----------------
		"gorr_blockNumber":      h.blockNumber,
		"gorr_getHead":          h.getHead,
		"gorr_getBlockByNumber": h.getBlockByNumber,

		// --------- ETHEREUM COMPATIBLE RPC ---------
		"eth_chainId":               h.ethChainId,
		"net_version":               h.netVersion,
		"eth_blockNumber":           h.ethBlockNumber,
		"eth_getBlockByNumber":      h.ethGetBlockByNumber,
		"eth_getBalance":            h.ethGetBalance,
		"eth_getTransactionCount":   h.ethGetTransactionCount,
		"eth_sendRawTransaction":    h.ethSendRawTransaction,
		"eth_getTransactionReceipt": h.ethGetTransactionReceipt,
		"eth_getTransactionByHash":  h.ethGetTransactionByHash,
	}
}

// ---------------------------------------------------------------
// GORR RPC IMPLEMENTATION
// ---------------------------------------------------------------
func (h *RPCHandlers) blockNumber(params []interface{}) (interface{}, error) {
	return h.Chain.Head().Header.Number, nil
}

func (h *RPCHandlers) getHead(params []interface{}) (interface{}, error) {
	return h.Chain.Head(), nil
}

func (h *RPCHandlers) getBlockByNumber(params []interface{}) (interface{}, error) {
	if len(params) == 0 {
		return nil, errors.New("missing block number")
	}

	numFloat, ok := params[0].(float64)
	if !ok {
		return nil, errors.New("block number must be float")
	}

	n := uint64(numFloat)
	return h.Chain.LoadBlock(n)
}

// ---------------------------------------------------------------
// ETHEREUM COMPATIBLE RPC IMPLEMENTATION
// ---------------------------------------------------------------
func (h *RPCHandlers) ethChainId(params []interface{}) (interface{}, error) {
	return fmt.Sprintf("0x%x", h.Chain.NetworkID()), nil
}

func (h *RPCHandlers) netVersion(params []interface{}) (interface{}, error) {
	return fmt.Sprintf("%d", h.Chain.NetworkID()), nil
}

func (h *RPCHandlers) ethBlockNumber(params []interface{}) (interface{}, error) {
	num := h.Chain.Head().Header.Number
	return fmt.Sprintf("0x%x", num), nil
}

func (h *RPCHandlers) ethGetBlockByNumber(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing block number")
	}

	raw := params[0].(string)

	if raw == "latest" {
		return h.Chain.Head(), nil
	}

	if len(raw) > 2 && raw[:2] == "0x" {
		n, err := strconv.ParseUint(raw[2:], 16, 64)
		if err != nil {
			return nil, err
		}
		return h.Chain.LoadBlock(n)
	}

	return nil, errors.New("invalid block number format")
}

func (h *RPCHandlers) ethGetTransactionReceipt(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing tx hash")
	}

	txHash := common.HexToHash(params[0].(string))

	blockNum, err := h.Chain.FindTxBlock(txHash)
	if err != nil {
		return nil, err
	}

	receipts, err := h.Chain.LoadReceipts(blockNum)
	if err != nil {
		return nil, err
	}

	for _, r := range receipts {
		if r.TxHash == txHash {
			return r, nil
		}
	}

	return nil, errors.New("receipt not found")
}

func (h *RPCHandlers) ethGetTransactionByHash(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing tx hash")
	}

	txHash := common.HexToHash(params[0].(string))

	blockNum, err := h.Chain.FindTxBlock(txHash)
	if err != nil {
		return nil, err
	}

	block, err := h.Chain.LoadBlock(blockNum)
	if err != nil {
		return nil, err
	}

	for idx, tx := range block.Transactions {
		if tx.Hash() == txHash {
			from, _ := tx.From()
			result := map[string]interface{}{
				"hash":             tx.Hash().Hex(),
				"nonce":            fmt.Sprintf("0x%x", tx.Nonce),
				"blockNumber":      fmt.Sprintf("0x%x", blockNum),
				"transactionIndex": fmt.Sprintf("0x%x", idx),
				"from":             from.Hex(),
				"to":               tx.To.Hex(),
				"value":            fmt.Sprintf("0x%x", tx.Value),
				"gas":              fmt.Sprintf("0x%x", tx.Gas),
				"gasPrice":         fmt.Sprintf("0x%x", tx.GasPrice),
			}
			return result, nil
		}
	}

	return nil, errors.New("tx not found")
}

// ------------------------------
// BALANCE
// ------------------------------
func (h *RPCHandlers) ethGetBalance(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addr := common.HexToAddress(params[0].(string))

	bal, err := h.Chain.State.GetBalance(addr)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("0x%x", bal), nil
}

// ------------------------------
// NONCE
// ------------------------------
func (h *RPCHandlers) ethGetTransactionCount(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing address")
	}

	addr := common.HexToAddress(params[0].(string))
	nonce, err := h.Chain.State.GetNonce(addr)
	if err != nil {
		return nil, err
	}

	return fmt.Sprintf("0x%x", nonce), nil
}

// ---------------------------------------------------------------
// eth_sendRawTransaction — Accept signed RLP encoded tx
// ---------------------------------------------------------------
func (h *RPCHandlers) ethSendRawTransaction(params []interface{}) (interface{}, error) {
	if len(params) < 1 {
		return nil, errors.New("missing raw transaction data")
	}

	rawHex, ok := params[0].(string)
	if !ok {
		return nil, errors.New("raw transaction must be a hex string")
	}

	rawBytes, err := hexutil.Decode(rawHex)
	if err != nil {
		return nil, err
	}

	tx, err := types.DecodeTx(rawBytes)
	if err != nil {
		return nil, err
	}

	// VALIDATE TX
	if err := tx.Validate(); err != nil {
		return nil, err
	}

	// ADD TO TX POOL
	if err := h.Chain.TxPool.Add(tx); err != nil {
		return nil, err
	}

	// RETURN TX HASH
	return tx.Hash().Hex(), nil
}
