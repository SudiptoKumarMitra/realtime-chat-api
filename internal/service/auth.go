package service

import (
	"errors"
	"sync"

	"realtime-chat-api/internal/model"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameEmpty      = errors.New("username cannot be empty")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters")
)

// AuthService handles user registration and login.
// It owns the in-memory user store, protected by RWMutex.
type AuthService struct {
	mu    sync.RWMutex
	users map[string]*model.User // username → User
}

// NewAuthService creates an AuthService with an empty user store.
func NewAuthService() *AuthService {
	return &AuthService{
		users: make(map[string]*model.User),
	}
}

// Register creates a new user with a hashed password.
func (s *AuthService) Register(username, password string) (*model.User, error) {
	if username == "" {
		return nil, ErrUsernameEmpty
	}
	if len(password) < 6 {
		return nil, ErrPasswordTooShort
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, ErrUsernameTaken
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
	}

	s.users[username] = user
	return user, nil
}

// Login verifies credentials and returns the user if valid.
func (s *AuthService) Login(username, password string) (*model.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}
