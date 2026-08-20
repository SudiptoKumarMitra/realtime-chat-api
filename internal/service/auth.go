package service

import (
	"database/sql"
	"errors"
	"time"

	"realtime-chat-api/internal/model"
	"realtime-chat-api/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
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
// It delegates persistence to a UserRepository.
type AuthService struct {
	repo      repository.UserRepository
	jwtSecret []byte
}

// NewAuthService creates an AuthService with the given repository and JWT secret.
func NewAuthService(repo repository.UserRepository, jwtSecret []byte) *AuthService {
	return &AuthService{
		repo:      repo,
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

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     username,
		PasswordHash: string(hash),
	}

	if err := s.repo.CreateUser(user); err != nil {
		if isUniqueViolation(err) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}

	return user, nil
}

// Login verifies credentials and returns the user plus a JWT.
func (s *AuthService) Login(username, password string) (*model.User, string, error) {
	user, err := s.repo.FindByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", ErrInvalidCredentials
		}
		return nil, "", err
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

// VerifyToken parses and validates a JWT token string.
// It checks the signature, expiration, and returns the claims.
func (s *AuthService) VerifyToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.jwtSecret, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// isUniqueViolation checks if the error is a PostgreSQL unique constraint violation.
// Uses pgconn.PgError with Code 23505 (unique_violation).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
