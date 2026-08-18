package websocket

import "github.com/gorilla/websocket"

// Client represents one connected WebSocket user.
// It holds the raw connection and a reference to the Hub
// so it can register and unregister itself.
type Client struct {
	hub  *Hub
	conn *websocket.Conn
}

// NewClient creates a Client bound to a Hub and a WebSocket connection.
func NewClient(hub *Hub, conn *websocket.Conn) *Client {
	return &Client{
		hub:  hub,
		conn: conn,
	}
}
