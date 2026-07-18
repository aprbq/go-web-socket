package handlers

import (
	"errors"
	"net/http"

	"github.com/aprbq/go-web-socket/repositories"
	"github.com/aprbq/go-web-socket/services"
)

type directHandler struct {
	chatSrv services.ChatService
	authSrv services.AuthService
}

func NewDirectHandler(chatSrv services.ChatService, authSrv services.AuthService) DirectHandler {
	return directHandler{chatSrv: chatSrv, authSrv: authSrv}
}

// GetHistory only ever returns conversations the requesting user is part of —
// the user id comes from the token, never from the request.
func (h directHandler) GetHistory(w http.ResponseWriter, r *http.Request) {
	me, err := bearerUser(r, h.authSrv)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	peer := r.URL.Query().Get("peer")
	if peer == "" {
		writeError(w, http.StatusBadRequest, errors.New("peer query param is required"))
		return
	}

	msgs, err := h.chatSrv.GetDirectMessages(me.ID, peer)
	if errors.Is(err, repositories.ErrUserNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": msgs})
}

func (h directHandler) GetConversations(w http.ResponseWriter, r *http.Request) {
	me, err := bearerUser(r, h.authSrv)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	convs, err := h.chatSrv.GetConversations(me.ID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"conversations": convs})
}
