package websocket

import (
	"log"
	"sync"
)

// joinRequest is a request to join a room.
type joinRequest struct {
	client *Client
	roomID string
}

// Hub manages all connected WebSocket clients.
// Only the Hub goroutine reads or writes the clients map.
// Other goroutines communicate via channels.
type Hub struct {
	clients     map[*Client]bool // all connected clients
	register    chan *Client     // incoming registration requests
	unregister  chan *Client     // incoming disconnect requests
	broadcast   chan Message     // incoming messages to send to all clients
	join        chan joinRequest // incoming join requests
	clientCount chan chan int    // request/response for client count
	stop        chan struct{}    // closed to signal shutdown
	stopOnce    sync.Once       // ensures close(h.stop) runs exactly once
}

// NewHub creates a Hub with initialized channels.
func NewHub() *Hub {
	return &Hub{
		clients:     make(map[*Client]bool),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		broadcast:   make(chan Message),
		join:        make(chan joinRequest),
		clientCount: make(chan chan int),
		stop:        make(chan struct{}),
	}
}

// Register sends a client to the register channel.
// Safe to call from any goroutine. Drops silently if Hub is stopped.
func (h *Hub) Register(client *Client) {
	select {
	case h.register <- client:
	case <-h.stop:
	}
}

// Unregister sends a client to the unregister channel.
// Safe to call from any goroutine. Drops silently if Hub is stopped.
func (h *Hub) Unregister(client *Client) {
	select {
	case h.unregister <- client:
	case <-h.stop:
	}
}

// Broadcast sends a message to the broadcast channel.
// Safe to call from any goroutine. Drops silently if Hub is stopped.
func (h *Hub) Broadcast(message Message) {
	select {
	case h.broadcast <- message:
	case <-h.stop:
	}
}

// Join sends a join request to the join channel.
// Safe to call from any goroutine. Drops silently if Hub is stopped.
func (h *Hub) Join(client *Client, roomID string) {
	select {
	case h.join <- joinRequest{client: client, roomID: roomID}:
	case <-h.stop:
	}
}

// ClientCount returns the number of currently connected clients.
// Safe to call from any goroutine. Returns 0 if Hub is stopped.
func (h *Hub) ClientCount() int {
	resp := make(chan int)
	select {
	case h.clientCount <- resp:
		return <-resp
	case <-h.stop:
		return 0
	}
}

// Stop signals the Hub to shut down.
// Safe to call multiple times — closing is guarded by sync.Once.
func (h *Hub) Stop() {
	h.stopOnce.Do(func() {
		close(h.stop)
	})
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
				close(client.send)
				client.conn.Close()
				log.Printf("client unregistered (%d total)", len(h.clients))
			}

		case req := <-h.join:
			req.client.room = req.roomID
			log.Printf("client joined room: %s", req.roomID)

		case resp := <-h.clientCount:
			resp <- len(h.clients)

		case <-h.stop:
			log.Printf("hub stopping, closing %d connections", len(h.clients))
			for client := range h.clients {
				close(client.send)
				client.conn.Close()
			}
			return

		case message := <-h.broadcast:
			for client := range h.clients {
				if message.RoomID != "" && client.room != message.RoomID {
					continue
				}
				select {
				case client.send <- message:
				default:
					log.Printf("client too slow, dropping: %s", client.conn.RemoteAddr())
					close(client.send)
					delete(h.clients, client)
					client.conn.Close()
				}
			}
		}
	}
}
