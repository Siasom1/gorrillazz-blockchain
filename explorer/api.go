package explorer

import (
	"fmt"
	"log"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/core/blockchain"
	"github.com/Siasom1/gorrillazz-chain/events"
)

type ExplorerAPI struct {
	Chain  *blockchain.Blockchain
	Events *events.EventBus
}

func NewExplorerAPI(chain *blockchain.Blockchain, bus *events.EventBus) *ExplorerAPI {
	return &ExplorerAPI{
		Chain:  chain,
		Events: bus,
	}
}

func (api *ExplorerAPI) Start(port int) {
	http.HandleFunc("/explorer/latest-blocks", api.handleLatestBlocks)
	http.HandleFunc("/explorer/block/", api.handleBlockByNumber)
	http.HandleFunc("/explorer/tx/", api.handleTransaction)
	http.HandleFunc("/explorer/address/", api.handleAddressTxs)

	// Live streams (SSE)
	http.HandleFunc("/explorer/stream/blocks", api.handleStreamBlocks)
	http.HandleFunc("/explorer/stream/txs", api.handleStreamTxs)

	addr := fmt.Sprintf(":%d", port)

	log.Printf("[ExplorerAPI] Running on %s\n", addr)
	go http.ListenAndServe(addr, nil)
}
