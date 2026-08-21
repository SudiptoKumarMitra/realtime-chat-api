package repository

import (
	"database/sql"

	"realtime-chat-api/internal/model"

	"github.com/jackc/pgx/v5/pgconn"
)

// MockUserRepository is an in-memory implementation of UserRepository for testing.
type MockUserRepository struct {
	Users map[string]*model.User
	Err   error // simulates database errors
}

// NewMockUserRepository creates a fresh mock repository.
func NewMockUserRepository() *MockUserRepository {
	return &MockUserRepository{
		Users: make(map[string]*model.User),
	}
}

// CreateUser stores the user in memory.
// Returns a simulated unique violation error if the username already exists.
func (m *MockUserRepository) CreateUser(user *model.User) error {
	if m.Err != nil {
		return m.Err
	}
	if _, exists := m.Users[user.Username]; exists {
		return &pgconn.PgError{Code: "23505"}
	}
	m.Users[user.Username] = user
	return nil
}

// FindByUsername retrieves a user from memory.
// Returns sql.ErrNoRows if not found.
func (m *MockUserRepository) FindByUsername(username string) (*model.User, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	user, exists := m.Users[username]
	if !exists {
		return nil, sql.ErrNoRows
	}
	return user, nil
}
