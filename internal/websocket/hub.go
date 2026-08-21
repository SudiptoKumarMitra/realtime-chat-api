package websocket

import "log"

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
// This is safe to call from any goroutine.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister sends a client to the unregister channel.
// This is safe to call from any goroutine.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to the broadcast channel.
// This is safe to call from any goroutine.
func (h *Hub) Broadcast(message Message) {
	h.broadcast <- message
}

// Join sends a join request to the join channel.
// This is safe to call from any goroutine.
func (h *Hub) Join(client *Client, roomID string) {
	h.join <- joinRequest{client: client, roomID: roomID}
}

// ClientCount returns the number of currently connected clients.
// Safe to call from any goroutine — sends a request to the Hub.
func (h *Hub) ClientCount() int {
	resp := make(chan int)
	h.clientCount <- resp
	return <-resp
}

// Stop signals the Hub to shut down.
// Closing the stop channel unblocks Run() and closes all client connections.
func (h *Hub) Stop() {
	close(h.stop)
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
