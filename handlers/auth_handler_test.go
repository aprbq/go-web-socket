package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aprbq/go-web-socket/repositories"
	"github.com/aprbq/go-web-socket/services"
)

func newTestAuthHandler() AuthHandler {
	authSrv := services.NewAuthService(repositories.NewUserRepositoryMock(), "test-secret")
	return NewAuthHandler(authSrv)
}

// post ยิง HTTP request ปลอมใส่ handler ตรงๆ ผ่าน httptest —
// ไม่มี server ไม่มีพอร์ต แต่ handler ทำงานเหมือนโดน request จริงทุกอย่าง
func post(h http.HandlerFunc, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func TestRegisterEndpoint(t *testing.T) {
	h := newTestAuthHandler()

	t.Run("success returns 201 with the user", func(t *testing.T) {
		rec := post(h.Register, "/register", `{"username":"alice","password":"1234"}`)
		if rec.Code != http.StatusCreated {
			t.Fatalf("want 201, got %d: %s", rec.Code, rec.Body)
		}

		var u services.User
		if err := json.NewDecoder(rec.Body).Decode(&u); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if u.Username != "alice" || u.ID == 0 {
			t.Fatalf("unexpected user: %+v", u)
		}
	})

	t.Run("duplicate username returns 400 with error json", func(t *testing.T) {
		rec := post(h.Register, "/register", `{"username":"alice","password":"1234"}`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body)
		}
		if !strings.Contains(rec.Body.String(), "error") {
			t.Fatalf("want error json, got %s", rec.Body)
		}
	})

	t.Run("invalid json returns 400", func(t *testing.T) {
		rec := post(h.Register, "/register", `not-json`)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("want 400, got %d: %s", rec.Code, rec.Body)
		}
	})
}

func TestLoginEndpoint(t *testing.T) {
	h := newTestAuthHandler()
	post(h.Register, "/register", `{"username":"alice","password":"1234"}`)

	t.Run("success returns 200 with a token", func(t *testing.T) {
		rec := post(h.Login, "/login", `{"username":"alice","password":"1234"}`)
		if rec.Code != http.StatusOK {
			t.Fatalf("want 200, got %d: %s", rec.Code, rec.Body)
		}

		var res services.AuthResult
		if err := json.NewDecoder(rec.Body).Decode(&res); err != nil {
			t.Fatalf("bad json: %v", err)
		}
		if res.Token == "" || res.User.Username != "alice" {
			t.Fatalf("unexpected result: %+v", res)
		}
	})

	t.Run("wrong password returns 401", func(t *testing.T) {
		rec := post(h.Login, "/login", `{"username":"alice","password":"wrong"}`)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("want 401, got %d: %s", rec.Code, rec.Body)
		}
	})
}
