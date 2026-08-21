package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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

	// Create server with explicit struct for graceful shutdown.
	addr := ":8081"
	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// Start server in a goroutine.
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()
	log.Printf("server starting on %s", addr)

	// Wait for interrupt signal.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("received signal: %s, shutting down...", sig)

	// Shutdown sequence with 10-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Stop HTTP server — stops accepting new connections, waits for in-flight.
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}

	// 2. Stop Hub — closes all WebSocket connections.
	hub.Stop()

	// 3. Close database connection pool.
	if err := db.Close(); err != nil {
		log.Printf("database close error: %v", err)
	}

	log.Println("server stopped")
}
