package repository_test

import (
	"context"
	"database/sql"
	"os"
	"testing"

	"realtime-chat-api/internal/database"
	"realtime-chat-api/internal/model"
	"realtime-chat-api/internal/repository"

	"github.com/google/uuid"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set, skipping integration test")
	}

	db, err := database.Connect()
	if err != nil {
		t.Fatalf("failed to connect to database: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
	})

	return db
}

func cleanupUsers(t *testing.T, db *sql.DB, prefix string) {
	t.Helper()
	db.Exec("DELETE FROM users WHERE username LIKE $1", prefix+"%")
}

func TestCreateUser_Success(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPostgresUserRepository(db)
	ctx := context.Background()

	testUsername := "test_create_success"
	cleanupUsers(t, db, "test_")
	t.Cleanup(func() { cleanupUsers(t, db, "test_") })

	user := &model.User{
		ID:           uuid.New().String(),
		Username:     testUsername,
		PasswordHash: "hashed_password_123",
	}

	err := repo.CreateUser(ctx, user)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Verify user was created
	var count int
	err = db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users WHERE username = $1", testUsername).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}
}

func TestCreateUser_DuplicateUsername(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPostgresUserRepository(db)
	ctx := context.Background()

	testUsername := "test_create_duplicate"
	cleanupUsers(t, db, "test_")
	t.Cleanup(func() { cleanupUsers(t, db, "test_") })

	user1 := &model.User{
		ID:           uuid.New().String(),
		Username:     testUsername,
		PasswordHash: "hash1",
	}

	err := repo.CreateUser(ctx, user1)
	if err != nil {
		t.Fatalf("first CreateUser failed: %v", err)
	}

	// Try to create another user with same username
	user2 := &model.User{
		ID:           uuid.New().String(),
		Username:     testUsername,
		PasswordHash: "hash2",
	}

	err = repo.CreateUser(ctx, user2)
	if err == nil {
		t.Error("expected error for duplicate username, got nil")
	}
}

func TestFindByUsername_Found(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPostgresUserRepository(db)
	ctx := context.Background()

	testUsername := "test_find_found"
	cleanupUsers(t, db, "test_")
	t.Cleanup(func() { cleanupUsers(t, db, "test_") })

	userID := uuid.New().String()
	original := &model.User{
		ID:           userID,
		Username:     testUsername,
		PasswordHash: "hashed_password_abc",
	}

	err := repo.CreateUser(ctx, original)
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	// Find the user
	found, err := repo.FindByUsername(ctx, testUsername)
	if err != nil {
		t.Fatalf("FindByUsername failed: %v", err)
	}

	if found.ID != original.ID {
		t.Errorf("ID mismatch: got %s, want %s", found.ID, original.ID)
	}
	if found.Username != original.Username {
		t.Errorf("Username mismatch: got %s, want %s", found.Username, original.Username)
	}
	if found.PasswordHash != original.PasswordHash {
		t.Errorf("PasswordHash mismatch: got %s, want %s", found.PasswordHash, original.PasswordHash)
	}
}

func TestFindByUsername_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repository.NewPostgresUserRepository(db)
	ctx := context.Background()

	cleanupUsers(t, db, "test_")
	t.Cleanup(func() { cleanupUsers(t, db, "test_") })

	// Try to find a non-existent user
	found, err := repo.FindByUsername(ctx, "nonexistent_user_xyz")
	if err != sql.ErrNoRows {
		t.Errorf("expected sql.ErrNoRows, got %v", err)
	}
	if found != nil {
		t.Errorf("expected nil user, got %v", found)
	}
}
