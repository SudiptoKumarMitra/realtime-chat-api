package websocket

import (
	"encoding/json"
	"log"
	"time"

	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the connection.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the connection.
	pongWait = 60 * time.Second

	// Send pings to client with this period. Must be less than pongWait.
	pingPeriod = 54 * time.Second

	// Maximum message size allowed from the peer.
	// 0 means no limit (can be added later if needed).
)

// Client represents one connected WebSocket user.
// It holds the raw connection, a reference to the Hub,
// and a send channel for outbound messages.
type Client struct {
	hub      *Hub
	conn     *websocket.Conn
	send     chan Message // outbound messages, buffered
	room     string       // current room ID, empty means global
	userID   string       // authenticated user ID from JWT
	username string       // authenticated username from JWT
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

// SetIdentity sets the authenticated user identity from verified JWT claims.
func (c *Client) SetIdentity(userID, username string) {
	c.userID = userID
	c.username = username
}

// WritePump reads messages from the send channel, encodes them as JSON,
// and writes them to the WebSocket connection. It runs in its own goroutine.
// When the send channel is closed (by the Hub), the loop exits
// and the goroutine stops.
func (c *Client) WritePump() {
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			data, err := json.Marshal(message)
			if err != nil {
				log.Printf("marshal error: %v", err)
				continue
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ReadPump reads messages from the WebSocket connection, decodes them
// from JSON, and broadcasts them through the Hub. It runs synchronously
// in the handler goroutine. On read error or invalid JSON, it
// unregisters the client and returns.
func (c *Client) ReadPump() {
	defer c.hub.Unregister(c)

	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		var msg Message
		if err := json.Unmarshal(raw, &msg); err != nil {
			log.Printf("invalid json from client")
			return
		}

		switch msg.Type {
		case "join":
			c.hub.Join(c, msg.RoomID)
		default:
			c.hub.Broadcast(msg)
		}
	}
}
