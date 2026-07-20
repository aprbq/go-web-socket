package services

import (
	"errors"
	"testing"
	"time"

	"github.com/aprbq/go-web-socket/repositories"
)

// fakeClient implements ClientSession without any websocket — this is the
// payoff of the hexagonal design: chat logic is testable in complete isolation.
type fakeClient struct {
	id       string
	userID   uint
	username string
	msgs     chan *RespMsg
}

func newFakeClient(id string, userID uint, username string) *fakeClient {
	return &fakeClient{id: id, userID: userID, username: username, msgs: make(chan *RespMsg, 16)}
}

func (f *fakeClient) ID() string       { return f.id }
func (f *fakeClient) UserID() uint     { return f.userID }
func (f *fakeClient) Username() string { return f.username }
func (f *fakeClient) Send(m *RespMsg)  { f.msgs <- m }

func (f *fakeClient) expectMsg(t *testing.T) *RespMsg {
	t.Helper()
	select {
	case m := <-f.msgs:
		return m
	case <-time.After(2 * time.Second):
		t.Fatalf("client %s timed out waiting for a message", f.id)
		return nil
	}
}

func (f *fakeClient) expectNoMsg(t *testing.T) {
	t.Helper()
	select {
	case m := <-f.msgs:
		t.Fatalf("client %s got an unexpected message: %+v", f.id, m)
	case <-time.After(150 * time.Millisecond):
	}
}

func newTestChatService() (ChatService, repositories.UserRepository, repositories.MessageRepository) {
	userRepo := repositories.NewUserRepositoryMock()
	msgRepo := repositories.NewMessageRepositoryMock()
	chatSrv := NewChatService(msgRepo, userRepo)
	go chatSrv.AcceptLoop()
	return chatSrv, userRepo, msgRepo
}

func TestDirectMsgDelivery(t *testing.T) {
	chatSrv, userRepo, _ := newTestChatService()
	userRepo.CreateUser("alice", "x") // id 1
	userRepo.CreateUser("bob", "x")   // id 2

	alice := newFakeClient("conn-a", 1, "alice")
	aliceTab2 := newFakeClient("conn-a2", 1, "alice") // แท็บที่สองของ alice
	bob := newFakeClient("conn-b", 2, "bob")
	chatSrv.Join(alice)
	chatSrv.Join(aliceTab2)
	chatSrv.Join(bob)
	time.Sleep(100 * time.Millisecond) // ให้ AcceptLoop ประมวลผล join ก่อน

	chatSrv.Dispatch(&ReqMsg{MsgType: MsgType_DirectMsg, Client: alice, To: "bob", Data: "hi bob"})

	// ผู้รับต้องได้ข้อความพร้อมตัวตนผู้ส่งที่ถูกต้อง
	got := bob.expectMsg(t)
	if got.SenderName != "alice" || got.To != "bob" || got.Data != "hi bob" {
		t.Fatalf("unexpected message: %+v", got)
	}

	// แท็บอื่นของผู้ส่งต้องได้ echo (เพื่อ sync ข้ามแท็บ)
	echo := aliceTab2.expectMsg(t)
	if echo.Data != "hi bob" {
		t.Fatalf("unexpected echo: %+v", echo)
	}

	// แต่ connection ที่เป็นคนส่งเองต้องไม่ได้รับสะท้อนกลับ
	alice.expectNoMsg(t)
}

func TestDirectMsgPersistsWhenRecipientOffline(t *testing.T) {
	chatSrv, userRepo, _ := newTestChatService()
	alice, _ := userRepo.CreateUser("alice", "x")
	userRepo.CreateUser("bob", "x") // bob มีตัวตนแต่ไม่ออนไลน์

	conn := newFakeClient("conn-a", alice.ID, "alice")
	chatSrv.Join(conn)
	time.Sleep(100 * time.Millisecond)

	chatSrv.Dispatch(&ReqMsg{MsgType: MsgType_DirectMsg, Client: conn, To: "bob", Data: "see you later"})

	// persist ทำงานใน goroutine แยก — poll จนกว่าจะเจอ (หรือ timeout)
	deadline := time.Now().Add(2 * time.Second)
	for {
		msgs, err := chatSrv.GetDirectMessages(alice.ID, "bob")
		if err == nil && len(msgs) == 1 {
			if msgs[0].SenderName != "alice" || msgs[0].Data != "see you later" {
				t.Fatalf("unexpected history: %+v", msgs)
			}
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("message was never persisted (err=%v msgs=%v)", err, msgs)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func TestGetDirectMessagesUnknownPeer(t *testing.T) {
	chatSrv, userRepo, _ := newTestChatService()
	alice, _ := userRepo.CreateUser("alice", "x")

	_, err := chatSrv.GetDirectMessages(alice.ID, "nobody")
	if !errors.Is(err, repositories.ErrUserNotFound) {
		t.Fatalf("want ErrUserNotFound, got %v", err)
	}
}

func TestGetConversations(t *testing.T) {
	chatSrv, _, msgRepo := newTestChatService()

	// seed ประวัติผ่าน repo ตรงๆ: alice(1) คุยกับ bob(2) แล้ว carol(3) ทักมา
	msgRepo.SaveDirectMessage(1, "alice", 2, "bob", "hi bob")
	msgRepo.SaveDirectMessage(3, "carol", 1, "alice", "hey alice")

	convs, err := chatSrv.GetConversations(1)
	if err != nil {
		t.Fatalf("conversations: %v", err)
	}
	if len(convs) != 2 {
		t.Fatalf("want 2 conversations, got %+v", convs)
	}

	// เรียงบทสนทนาล่าสุดก่อน และ peerName ต้องเป็น "อีกฝ่าย" เสมอ
	// ไม่ว่าเราจะเป็นผู้ส่งหรือผู้รับข้อความล่าสุดของคู่นั้น
	if convs[0].PeerName != "carol" || convs[0].LastSender != "carol" {
		t.Fatalf("want carol first, got %+v", convs[0])
	}
	if convs[1].PeerName != "bob" || convs[1].LastSender != "alice" {
		t.Fatalf("want bob with alice as last sender, got %+v", convs[1])
	}
}
