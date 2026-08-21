package service_test

import (
	"context"
	"testing"
	"time"

	"realtime-chat-api/internal/repository"
	"realtime-chat-api/internal/service"

	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-secret-key-for-unit-tests-32bytes"

func setupAuthService(t *testing.T) (*service.AuthService, *repository.MockUserRepository) {
	t.Helper()
	repo := repository.NewMockUserRepository()
	auth, err := service.NewAuthService(repo, []byte(testJWTSecret))
	if err != nil {
		t.Fatalf("NewAuthService failed: %v", err)
	}
	return auth, repo
}

// --- Registration tests ---

func TestRegister_EmptyUsername(t *testing.T) {
	auth, _ := setupAuthService(t)

	user, err := auth.Register(context.Background(), "", "password123")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err != service.ErrUsernameEmpty {
		t.Errorf("expected ErrUsernameEmpty, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	auth, _ := setupAuthService(t)

	user, err := auth.Register(context.Background(), "alice", "abc")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err != service.ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegister_Success(t *testing.T) {
	auth, repo := setupAuthService(t)

	user, err := auth.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "alice" {
		t.Errorf("expected username alice, got %s", user.Username)
	}
	if user.ID == "" {
		t.Error("expected non-empty user ID")
	}
	if user.PasswordHash == "" {
		t.Error("expected non-empty password hash")
	}

	// Verify user was stored in mock repository
	stored, exists := repo.Users["alice"]
	if !exists {
		t.Fatal("expected user to be stored in repository")
	}
	if stored.ID != user.ID {
		t.Errorf("stored user ID mismatch: got %s, want %s", stored.ID, user.ID)
	}
}

func TestRegister_DuplicateUsername(t *testing.T) {
	auth, _ := setupAuthService(t)

	// Register first user
	_, err := auth.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Try to register same username again
	user, err := auth.Register(context.Background(), "alice", "anotherpassword")
	if user != nil {
		t.Errorf("expected nil user on duplicate, got %v", user)
	}
	if err != service.ErrUsernameTaken {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}
}

// --- Login tests ---

func TestLogin_UnknownUser(t *testing.T) {
	auth, _ := setupAuthService(t)

	user, token, err := auth.Login(context.Background(), "unknown", "password123")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if token != "" {
		t.Errorf("expected empty token, got %s", token)
	}
	if err != service.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_WrongPassword(t *testing.T) {
	auth, _ := setupAuthService(t)

	// Register a user
	_, err := auth.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Try to login with wrong password
	user, token, err := auth.Login(context.Background(), "alice", "wrongpassword")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if token != "" {
		t.Errorf("expected empty token, got %s", token)
	}
	if err != service.ErrInvalidCredentials {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestLogin_Success(t *testing.T) {
	auth, _ := setupAuthService(t)

	// Register a user
	_, err := auth.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Login with correct credentials
	user, token, err := auth.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if user == nil {
		t.Fatal("expected user, got nil")
	}
	if user.Username != "alice" {
		t.Errorf("expected username alice, got %s", user.Username)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
}

// --- JWT tests ---

func TestVerifyToken_Valid(t *testing.T) {
	auth, _ := setupAuthService(t)

	// Register and login to get a token
	_, err := auth.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, token, err := auth.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Verify the token
	claims, err := auth.VerifyToken(token)
	if err != nil {
		t.Fatalf("verify token failed: %v", err)
	}
	if claims == nil {
		t.Fatal("expected claims, got nil")
	}
	if claims.Username != "alice" {
		t.Errorf("expected username alice, got %s", claims.Username)
	}
	if claims.UserID == "" {
		t.Error("expected non-empty user ID in claims")
	}
}

func TestVerifyToken_InvalidString(t *testing.T) {
	auth, _ := setupAuthService(t)

	claims, err := auth.VerifyToken("this-is-not-a-valid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
	if claims != nil {
		t.Errorf("expected nil claims, got %v", claims)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	secret1 := "secret-one-for-wrong-secret-test-1111"
	secret2 := "secret-two-for-wrong-secret-test-2222"

	// Create auth service with one secret
	repo1 := repository.NewMockUserRepository()
	auth1, err := service.NewAuthService(repo1, []byte(secret1))
	if err != nil {
		t.Fatalf("NewAuthService failed: %v", err)
	}

	// Create another auth service with different secret
	repo2 := repository.NewMockUserRepository()
	auth2, err := service.NewAuthService(repo2, []byte(secret2))
	if err != nil {
		t.Fatalf("NewAuthService failed: %v", err)
	}

	// Register user with auth1
	_, err = auth1.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Login with auth1 to get a token
	_, token, err := auth1.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}

	// Try to verify with auth2 (different secret)
	claims, err := auth2.VerifyToken(token)
	if err == nil {
		t.Error("expected error when verifying with wrong secret")
	}
	if claims != nil {
		t.Errorf("expected nil claims, got %v", claims)
	}
}

func TestVerifyToken_Expired(t *testing.T) {
	auth, _ := setupAuthService(t)

	// Manually create an expired token
	claims := service.Claims{
		UserID:   "user-123",
		Username: "alice",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatalf("failed to sign token: %v", err)
	}

	// Verify should reject the expired token
	result, err := auth.VerifyToken(tokenString)
	if err == nil {
		t.Error("expected error for expired token")
	}
	if result != nil {
		t.Errorf("expected nil claims, got %v", result)
	}
}

func TestNewAuthService_ShortSecret(t *testing.T) {
	repo := repository.NewMockUserRepository()
	_, err := service.NewAuthService(repo, []byte("short"))
	if err != service.ErrJWTSecretTooShort {
		t.Errorf("expected ErrJWTSecretTooShort, got %v", err)
	}
}
