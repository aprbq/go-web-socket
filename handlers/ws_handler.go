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

// client is the websocket adapter implementing services.ClientSession
type client struct {
	id    string
	mu    *sync.RWMutex
	conn  *websocket.Conn
	msgCH chan *services.RespMsg
	done  chan struct{}
}

func newClient(conn *websocket.Conn) *client {
	id := rand.Text()[:9]
	return &client{
		id:    id,
		mu:    new(sync.RWMutex),
		conn:  conn,
		msgCH: make(chan *services.RespMsg, 64),
		done:  make(chan struct{}),
	}

}

func (c *client) ID() string {
	return c.id
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
}

func NewWSHandler(chatSrv services.ChatService) WSHandler {
	return &wsHandler{chatSrv: chatSrv}
}

func (h *wsHandler) HandleWS(w http.ResponseWriter, r *http.Request) {
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

	c := newClient(conn)
	h.chatSrv.Join(c)

	go c.writeMsgLoop()
	go c.readMsgLoop(h.chatSrv)
}
