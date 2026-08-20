package handler

import (
	"log"
	"net/http"
	"strings"

	"realtime-chat-api/internal/service"
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

// HandleWebSocket authenticates the request via JWT, upgrades to WebSocket,
// creates a Client with verified identity, and starts the pump goroutines.
func HandleWebSocket(hub *ws.Hub, auth *service.AuthService, w http.ResponseWriter, r *http.Request) {
	// Step 1: Extract and verify JWT from Authorization header.
	claims, err := extractAndVerifyToken(r, auth)
	if err != nil {
		log.Printf("auth failed: %v", err)
		http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
		return
	}

	// Step 2: Upgrade the connection.
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("upgrade error: %v", err)
		return
	}

	// Step 3: Create a Client, attach verified identity, register with Hub.
	client := ws.NewClient(hub, conn)
	client.SetIdentity(claims.UserID, claims.Username)
	hub.Register(client)

	log.Printf("client connected: %s (%s)", conn.RemoteAddr(), claims.Username)

	// Step 4: Start WritePump in its own goroutine.
	go client.WritePump()

	// Step 5: Run ReadPump synchronously.
	client.ReadPump()
}

// extractAndVerifyToken reads the Bearer token from the Authorization header
// and verifies it using the AuthService.
func extractAndVerifyToken(r *http.Request, auth *service.AuthService) (*service.Claims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errNoToken
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return nil, errMalformedToken
	}

	return auth.VerifyToken(parts[1])
}

var (
	errNoToken        = &authError{"missing authorization token"}
	errMalformedToken = &authError{"malformed authorization header"}
)

type authError struct {
	msg string
}

func (e *authError) Error() string {
	return e.msg
}
