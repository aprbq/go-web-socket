package services

import "errors"

// User is the DTO exposed to handlers — never the repo model.
type User struct {
	ID       uint   `json:"id"`
	Username string `json:"username"`
}

type AuthResult struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUsernameTaken      = errors.New("username already taken")
	ErrInvalidToken       = errors.New("invalid or expired token")
)

type AuthService interface {
	Register(username, password string) (*User, error)
	Login(username, password string) (*AuthResult, error)
	ValidateToken(token string) (*User, error)
	SearchUsers(q string, excludeID uint) ([]User, error)
}
