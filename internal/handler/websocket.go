package handler

import (
	"log"
	"net/http"

	ws "realtime-chat-api/internal/websocket"

	"github.com/gorilla/websocket"
)

// upgrader converts a plain HTTP connection into a WebSocket connection.
// CheckOrigin returns true to allow all connections for this learning stage.
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// HandleWebSocket upgrades the HTTP request to a WebSocket connection,
// creates a Client, registers it with the Hub, and starts the pump goroutines.
func HandleWebSocket(hub *ws.Hub, w http.ResponseWriter, r *http.Request) {
	// Step 1: Upgrade the connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	// Step 2: Create a Client and register it with the Hub.
	client := ws.NewClient(hub, conn)
	hub.Register(client)

	log.Printf("client connected: %s", conn.RemoteAddr())

	// Step 3: Start WritePump in its own goroutine.
	go client.WritePump()

	// Step 4: Run ReadPump synchronously.
	// This blocks until the client disconnects.
	client.ReadPump()
}
