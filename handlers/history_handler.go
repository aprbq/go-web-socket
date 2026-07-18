package handlers

import (
	"errors"
	"net/http"

	"github.com/aprbq/go-web-socket/services"
)

type historyHandler struct {
	chatSrv services.ChatService
	authSrv services.AuthService
}

func NewHistoryHandler(chatSrv services.ChatService, authSrv services.AuthService) HistoryHandler {
	return historyHandler{chatSrv: chatSrv, authSrv: authSrv}
}

func (h historyHandler) GetRoomMessages(w http.ResponseWriter, r *http.Request) {
	_, err := bearerUser(r, h.authSrv)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return
	}

	roomID := r.URL.Query().Get("roomID")
	if roomID == "" {
		writeError(w, http.StatusBadRequest, errors.New("roomID query param is required"))
		return
	}

	msgs, err := h.chatSrv.GetRoomMessages(roomID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{"messages": msgs})
}
