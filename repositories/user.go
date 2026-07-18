package repositories

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// user is the DB schema model — it never leaves this layer.
type user struct {
	ID           uint   `gorm:"primaryKey"`
	Username     string `gorm:"uniqueIndex"`
	PasswordHash string
	CreatedAt    time.Time
}

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	CreateUser(username, passwordHash string) (*user, error)
	GetUserByUsername(username string) (*user, error)
	// SearchUsers finds users whose username contains q, excluding excludeID.
	SearchUsers(q string, excludeID uint, limit int) ([]user, error)
}

// userRepositoryMock is an in-memory implementation for tests.
type userRepositoryMock struct {
	mu     sync.Mutex
	users  map[string]*user
	nextID uint
}

func NewUserRepositoryMock() UserRepository {
	return &userRepositoryMock{users: map[string]*user{}, nextID: 1}
}

func (r *userRepositoryMock) CreateUser(username, passwordHash string) (*user, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[username]; ok {
		return nil, errors.New("username already exists")
	}

	u := &user{
		ID:           r.nextID,
		Username:     username,
		PasswordHash: passwordHash,
		CreatedAt:    time.Now(),
	}
	r.nextID++
	r.users[username] = u
	return u, nil
}

func (r *userRepositoryMock) GetUserByUsername(username string) (*user, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	u, ok := r.users[username]
	if !ok {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (r *userRepositoryMock) SearchUsers(q string, excludeID uint, limit int) ([]user, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	users := []user{}
	for _, u := range r.users {
		if u.ID != excludeID && strings.Contains(strings.ToLower(u.Username), strings.ToLower(q)) {
			users = append(users, *u)
			if len(users) == limit {
				break
			}
		}
	}
	return users, nil
}
