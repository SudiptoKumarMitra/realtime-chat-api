package repository

import (
	"context"
	"database/sql"

	"realtime-chat-api/internal/model"
)

// PostgresUserRepository implements UserRepository using PostgreSQL.
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository creates a repository backed by the given *sql.DB.
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// CreateUser inserts a new user row.
func (r *PostgresUserRepository) CreateUser(ctx context.Context, user *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, username, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Username, user.PasswordHash,
	)
	return err
}

// FindByUsername retrieves a user by username.
// Returns sql.ErrNoRows if no match is found.
func (r *PostgresUserRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	user := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, username, password_hash FROM users WHERE username = $1`,
		username,
	).Scan(&user.ID, &user.Username, &user.PasswordHash)
	if err != nil {
		return nil, err
	}
	return user, nil
}
