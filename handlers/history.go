package handlers

import "net/http"

type HistoryHandler interface {
	GetRoomMessages(w http.ResponseWriter, r *http.Request)
}
