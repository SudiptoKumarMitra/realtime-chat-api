-- 001_create_users.sql
-- Creates the users table for authentication.

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY,
    username      VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT NOW()
);

-- Index on username for fast login lookups.
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);
