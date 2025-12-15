package rpc

import (
	"log"
	"net/http"

	"github.com/Siasom1/gorrillazz-chain/events"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type wsMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

func WSHandler(bus *events.EventBus) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Println("[WS] upgrade error:", err)
			return
		}
		defer conn.Close()

		log.Println("[WS] client connected")

		done := make(chan struct{})

		go func() {
			defer close(done)
			for {
				// We don't expect messages from client (push-only)
				if _, _, err := conn.ReadMessage(); err != nil {
					return
				}
			}
		}()

		for {
			select {

			case blk := <-bus.Blocks:
				_ = conn.WriteJSON(wsMessage{
					Type: "block",
					Data: blk,
				})

			case tx := <-bus.Txs:
				_ = conn.WriteJSON(wsMessage{
					Type: "tx",
					Data: tx,
				})

			case pay := <-bus.Payments:
				_ = conn.WriteJSON(wsMessage{
					Type: "payment",
					Data: pay,
				})

			case <-done:
				log.Println("[WS] client disconnected")
				return
			}
		}
	}
}
