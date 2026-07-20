package services

import (
	"errors"
	"testing"
	"time"

	"github.com/aprbq/go-web-socket/repositories"
	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret"

func newTestAuthService() AuthService {
	return NewAuthService(repositories.NewUserRepositoryMock(), testSecret)
}

func TestRegister(t *testing.T) {
	authSrv := newTestAuthService()

	t.Run("success returns the new user", func(t *testing.T) {
		u, err := authSrv.Register("alice", "1234")
		if err != nil {
			t.Fatalf("register: %v", err)
		}
		if u.Username != "alice" || u.ID == 0 {
			t.Fatalf("unexpected user: %+v", u)
		}
	})

	t.Run("duplicate username is rejected", func(t *testing.T) {
		_, err := authSrv.Register("alice", "5678")
		if !errors.Is(err, ErrUsernameTaken) {
			t.Fatalf("want ErrUsernameTaken, got %v", err)
		}
	})

	t.Run("empty username or password is rejected", func(t *testing.T) {
		if _, err := authSrv.Register("", "1234"); err == nil {
			t.Fatal("empty username should fail")
		}
		if _, err := authSrv.Register("bob", ""); err == nil {
			t.Fatal("empty password should fail")
		}
	})
}

func TestLogin(t *testing.T) {
	authSrv := newTestAuthService()
	if _, err := authSrv.Register("alice", "1234"); err != nil {
		t.Fatalf("register: %v", err)
	}

	t.Run("correct password returns a token that validates back", func(t *testing.T) {
		res, err := authSrv.Login("alice", "1234")
		if err != nil {
			t.Fatalf("login: %v", err)
		}
		if res.Token == "" {
			t.Fatal("empty token")
		}

		u, err := authSrv.ValidateToken(res.Token)
		if err != nil {
			t.Fatalf("validate: %v", err)
		}
		if u.Username != "alice" || u.ID != res.User.ID {
			t.Fatalf("token does not round-trip: %+v vs %+v", u, res.User)
		}
	})

	t.Run("wrong password is rejected", func(t *testing.T) {
		_, err := authSrv.Login("alice", "wrong")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("want ErrInvalidCredentials, got %v", err)
		}
	})

	t.Run("unknown user is rejected with the same error", func(t *testing.T) {
		// สังเกตว่า user ไม่มีอยู่ก็ต้องได้ error เดียวกับรหัสผิด
		// เพื่อไม่ให้คนนอกเดาได้ว่า username ไหนมีอยู่จริง
		_, err := authSrv.Login("nobody", "1234")
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("want ErrInvalidCredentials, got %v", err)
		}
	})
}

func TestValidateTokenRejectsBadTokens(t *testing.T) {
	authSrv := newTestAuthService()

	t.Run("garbage token", func(t *testing.T) {
		if _, err := authSrv.ValidateToken("not-a-jwt"); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("want ErrInvalidToken, got %v", err)
		}
	})

	t.Run("token signed with another secret", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":      float64(1),
			"username": "alice",
			"exp":      time.Now().Add(time.Hour).Unix(),
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte("attacker-secret"))
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		if _, err := authSrv.ValidateToken(token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("want ErrInvalidToken, got %v", err)
		}
	})

	t.Run("expired token", func(t *testing.T) {
		claims := jwt.MapClaims{
			"sub":      float64(1),
			"username": "alice",
			"exp":      time.Now().Add(-time.Hour).Unix(), // หมดอายุไปแล้ว 1 ชม.
		}
		token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
		if err != nil {
			t.Fatalf("sign: %v", err)
		}

		if _, err := authSrv.ValidateToken(token); !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("want ErrInvalidToken, got %v", err)
		}
	})
}

func TestSearchUsers(t *testing.T) {
	authSrv := newTestAuthService()
	alice, _ := authSrv.Register("alice", "x")
	authSrv.Register("bob", "x")
	authSrv.Register("bobby", "x")

	t.Run("matches by substring", func(t *testing.T) {
		users, err := authSrv.SearchUsers("bob", alice.ID)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(users) != 2 {
			t.Fatalf("want bob and bobby, got %+v", users)
		}
	})

	t.Run("excludes the requesting user", func(t *testing.T) {
		users, err := authSrv.SearchUsers("alice", alice.ID)
		if err != nil {
			t.Fatalf("search: %v", err)
		}
		if len(users) != 0 {
			t.Fatalf("must not find myself, got %+v", users)
		}
	})
}
