package services

type MsgType string

const (
	MsgType_Broadcast MsgType = "broadcast"
	MsgType_JoinRoom  MsgType = "join-room"
	MsgType_LeaveRoom MsgType = "leave-room"
	MsgType_RoomMsg   MsgType = "room-message"
)

type ReqMsg struct {
	MsgType MsgType       `json:"type"`
	Client  ClientSession `json:"-"`
	Data    interface{}   `json:"data"`
	RoomID  string        `json:"roomID"`
}

type RespMsg struct {
	MsgType  MsgType     `json:"type"`
	Data     interface{} `json:"data"`
	SenderID string      `json:"senderID"`
	RoomID   string      `json:"roomID"`
}

func NewRespMsg(msg *ReqMsg) *RespMsg {
	return &RespMsg{
		MsgType:  msg.MsgType,
		Data:     msg.Data,
		SenderID: msg.Client.ID(),
		RoomID:   msg.RoomID,
	}
}

// ClientSession is the outbound port a connected client must satisfy;
// the websocket adapter in handlers implements it.
type ClientSession interface {
	ID() string
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

	// test helpers
	GetRoomTestResults(roomID string) *TestRoomResults
	GetServerTestResults() int
}
