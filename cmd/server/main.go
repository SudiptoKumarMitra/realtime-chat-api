package main

import (
	"log"
	"net/http"

	"realtime-chat-api/internal/handler"
	ws "realtime-chat-api/internal/websocket"
)

func main() {
	// Create the Hub and start its event loop in a background goroutine.
	hub := ws.NewHub()
	go hub.Run()

	// Use a closure to pass the Hub to the WebSocket handler.
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleWebSocket(hub, w, r)
	})

	addr := ":8080"
	log.Printf("server starting on %s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
