package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/aprbq/go-web-socket/repositories"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type authService struct {
	userRepo  repositories.UserRepository
	jwtSecret []byte
}

func NewAuthService(userRepo repositories.UserRepository, jwtSecret string) AuthService {
	return authService{userRepo: userRepo, jwtSecret: []byte(jwtSecret)}
}

func (s authService) Register(username, password string) (*User, error) {
	if username == "" || password == "" {
		return nil, errors.New("username and password are required")
	}

	_, err := s.userRepo.GetUserByUsername(username)
	if err == nil {
		return nil, ErrUsernameTaken
	}
	if !errors.Is(err, repositories.ErrUserNotFound) {
		return nil, err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	u, err := s.userRepo.CreateUser(username, string(hash))
	if err != nil {
		return nil, err
	}

	return &User{ID: u.ID, Username: u.Username}, nil
}

func (s authService) Login(username, password string) (*AuthResult, error) {
	u, err := s.userRepo.GetUserByUsername(username)
	if errors.Is(err, repositories.ErrUserNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	claims := jwt.MapClaims{
		"sub":      u.ID,
		"username": u.Username,
		"exp":      time.Now().Add(24 * time.Hour).Unix(),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.jwtSecret)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  User{ID: u.ID, Username: u.Username},
	}, nil
}

func (s authService) SearchUsers(q string, excludeID uint) ([]User, error) {
	usersDB, err := s.userRepo.SearchUsers(q, excludeID, 20)
	if err != nil {
		return nil, err
	}

	users := []User{}
	for _, u := range usersDB {
		users = append(users, User{ID: u.ID, Username: u.Username})
	}
	return users, nil
}

func (s authService) ValidateToken(tokenStr string) (*User, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.jwtSecret, nil
	})
	if err != nil || !token.Valid {
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrInvalidToken
	}

	// JSON numbers unmarshal as float64
	sub, ok := claims["sub"].(float64)
	if !ok {
		return nil, ErrInvalidToken
	}
	username, ok := claims["username"].(string)
	if !ok {
		return nil, ErrInvalidToken
	}

	return &User{ID: uint(sub), Username: username}, nil
}
