package main

import (
	"crypto/rand"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

type Client struct {
	ID   string
	mu   *sync.RWMutex
	conn *websocket.Conn
}

func NewClient(conn *websocket.Conn) *Client {
	ID := rand.Text()[:9]
	return &Client{
		ID:   ID,
		mu:   new(sync.RWMutex),
		conn: conn,
	}
}

type Server struct {
	clients []*Client
	mu      *sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		clients: []*Client{},
		mu:      new(sync.RWMutex),
	}
}

func handlerWS(r http.ResponseWriter, w *http.Request) {
	//client
}

func main() {
	http.HandleFunc("/", handlerWS)
}
