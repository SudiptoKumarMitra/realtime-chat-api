package model

// User represents a registered user.
// PasswordHash is excluded from JSON responses.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"-"`
}
