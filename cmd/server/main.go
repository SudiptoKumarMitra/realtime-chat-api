package main

import (
	"log"
	"net/http"

	"realtime-chat-api/internal/handler"
)

func main() {
	// Register the /ws endpoint with our WebSocket handler.
	http.HandleFunc("/ws", handler.HandleWebSocket)

	addr := ":8080"
	log.Printf("server starting on %s", addr)

	// Start the HTTP server. ListenAndServe blocks until the server stops.
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
