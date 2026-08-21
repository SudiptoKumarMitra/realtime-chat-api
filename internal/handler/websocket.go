package handler

import (
	"log"
	"net/http"
	"os"
	"strings"

	"realtime-chat-api/internal/service"
	ws "realtime-chat-api/internal/websocket"

	"github.com/gorilla/websocket"
)

// AllowedOrigins is the set of permitted WebSocket Origin headers.
// Populated from WS_ALLOWED_ORIGINS at init time. Empty = reject all.
// Each entry should be a scheme+host without port (e.g. "http://localhost").
// Origins with ports (e.g. from httptest.Server) match by prefix.
var AllowedOrigins map[string]bool

func init() {
	AllowedOrigins = make(map[string]bool)
	origins := os.Getenv("WS_ALLOWED_ORIGINS")
	if origins == "" {
		return
	}
	for _, o := range strings.Split(origins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			AllowedOrigins[o] = true
		}
	}
}

func checkOrigin(r *http.Request) bool {
	// Allow requests with no Origin header (non-browser clients, same-origin).
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	if len(AllowedOrigins) == 0 {
		return false
	}
	// Exact match first.
	if AllowedOrigins[origin] {
		return true
	}
	// Prefix match: "http://localhost" matches "http://localhost:8080".
	for allowed := range AllowedOrigins {
		if strings.HasPrefix(origin, allowed) {
			return true
		}
	}
	return false
}

// upgrader converts a plain HTTP connection into a WebSocket connection.
// CheckOrigin validates the Origin header against the allowlist.
var upgrader = websocket.Upgrader{
	CheckOrigin: checkOrigin,
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
