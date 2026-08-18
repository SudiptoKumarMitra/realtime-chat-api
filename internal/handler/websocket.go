package handler

import (
	"log"
	"net/http"

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
// then reads and echoes messages back to the client.
func HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Step 1: Upgrade the connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}
	// Step 7: Ensure the connection is closed when this function returns.
	defer conn.Close()

	log.Printf("client connected: %s", conn.RemoteAddr())

	// Step 2: Read messages in a loop until the client disconnects or an error occurs.
	for {
		// ReadMessage returns the message type and the message bytes.
		// messageType is usually textMessage (1) or binaryMessage (2).
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			// A read error means the client disconnected or the connection broke.
			log.Printf("read error: %v", err)
			break
		}

		log.Printf("received: %s", message)

		// Step 3: Echo the message back using the same message type.
		if err := conn.WriteMessage(messageType, message); err != nil {
			log.Printf("write error: %v", err)
			break
		}
	}

	log.Printf("client disconnected: %s", conn.RemoteAddr())
}
