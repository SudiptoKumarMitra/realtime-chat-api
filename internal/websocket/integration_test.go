package websocket_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"realtime-chat-api/internal/handler"
	"realtime-chat-api/internal/repository"
	"realtime-chat-api/internal/service"
	ws "realtime-chat-api/internal/websocket"

	wslib "github.com/gorilla/websocket"
)

const testSecret = "test-secret-for-ws-integration-32bytes!"

// setupTestServer creates a test HTTP server with WebSocket endpoint.
func setupTestServer(t *testing.T) (*service.AuthService, *ws.Hub, string) {
	t.Helper()

	repo := repository.NewMockUserRepository()
	auth, err := service.NewAuthService(repo, []byte(testSecret))
	if err != nil {
		t.Fatalf("NewAuthService failed: %v", err)
	}
	hub := ws.NewHub()
	go hub.Run()

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleWebSocket(hub, auth, w, r)
	})

	server := httptest.NewServer(mux)
	t.Cleanup(func() {
		server.Close()
	})

	return auth, hub, server.URL
}

// registerAndGetToken registers a user and returns a JWT.
func registerAndGetToken(t *testing.T, auth *service.AuthService, username string) string {
	t.Helper()
	_, err := auth.Register(context.Background(), username, "password123")
	if err != nil {
		t.Fatalf("register %s failed: %v", username, err)
	}
	_, token, err := auth.Login(context.Background(), username, "password123")
	if err != nil {
		t.Fatalf("login %s failed: %v", username, err)
	}
	return token
}

// dialWS connects a WebSocket client with optional Bearer token.
func dialWS(t *testing.T, serverURL, token string) (*wslib.Conn, error) {
	t.Helper()
	header := http.Header{}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	url := "ws" + strings.TrimPrefix(serverURL, "http") + "/ws"
	dialer := wslib.Dialer{}
	conn, _, err := dialer.Dial(url, header)
	return conn, err
}

// waitForClientCount waits until Hub has the expected number of clients.
func waitForClientCount(t *testing.T, hub *ws.Hub, expected int, timeout time.Duration) bool {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hub.ClientCount() == expected {
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// --- Connection tests ---

func TestWS_ValidConnection(t *testing.T) {
	auth, _, serverURL := setupTestServer(t)
	token := registerAndGetToken(t, auth, "alice")

	conn, err := dialWS(t, serverURL, token)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	t.Log("valid connection established successfully")
}

func TestWS_MissingToken(t *testing.T) {
	_, _, serverURL := setupTestServer(t)

	conn, err := dialWS(t, serverURL, "")
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to fail without token")
	}
	t.Logf("correctly rejected: %v", err)
}

func TestWS_InvalidToken(t *testing.T) {
	_, _, serverURL := setupTestServer(t)

	conn, err := dialWS(t, serverURL, "invalid.token.here")
	if err == nil {
		conn.Close()
		t.Fatal("expected connection to fail with invalid token")
	}
	t.Logf("correctly rejected: %v", err)
}

// --- Registration test ---

func TestWS_AuthenticatedClientRegistered(t *testing.T) {
	auth, hub, serverURL := setupTestServer(t)
	token := registerAndGetToken(t, auth, "alice")

	conn, err := dialWS(t, serverURL, token)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	if !waitForClientCount(t, hub, 1, 2*time.Second) {
		t.Fatalf("expected 1 client in hub, got %d", hub.ClientCount())
	}
}

// --- Broadcast test ---

func TestWS_Broadcast(t *testing.T) {
	auth, hub, serverURL := setupTestServer(t)

	tokenAlice := registerAndGetToken(t, auth, "alice")
	tokenBob := registerAndGetToken(t, auth, "bob")

	connAlice, err := dialWS(t, serverURL, tokenAlice)
	if err != nil {
		t.Fatalf("dial alice failed: %v", err)
	}
	defer connAlice.Close()

	connBob, err := dialWS(t, serverURL, tokenBob)
	if err != nil {
		t.Fatalf("dial bob failed: %v", err)
	}
	defer connBob.Close()

	// Wait for both to register
	if !waitForClientCount(t, hub, 2, 2*time.Second) {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount())
	}

	// Set read deadlines to avoid blocking forever
	connAlice.SetReadDeadline(time.Now().Add(2 * time.Second))
	connBob.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Alice sends a message
	msg := ws.Message{Type: "message", Content: "hello from alice"}
	if err := connAlice.WriteJSON(msg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Both should receive it
	var receivedAlice, receivedBob ws.Message
	if err := connAlice.ReadJSON(&receivedAlice); err != nil {
		t.Fatalf("alice read failed: %v", err)
	}
	if err := connBob.ReadJSON(&receivedBob); err != nil {
		t.Fatalf("bob read failed: %v", err)
	}

	if receivedAlice.Content != "hello from alice" {
		t.Errorf("alice got %q, want %q", receivedAlice.Content, "hello from alice")
	}
	if receivedBob.Content != "hello from alice" {
		t.Errorf("bob got %q, want %q", receivedBob.Content, "hello from alice")
	}
}

// --- Room routing test ---

func TestWS_RoomRouting(t *testing.T) {
	auth, hub, serverURL := setupTestServer(t)

	tokenAlice := registerAndGetToken(t, auth, "alice")
	tokenBob := registerAndGetToken(t, auth, "bob")

	connAlice, err := dialWS(t, serverURL, tokenAlice)
	if err != nil {
		t.Fatalf("dial alice failed: %v", err)
	}
	defer connAlice.Close()

	connBob, err := dialWS(t, serverURL, tokenBob)
	if err != nil {
		t.Fatalf("dial bob failed: %v", err)
	}
	defer connBob.Close()

	// Wait for both to register
	if !waitForClientCount(t, hub, 2, 2*time.Second) {
		t.Fatalf("expected 2 clients, got %d", hub.ClientCount())
	}

	// Alice joins room1
	joinMsg := ws.Message{Type: "join", RoomID: "room1"}
	if err := connAlice.WriteJSON(joinMsg); err != nil {
		t.Fatalf("alice join failed: %v", err)
	}

	// Bob joins room2
	joinMsg = ws.Message{Type: "join", RoomID: "room2"}
	if err := connBob.WriteJSON(joinMsg); err != nil {
		t.Fatalf("bob join failed: %v", err)
	}

	// Wait for joins to propagate
	time.Sleep(100 * time.Millisecond)

	// Set read deadlines
	connAlice.SetReadDeadline(time.Now().Add(2 * time.Second))
	connBob.SetReadDeadline(time.Now().Add(2 * time.Second))

	// Send message to room1
	roomMsg := ws.Message{Type: "message", Content: "hi room1", RoomID: "room1"}
	if err := connAlice.WriteJSON(roomMsg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Alice (in room1) should receive it
	var receivedAlice ws.Message
	if err := connAlice.ReadJSON(&receivedAlice); err != nil {
		t.Fatalf("alice read failed: %v", err)
	}
	if receivedAlice.Content != "hi room1" {
		t.Errorf("alice got %q, want %q", receivedAlice.Content, "hi room1")
	}

	// Bob (in room2) should NOT receive it — read should timeout
	connBob.SetReadDeadline(time.Now().Add(300 * time.Millisecond))
	_, _, err = connBob.ReadMessage()
	if err == nil {
		t.Error("bob should not have received room1 message")
	} else {
		t.Log("bob correctly did not receive room1 message (timeout)")
	}
}

// --- Clean disconnect test ---

func TestWS_CleanDisconnect(t *testing.T) {
	auth, hub, serverURL := setupTestServer(t)
	token := registerAndGetToken(t, auth, "alice")

	conn, err := dialWS(t, serverURL, token)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}

	// Wait for registration
	if !waitForClientCount(t, hub, 1, 2*time.Second) {
		t.Fatalf("expected 1 client, got %d", hub.ClientCount())
	}

	// Disconnect
	conn.Close()

	// Wait for unregistration
	if !waitForClientCount(t, hub, 0, 2*time.Second) {
		t.Fatalf("expected 0 clients after disconnect, got %d", hub.ClientCount())
	}
}
