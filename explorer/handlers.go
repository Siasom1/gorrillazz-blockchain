package explorer

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

// Utility response
func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// ------------------------------------------------------------
// 1. /explorer/latest-blocks
// ------------------------------------------------------------
func (api *ExplorerAPI) handleLatestBlocks(w http.ResponseWriter, r *http.Request) {
	head := api.Chain.Head()

	out := []interface{}{}
	for i := 0; i < 10; i++ {
		if head.Header.Number < uint64(i) {
			break
		}
		block, _ := api.Chain.LoadBlock(head.Header.Number - uint64(i))
		out = append(out, block)
	}

	writeJSON(w, out)
}

// ------------------------------------------------------------
// 2. /explorer/block/{number}
// ------------------------------------------------------------
func (api *ExplorerAPI) handleBlockByNumber(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		writeJSON(w, "invalid URL")
		return
	}

	number, err := strconv.ParseUint(parts[3], 10, 64)
	if err != nil {
		writeJSON(w, "invalid block number")
		return
	}

	block, err := api.Chain.LoadBlock(number)
	if err != nil {
		writeJSON(w, "block not found")
		return
	}

	writeJSON(w, block)
}

// ------------------------------------------------------------
// 3. /explorer/tx/{hash}
// ------------------------------------------------------------
func (api *ExplorerAPI) handleTransaction(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		writeJSON(w, "invalid URL")
		return
	}

	txHash := common.HexToHash(parts[3])

	// find block number
	blockNum, err := api.Chain.FindTxBlock(txHash)
	if err != nil {
		writeJSON(w, "tx not found")
		return
	}

	// load block
	block, err := api.Chain.LoadBlock(blockNum)
	if err != nil {
		writeJSON(w, "block not found")
		return
	}

	// find tx inside block
	for idx, tx := range block.Transactions {
		if tx.Hash() == txHash {

			receiptList, _ := api.Chain.LoadReceipts(blockNum)
			var receipt interface{}
			for _, r := range receiptList {
				if r.TxHash == txHash {
					receipt = r
					break
				}
			}

			result := map[string]interface{}{
				"transaction": tx,
				"blockNumber": blockNum,
				"index":       idx,
				"receipt":     receipt,
			}

			writeJSON(w, result)
			return
		}
	}

	writeJSON(w, "tx not found")
}

// ------------------------------------------------------------
// 4. /explorer/address/{address}/txs
// ------------------------------------------------------------
func (api *ExplorerAPI) handleAddressTxs(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) < 4 {
		writeJSON(w, "invalid URL")
		return
	}

	addr := common.HexToAddress(parts[3])

	// iterate last 1000 blocks
	head := api.Chain.Head()
	maxBlocks := uint64(1000)

	var out []interface{}

	for i := uint64(0); i < maxBlocks; i++ {
		if head.Header.Number < i {
			break
		}

		blockNum := head.Header.Number - i
		block, _ := api.Chain.LoadBlock(blockNum)

		for _, tx := range block.Transactions {
			from, _ := tx.From()
			if from == addr || (tx.To != nil && *tx.To == addr) {
				out = append(out, map[string]interface{}{
					"hash":        tx.Hash().Hex(),
					"blockNumber": blockNum,
					"from":        from.Hex(),
					"to":          tx.To.Hex(),
					"value":       tx.Value.String(),
				})
			}
		}
	}

	writeJSON(w, out)
}

// ------------------------------------------------------------
// 5. /explorer/stream/blocks  (SSE)
// ------------------------------------------------------------
func (api *ExplorerAPI) handleStreamBlocks(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := api.Events.SubscribeBlocks()
	ctx := r.Context()

	for {
		select {
		case block := <-ch:
			data, _ := json.Marshal(block)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}

// ------------------------------------------------------------
// 6. /explorer/stream/txs  (SSE)
// ------------------------------------------------------------
func (api *ExplorerAPI) handleStreamTxs(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ch := api.Events.SubscribeTxs()
	ctx := r.Context()

	for {
		select {
		case tx := <-ch:
			data, _ := json.Marshal(tx)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		case <-ctx.Done():
			return
		}
	}
}
