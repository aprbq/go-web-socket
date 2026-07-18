package services

import (
	"encoding/json"
	"fmt"

	"github.com/aprbq/go-web-socket/repositories"
)

type chatService struct {
	msgRepo       repositories.MessageRepository
	userRepo      repositories.UserRepository
	clients       map[string]ClientSession
	joinServerCH  chan ClientSession
	leaveServerCH chan ClientSession
	directMsgCH   chan *ReqMsg
}

func NewChatService(msgRepo repositories.MessageRepository, userRepo repositories.UserRepository) ChatService {
	return &chatService{
		msgRepo:       msgRepo,
		userRepo:      userRepo,
		clients:       map[string]ClientSession{},
		joinServerCH:  make(chan ClientSession, 64),
		leaveServerCH: make(chan ClientSession, 64),
		directMsgCH:   make(chan *ReqMsg, 64),
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
		case msg := <-s.directMsgCH:
			s.directMsg(msg)
		}
	}
}

func (s *chatService) joinServer(c ClientSession) {
	s.clients[c.ID()] = c
	fmt.Printf("client joined the server, cID = %s\n", c.ID())
}

func (s *chatService) leaveServer(c ClientSession) {
	delete(s.clients, c.ID())
	fmt.Printf("client left the server, cID = %s\n", c.ID())
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

func (s *chatService) sendMsg(msg *ReqMsg, cls map[string]ClientSession) {
	resp := NewRespMsg(msg)
	for _, c := range cls {
		c.Send(resp)
	}
	cls = nil
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
