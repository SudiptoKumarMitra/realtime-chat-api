package service_test

import (
	"testing"

	"realtime-chat-api/internal/repository"
	"realtime-chat-api/internal/service"
)

const testJWTSecret = "test-secret-key-for-unit-tests"

func setupAuthService() (*service.AuthService, *repository.MockUserRepository) {
	repo := repository.NewMockUserRepository()
	auth := service.NewAuthService(repo, []byte(testJWTSecret))
	return auth, repo
}

// --- Registration tests ---

func TestRegister_EmptyUsername(t *testing.T) {
	auth, _ := setupAuthService()

	user, err := auth.Register("", "password123")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err != service.ErrUsernameEmpty {
		t.Errorf("expected ErrUsernameEmpty, got %v", err)
	}
}

func TestRegister_ShortPassword(t *testing.T) {
	auth, _ := setupAuthService()

	user, err := auth.Register("alice", "abc")
	if user != nil {
		t.Errorf("expected nil user, got %v", user)
	}
	if err != service.ErrPasswordTooShort {
		t.Errorf("expected ErrPasswordTooShort, got %v", err)
	}
}

func TestRegister_Success(t *testing.T) {
	auth, repo := setupAuthService()

	user, err := auth.Register("alice", "password123")
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
	auth, _ := setupAuthService()

	// Register first user
	_, err := auth.Register("alice", "password123")
	if err != nil {
		t.Fatalf("first register failed: %v", err)
	}

	// Try to register same username again
	user, err := auth.Register("alice", "anotherpassword")
	if user != nil {
		t.Errorf("expected nil user on duplicate, got %v", user)
	}
	if err != service.ErrUsernameTaken {
		t.Errorf("expected ErrUsernameTaken, got %v", err)
	}
}

// --- Login tests ---

func TestLogin_UnknownUser(t *testing.T) {
	auth, _ := setupAuthService()

	user, token, err := auth.Login("unknown", "password123")
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
	auth, _ := setupAuthService()

	// Register a user
	_, err := auth.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Try to login with wrong password
	user, token, err := auth.Login("alice", "wrongpassword")
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
	auth, _ := setupAuthService()

	// Register a user
	_, err := auth.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Login with correct credentials
	user, token, err := auth.Login("alice", "password123")
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
	auth, _ := setupAuthService()

	// Register and login to get a token
	_, err := auth.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, token, err := auth.Login("alice", "password123")
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
	auth, _ := setupAuthService()

	claims, err := auth.VerifyToken("this-is-not-a-valid-token")
	if err == nil {
		t.Error("expected error for invalid token")
	}
	if claims != nil {
		t.Errorf("expected nil claims, got %v", claims)
	}
}

func TestVerifyToken_WrongSecret(t *testing.T) {
	// Create auth service with one secret
	repo1 := repository.NewMockUserRepository()
	auth1 := service.NewAuthService(repo1, []byte("secret-one"))

	// Create another auth service with different secret
	repo2 := repository.NewMockUserRepository()
	auth2 := service.NewAuthService(repo2, []byte("secret-two"))

	// Register user with auth1
	_, err := auth1.Register("alice", "password123")
	if err != nil {
		t.Fatalf("register failed: %v", err)
	}

	// Login with auth1 to get a token
	_, token, err := auth1.Login("alice", "password123")
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
