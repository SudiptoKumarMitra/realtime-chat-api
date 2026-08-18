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
// creates a Client, registers it with the Hub, and runs the read loop.
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

	// Step 3: Read messages in a loop until the client disconnects.
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			break
		}

		log.Printf("received: %s", message)

		// Send the message to the Hub for broadcasting to all clients.
		hub.Broadcast(message)
	}

	// Step 4: Unregister the client when the read loop exits.
	hub.Unregister(client)
	log.Printf("client disconnected: %s", conn.RemoteAddr())
}
