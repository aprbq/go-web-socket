package services

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/aprbq/go-web-socket/repositories"
)

type room struct {
	clients map[string]ClientSession
	ID      string

	// test data
	clientsCount *atomic.Int64
}

func newRoom(id string) *room {
	return &room{
		ID:           id,
		clients:      map[string]ClientSession{},
		clientsCount: new(atomic.Int64),
	}
}

type chatService struct {
	msgRepo       repositories.MessageRepository
	userRepo      repositories.UserRepository
	clients       map[string]ClientSession
	rooms         map[string]*room
	mu            *sync.RWMutex
	joinServerCH  chan ClientSession
	leaveServerCH chan ClientSession
	broadcastCH   chan *ReqMsg
	joinRoomCH    chan *ReqMsg
	leaveRoomCH   chan *ReqMsg
	roomMsgCH     chan *ReqMsg
	directMsgCH   chan *ReqMsg

	reqCH         chan struct{}
	clientCountCH chan int
}

func NewChatService(msgRepo repositories.MessageRepository, userRepo repositories.UserRepository) ChatService {
	return &chatService{
		msgRepo:       msgRepo,
		userRepo:      userRepo,
		clients:       map[string]ClientSession{},
		rooms:         map[string]*room{},
		mu:            new(sync.RWMutex),
		joinServerCH:  make(chan ClientSession, 64),
		leaveServerCH: make(chan ClientSession, 64),
		broadcastCH:   make(chan *ReqMsg, 64),
		joinRoomCH:    make(chan *ReqMsg, 64),
		leaveRoomCH:   make(chan *ReqMsg, 64),
		roomMsgCH:     make(chan *ReqMsg, 64),
		directMsgCH:   make(chan *ReqMsg, 64),

		// for testing
		reqCH:         make(chan struct{}),
		clientCountCH: make(chan int),
	}
}

func (s *chatService) Join(c ClientSession) {
	s.joinServerCH <- c
}

func (s *chatService) Leave(c ClientSession) {
	s.leaveServerCH <- c
}

func (s *chatService) Dispatch(msg *ReqMsg) {
	switch msg.MsgType {
	case MsgType_Broadcast:
		s.broadcastCH <- msg
	case MsgType_JoinRoom:
		s.joinRoomCH <- msg
	case MsgType_LeaveRoom:
		s.leaveRoomCH <- msg
	case MsgType_RoomMsg:
		s.roomMsgCH <- msg
	case MsgType_DirectMsg:
		s.directMsgCH <- msg
	default:
		fmt.Println("unknown msg type -> ignoring it!")
		// TODO -> return err to client?
	}
}

func (s *chatService) AcceptLoop() {
	for {
		select {
		case c := <-s.joinServerCH:
			s.joinServer(c)
		case c := <-s.leaveServerCH:
			s.leaveServer(c)
		case msg := <-s.joinRoomCH:
			s.joinRoom(msg)
		case msg := <-s.leaveRoomCH:
			s.leaveRoom(msg)
		case msg := <-s.roomMsgCH:
			s.roomMsg(msg)
		case msg := <-s.directMsgCH:
			s.directMsg(msg)
		case msg := <-s.broadcastCH:
			s.broadcast(msg)
		case <-s.reqCH:
			s.clientCountCH <- len(s.clients)
		}
	}
}

func (s *chatService) joinServer(c ClientSession) {
	s.clients[c.ID()] = c
	fmt.Printf("client joined the server, cID = %s\n", c.ID())
}

func (s *chatService) leaveServer(c ClientSession) {
	delete(s.clients, c.ID())

	for _, r := range s.rooms {
		_, ok := r.clients[c.ID()]
		if ok {
			delete(r.clients, c.ID())
		}
	}

	fmt.Printf("client left the server, cID = %s\n", c.ID())
}

func (s *chatService) broadcast(msg *ReqMsg) {
	cls := map[string]ClientSession{}
	for id, c := range s.clients {
		if id != msg.Client.ID() {
			cls[id] = c
		}
	}

	go s.sendMsg(msg, cls)
	go s.persistMsg(msg)
	fmt.Println("broadcast was sent")
}

func (s *chatService) roomMsg(msg *ReqMsg) {
	rID := msg.RoomID
	room, ok := s.rooms[rID]
	if !ok {
		fmt.Printf("the room does not exist -> cannot send msg into it")
		return
	}

	_, ok = room.clients[msg.Client.ID()]
	if !ok {
		fmt.Printf("the cleint = %s does not belong to the room %s -> cannot send msg into it\n", msg.Client.ID(), rID)
		return

	}

	cls := map[string]ClientSession{}
	for id, c := range room.clients {
		if id != msg.Client.ID() {
			cls[id] = c
		}
	}

	go s.sendMsg(msg, cls)
	go s.persistMsg(msg)
	fmt.Printf("the cleint = %s sent msg to the room %s\n", msg.Client.ID(), rID)
}

// directMsg delivers the msg to every connection of the recipient user
// (they may have several tabs open) and to the sender's other connections.
// The message is persisted even when the recipient is offline.
func (s *chatService) directMsg(msg *ReqMsg) {
	cls := map[string]ClientSession{}
	for id, c := range s.clients {
		sameUser := c.UserID() == msg.Client.UserID()
		if (c.Username() == msg.To || sameUser) && id != msg.Client.ID() {
			cls[id] = c
		}
	}

	go s.sendMsg(msg, cls)
	go s.persistDirectMsg(msg)
	fmt.Printf("the client = %s sent direct msg to %s\n", msg.Client.ID(), msg.To)
}

// persistDirectMsg runs in its own goroutine — the recipient lookup and the
// DB write must not block the AcceptLoop.
func (s *chatService) persistDirectMsg(msg *ReqMsg) {
	recipient, err := s.userRepo.GetUserByUsername(msg.To)
	if err != nil {
		fmt.Printf("unable to persist direct msg, recipient %q: %v\n", msg.To, err)
		return
	}

	err = s.msgRepo.SaveDirectMessage(
		msg.Client.UserID(), msg.Client.Username(),
		recipient.ID, recipient.Username,
		msgDataString(msg),
	)
	if err != nil {
		fmt.Printf("unable to persist direct msg %v\n", err)
	}
}

// persistMsg runs in its own goroutine — a slow DB write must not block the AcceptLoop.
func (s *chatService) persistMsg(msg *ReqMsg) {
	err := s.msgRepo.SaveMessage(msg.Client.UserID(), msg.Client.Username(), msg.RoomID, msgDataString(msg))
	if err != nil {
		fmt.Printf("unable to persist msg %v\n", err)
	}
}

func msgDataString(msg *ReqMsg) string {
	data, ok := msg.Data.(string)
	if !ok {
		b, err := json.Marshal(msg.Data)
		if err != nil {
			fmt.Printf("unable to marshal msg data for persisting %v\n", err)
			return ""
		}
		data = string(b)
	}
	return data
}

func (s *chatService) GetRoomMessages(roomID string) ([]Message, error) {
	msgsDB, err := s.msgRepo.GetRoomMessages(roomID, 50)
	if err != nil {
		return nil, err
	}

	// map repo model -> DTO; the repo model never leaves the repositories layer
	msgs := []Message{}
	for _, m := range msgsDB {
		msgs = append(msgs, Message{
			SenderName: m.SenderName,
			RoomID:     m.RoomID,
			Data:       m.Data,
			CreatedAt:  m.CreatedAt,
		})
	}
	return msgs, nil
}

func (s *chatService) sendMsg(msg *ReqMsg, cls map[string]ClientSession) {
	resp := NewRespMsg(msg)
	for _, c := range cls {
		c.Send(resp)
	}
	cls = nil
}

func (s *chatService) joinRoom(msg *ReqMsg) {
	rID := msg.RoomID
	room, ok := s.rooms[rID]
	if !ok {
		room = newRoom(rID)
		s.rooms[rID] = room
	}

	room.clients[msg.Client.ID()] = msg.Client
	room.clientsCount.Add(1)
	fmt.Printf("client joined the Room %s, cID = %s\n", rID, msg.Client.ID())
}

func (s *chatService) leaveRoom(msg *ReqMsg) {
	rID := msg.RoomID
	room, ok := s.rooms[rID]
	if !ok {
		fmt.Printf("cannot leave room that does not exist rID = %s, cID = %s\n", rID, msg.Client.ID())
		return
	}
	delete(room.clients, msg.Client.ID())
	room.clientsCount.Add(-1)
	fmt.Printf("client left the room rID = %s, cID = %s\n", rID, msg.Client.ID())
}

func (s *chatService) GetDirectMessages(meID uint, peerUsername string) ([]Message, error) {
	peer, err := s.userRepo.GetUserByUsername(peerUsername)
	if err != nil {
		return nil, err
	}

	msgsDB, err := s.msgRepo.GetDirectMessages(meID, peer.ID, 50)
	if err != nil {
		return nil, err
	}

	msgs := []Message{}
	for _, m := range msgsDB {
		msgs = append(msgs, Message{
			SenderName: m.SenderName,
			Data:       m.Data,
			CreatedAt:  m.CreatedAt,
		})
	}
	return msgs, nil
}

func (s *chatService) GetConversations(meID uint) ([]Conversation, error) {
	msgsDB, err := s.msgRepo.GetLatestDirectMessages(meID)
	if err != nil {
		return nil, err
	}

	convs := []Conversation{}
	for _, m := range msgsDB {
		peerName := m.SenderName
		if m.SenderID == meID {
			peerName = m.RecipientName
		}
		convs = append(convs, Conversation{
			PeerName:    peerName,
			LastMessage: m.Data,
			LastSender:  m.SenderName,
			LastAt:      m.CreatedAt,
		})
	}
	return convs, nil
}

func (s *chatService) GetRoomTestResults(roomID string) *TestRoomResults {
	room := s.rooms[roomID]
	return &TestRoomResults{
		RoomID:       roomID,
		ClientsCount: int(room.clientsCount.Load()),
	}
}

func (s *chatService) GetServerTestResults() int {
	s.reqCH <- struct{}{}
	return <-s.clientCountCH
}
