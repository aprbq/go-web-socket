package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/aprbq/go-web-socket/services"
)

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type authHandler struct {
	authSrv services.AuthService
}

func NewAuthHandler(authSrv services.AuthService) AuthHandler {
	return authHandler{authSrv: authSrv}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func (h authHandler) Register(w http.ResponseWriter, r *http.Request) {
	creds := new(credentials)
	if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	user, err := h.authSrv.Register(creds.Username, creds.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (h authHandler) Login(w http.ResponseWriter, r *http.Request) {
	creds := new(credentials)
	if err := json.NewDecoder(r.Body).Decode(creds); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	result, err := h.authSrv.Login(creds.Username, creds.Password)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	writeJSON(w, http.StatusOK, result)
}
