package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"realtime-chat-api/internal/service"
)

type authHandler struct {
	auth *service.AuthService
}

// NewAuthHandler creates an authHandler with the given AuthService.
func NewAuthHandler(auth *service.AuthService) *authHandler {
	return &authHandler{auth: auth}
}

// RegisterRoutes registers /register and /login on the given mux.
func (h *authHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/register", h.handleRegister)
	mux.HandleFunc("/login", h.handleLogin)
}

func (h *authHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	user, err := h.auth.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, service.ErrUsernameTaken) {
			status = http.StatusConflict
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func (h *authHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid json"}`, http.StatusBadRequest)
		return
	}

	user, token, err := h.auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		if err := json.NewEncoder(w).Encode(map[string]string{"error": err.Error()}); err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"message":  "login successful",
		"user_id":  user.ID,
		"username": user.Username,
		"token":    token,
	}); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
