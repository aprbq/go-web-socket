package handlers

import (
	"errors"
	"net/http"

	"github.com/aprbq/go-web-socket/services"
)

type userHandler struct {
	authSrv services.AuthService
}

func NewUserHandler(authSrv services.AuthService) UserHandler {
	return userHandler{authSrv: authSrv}
}

func (h userHandler) SearchUsers(w http.ResponseWriter, r *http.Request) {
	me, err := bearerUser(r, h.authSrv)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	q := r.URL.Query().Get("q")
	if q == "" {
		writeError(w, http.StatusBadRequest, errors.New("q query param is required"))
		return
	}

	users, err := h.authSrv.SearchUsers(q, me.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"users": users})
}
