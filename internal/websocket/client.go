package websocket

import (
	"log"

	"github.com/gorilla/websocket"
)

// Client represents one connected WebSocket user.
// It holds the raw connection, a reference to the Hub,
// and a send channel for outbound messages.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan []byte // outbound messages, buffered
}

// NewClient creates a Client bound to a Hub and a WebSocket connection.
// The send channel is buffered with capacity 256.
// This means the Hub can send 256 messages before blocking.
// 256 is a reasonable small capacity: enough to handle burst traffic
// without using excessive memory, and small enough to apply backpressure
// quickly if a client falls behind.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
	}
}

// WritePump reads messages from the send channel and writes them
// to the WebSocket connection. It runs in its own goroutine.
// When the send channel is closed (by the Hub), the loop exits
// and the goroutine stops.
func (c *Client) WritePump() {
	for message := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}

// ReadPump reads messages from the WebSocket connection and broadcasts
// them through the Hub. It runs synchronously in the handler goroutine.
// On read error or disconnect, it unregisters the client and returns.
func (c *Client) ReadPump() {
	defer c.hub.Unregister(c)

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		log.Printf("received: %s", message)
		c.hub.Broadcast(message)
	}
}
