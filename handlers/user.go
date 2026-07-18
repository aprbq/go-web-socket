package handlers

import "net/http"

type UserHandler interface {
	SearchUsers(w http.ResponseWriter, r *http.Request)
}
