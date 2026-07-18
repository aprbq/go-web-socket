package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/aprbq/go-web-socket/services"
)

// bearerUser authenticates a REST request from its Authorization header.
func bearerUser(r *http.Request, authSrv services.AuthService) (*services.User, error) {
	token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found {
		return nil, errors.New("missing bearer token")
	}
	return authSrv.ValidateToken(token)
}
