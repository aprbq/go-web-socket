package handlers

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/aprbq/go-web-socket/services"
	"github.com/gorilla/websocket"
)

// client is the websocket adapter implementing services.ClientSession.
// id is unique per connection; userID/username come from the validated JWT,
// so the same user can be connected from several tabs at once.
type client struct {
	id       string
	userID   uint
	username string
	mu       *sync.RWMutex
	conn     *websocket.Conn
	msgCH    chan *services.RespMsg
	done     chan struct{}
}

func newClient(conn *websocket.Conn, user *services.User) *client {
	id := rand.Text()[:9]
	return &client{
		id:       id,
		userID:   user.ID,
		username: user.Username,
		mu:       new(sync.RWMutex),
		conn:     conn,
		msgCH:    make(chan *services.RespMsg, 64),
		done:     make(chan struct{}),
	}

}

func (c *client) ID() string {
	return c.id
}

func (c *client) UserID() uint {
	return c.userID
}

func (c *client) Username() string {
	return c.username
}

func (c *client) Send(msg *services.RespMsg) {
	c.msgCH <- msg
}

func (c *client) writeMsgLoop() {
	defer c.conn.Close()
	for {
		select {
		case <-c.done:
			return
		case msg := <-c.msgCH:
			err := c.conn.WriteJSON(msg)
			if err != nil {
				fmt.Printf("error sending msg to clientID = %s\n", c.id)
				return
			}
		}
	}
}

func (c *client) readMsgLoop(chatSrv services.ChatService) {
	defer func() {
		close(c.done)
		chatSrv.Leave(c)
	}()

	for {
		_, b, err := c.conn.ReadMessage()
		if err != nil {
			return
		}

		msg := new(services.ReqMsg)
		err = json.Unmarshal(b, msg)
		if err != nil {
			fmt.Printf("unable to unmarshal the msg %v\n", err)
			continue
		}
		msg.Client = c

		chatSrv.Dispatch(msg)
	}
}

type wsHandler struct {
	chatSrv services.ChatService
	authSrv services.AuthService
}

func NewWSHandler(chatSrv services.ChatService, authSrv services.AuthService) WSHandler {
	return &wsHandler{chatSrv: chatSrv, authSrv: authSrv}
}

func (h *wsHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
	// browser WebSocket API cannot set an Authorization header,
	// so the JWT comes in as a query param — validated BEFORE the upgrade
	user, err := h.authSrv.ValidateToken(r.URL.Query().Get("token"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  512,
		WriteBufferSize: 512,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Error on HTTP conn upgrade %v\n", err)
		return
	}

	c := newClient(conn, user)
	h.chatSrv.Join(c)

	go c.writeMsgLoop()
	go c.readMsgLoop(h.chatSrv)
}
