package main

import (
	"log"
	"net/http"
	"os"

	"realtime-chat-api/internal/database"
	"realtime-chat-api/internal/handler"
	"realtime-chat-api/internal/repository"
	"realtime-chat-api/internal/service"
	ws "realtime-chat-api/internal/websocket"
)

func main() {
	// JWT_SECRET is required.
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	// Connect to PostgreSQL.
	db, err := database.Connect()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()
	log.Println("connected to PostgreSQL")

	// Create the Hub and start its event loop in a background goroutine.
	hub := ws.NewHub()
	go hub.Run()

	// Create repository and auth service.
	userRepo := repository.NewPostgresUserRepository(db)
	auth := service.NewAuthService(userRepo, []byte(jwtSecret))
	authHandler := handler.NewAuthHandler(auth)

	// Set up routes.
	mux := http.NewServeMux()

	// WebSocket endpoint (requires JWT).
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		handler.HandleWebSocket(hub, auth, w, r)
	})

	// Auth endpoints.
	authHandler.RegisterRoutes(mux)

	addr := ":8081"
	log.Printf("server starting on %s", addr)

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
