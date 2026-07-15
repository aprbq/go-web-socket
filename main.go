package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	WSPort = "localhost:3223"
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
	clients      map[string]*Client
	mu           *sync.RWMutex
	joinServerCH chan *Client
	leaveServeCH chan *Client
}

func NewServer() *Server {
	return &Server{
		clients:      map[string]*Client{},
		mu:           new(sync.RWMutex),
		joinServerCH: make(chan *Client, 64),
		leaveServeCH: make(chan *Client, 64),
	}
}

func (s *Server) handlerWS(w http.ResponseWriter, r *http.Request) {
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

	client := NewClient(conn)
	s.joinServerCH <- client
	// add client
	// read msg loop
}

func (s *Server) AcceptLoop() {
	for {
		select {
		case c := <-s.joinServerCH:
			s.joinServer(c)
		case c := <-s.leaveServeCH:
			s.leaveServer(c)
		}
	}
}

func (s *Server) joinServer(c *Client) {
	s.clients[c.ID] = c
	fmt.Printf("client joined the server, cID = %v\n", c.ID)
}

func (s *Server) leaveServer(c *Client) {
	delete(s.clients, c.ID)
	fmt.Printf("client left the server, cID = %v\n", c.ID)
}

func createWSServer() {
	s := NewServer()
	go s.AcceptLoop()
	http.HandleFunc("/", s.handlerWS)

	fmt.Printf("starting server on port: %v\n", WSPort)
	log.Fatal(http.ListenAndServe(WSPort, nil))
}

func main() {
	createWSServer()
}
