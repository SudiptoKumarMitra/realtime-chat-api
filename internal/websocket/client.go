package websocket

import (
	"encoding/json"
	"log"

	"github.com/gorilla/websocket"
)

// Client represents one connected WebSocket user.
// It holds the raw connection, a reference to the Hub,
// and a send channel for outbound messages.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	send chan Message // outbound messages, buffered
}

// NewClient creates a Client bound to a Hub and a WebSocket connection.
// The send channel is buffered with capacity 256.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
		send: make(chan Message, 256),
	}
}

// WritePump reads messages from the send channel, encodes them as JSON,
// and writes them to the WebSocket connection. It runs in its own goroutine.
// When the send channel is closed (by the Hub), the loop exits
// and the goroutine stops.
func (c *Client) WritePump() {
	for message := range c.send {
		data, err := json.Marshal(message)
		if err != nil {
			log.Printf("marshal error: %v", err)
			continue
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
			log.Printf("write error: %v", err)
			return
		}
	}
}

// ReadPump reads messages from the WebSocket connection, decodes them
// from JSON, and broadcasts them through the Hub. It runs synchronously
// in the handler goroutine. On read error or invalid JSON, it
// unregisters the client and returns.
func (c *Client) ReadPump() {
	defer c.hub.Unregister(c)

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			log.Printf("read error: %v", err)
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("invalid json: %v", err)
			return
		}

		log.Printf("received: %s", msg.Content)
		c.hub.Broadcast(msg)
	}
}
