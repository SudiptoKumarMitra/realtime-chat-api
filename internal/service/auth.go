package service

import (
	"errors"
	"sync"
	"time"

	"realtime-chat-api/internal/model"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUsernameEmpty      = errors.New("username cannot be empty")
	ErrPasswordTooShort   = errors.New("password must be at least 6 characters")
	ErrJWTSecretNotSet    = errors.New("JWT_SECRET environment variable is required")
)

// Claims defines the JWT token claims.
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// AuthService handles user registration and login.
// It owns the in-memory user store, protected by RWMutex.
type AuthService struct {
	mu        sync.RWMutex
	users     map[string]*model.User // username → User
	jwtSecret []byte
}

// NewAuthService creates an AuthService with the given JWT secret.
func NewAuthService(jwtSecret []byte) *AuthService {
	return &AuthService{
		users:     make(map[string]*model.User),
		jwtSecret: jwtSecret,
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

// Login verifies credentials and returns the user plus a JWT.
func (s *AuthService) Login(username, password string) (*model.User, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

// generateToken creates a signed JWT for the given user.
func (s *AuthService) generateToken(user *model.User) (string, error) {
	claims := Claims{
		UserID:   user.ID,
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
