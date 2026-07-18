package services

import "time"

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_JoinRoom  MsgType = "join-room"
	MsgType_LeaveRoom MsgType = "leave-room"
	MsgType_RoomMsg   MsgType = "room-message"
	MsgType_DirectMsg MsgType = "direct-message"
)

type ReqMsg struct {
	MsgType MsgType       `json:"type"`
	Client  ClientSession `json:"-"`
	Data    interface{}   `json:"data"`
	RoomID  string        `json:"roomID"`
	// To is the recipient username for direct messages
	To string `json:"to"`
}

type RespMsg struct {
	MsgType    MsgType     `json:"type"`
	Data       interface{} `json:"data"`
	SenderID   string      `json:"senderID"`
	SenderName string      `json:"senderName"`
	RoomID     string      `json:"roomID"`
	To         string      `json:"to,omitempty"`
}

func NewRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:    msg.MsgType,
		Data:       msg.Data,
		SenderID:   msg.Client.ID(),
		SenderName: msg.Client.Username(),
		RoomID:     msg.RoomID,
		To:         msg.To,
	}
}

// Message is the DTO for chat history exposed to handlers.
type Message struct {
	SenderName string    `json:"senderName"`
	RoomID     string    `json:"roomID"`
	Data       string    `json:"data"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Conversation is one entry of a user's Messenger-style chat list.
type Conversation struct {
	PeerName    string    `json:"peerName"`
	LastMessage string    `json:"lastMessage"`
	LastSender  string    `json:"lastSender"`
	LastAt      time.Time `json:"lastAt"`
}

// ClientSession is the outbound port a connected client must satisfy;
// the websocket adapter in handlers implements it.
// ID is the unique connection id (one user may have many connections),
// UserID/Username identify the authenticated user behind it.
type ClientSession interface {
	ID() string
	UserID() uint
	Username() string
	Send(msg *RespMsg)
}

type TestRoomResults struct {
	RoomID       string
	ClientsCount int
}

type ChatService interface {
	AcceptLoop()
	Join(c ClientSession)
	Leave(c ClientSession)
	Dispatch(msg *ReqMsg)
	GetRoomMessages(roomID string) ([]Message, error)
	GetDirectMessages(meID uint, peerUsername string) ([]Message, error)
	GetConversations(meID uint) ([]Conversation, error)

	// test helpers
	GetRoomTestResults(roomID string) *TestRoomResults
	GetServerTestResults() int
}
