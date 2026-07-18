package handlers

import "net/http"

type DirectHandler interface {
	GetHistory(w http.ResponseWriter, r *http.Request)
	GetConversations(w http.ResponseWriter, r *http.Request)
}
