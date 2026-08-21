package repository

import (
	"context"

	"realtime-chat-api/internal/model"
)

// UserRepository defines the interface for user persistence.
// Any implementation (PostgreSQL, in-memory, mock) must satisfy this contract.
type UserRepository interface {
	// CreateUser inserts a new user into the store.
	// Returns an error if the username already exists.
	CreateUser(ctx context.Context, user *model.User) error

	// FindByUsername retrieves a user by username.
	// Returns the user if found, or an error if not found.
	FindByUsername(ctx context.Context, username string) (*model.User, error)
}
