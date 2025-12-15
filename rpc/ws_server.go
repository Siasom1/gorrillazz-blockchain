package rpc

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type WebSocketHub struct {
	clients    map[*websocket.Conn]bool
	register   chan *websocket.Conn
	unregister chan *websocket.Conn
	broadcast  chan WSMessage
}

func NewWebSocketHub() *WebSocketHub {
	return &WebSocketHub{
		clients:    make(map[*websocket.Conn]bool),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		broadcast:  make(chan WSMessage),
	}
}

func (h *WebSocketHub) Run() {
	for {
		select {
		case conn := <-h.register:
			h.clients[conn] = true

		case conn := <-h.unregister:
			delete(h.clients, conn)
			conn.Close()

		case msg := <-h.broadcast:
			data, _ := json.Marshal(msg)
			for c := range h.clients {
				c.WriteMessage(websocket.TextMessage, data)
			}
		}
	}
}

func (h *WebSocketHub) Broadcast(msg WSMessage) {
	h.broadcast <- msg
}

func (h *WebSocketHub) HandleWS(w http.ResponseWriter, r *http.Request) {
	con, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WS Upgrade error:", err)
		return
	}
	h.register <- con
}
