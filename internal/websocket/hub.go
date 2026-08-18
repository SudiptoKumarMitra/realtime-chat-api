package websocket

import "log"

// Hub manages all connected WebSocket clients.
// Only the Hub goroutine reads or writes the clients map.
// Other goroutines communicate via channels.
type Hub struct {
	clients    map[*Client]bool // all connected clients
	register   chan *Client     // incoming registration requests
	unregister chan *Client     // incoming disconnect requests
}

// NewHub creates a Hub with initialized channels.
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Register sends a client to the register channel.
// This is safe to call from any goroutine.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister sends a client to the unregister channel.
// This is safe to call from any goroutine.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Run starts the Hub event loop.
// It blocks forever, processing register/unregister events one at a time.
// This goroutine is the ONLY code that touches the clients map.
func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			log.Printf("client registered (%d total)", len(h.clients))

		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				client.conn.Close()
				log.Printf("client unregistered (%d total)", len(h.clients))
			}
		}
	}
}
