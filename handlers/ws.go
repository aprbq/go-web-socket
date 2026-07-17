package handlers

import "net/http"

type WSHandler interface {
	HandleWS(w http.ResponseWriter, r *http.Request)
}
