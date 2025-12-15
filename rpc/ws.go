package rpc

import (
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/events"
)

// WSHandler exposeert EventBus via websocket
func WSHandler(bus *events.EventBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		txCh := bus.SubscribeTxs()
		blockCh := bus.SubscribeBlocks()

		for {
			select {
			case tx := <-txCh:
				_ = conn.WriteJSON(map[string]interface{}{
					"type": "tx",
					"data": tx,
				})

			case blk := <-blockCh:
				_ = conn.WriteJSON(map[string]interface{}{
					"type": "block",
					"data": blk,
				})
			}
		}
	}
}
