package main

import (
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aprbq/go-web-socket/config"
	"github.com/aprbq/go-web-socket/handlers"
	"github.com/aprbq/go-web-socket/repositories"
	"github.com/aprbq/go-web-socket/services"
	"github.com/gorilla/websocket"
)

const host = "ws://localhost"

// the whole test binary shares one running server (createWSServer registers on
// the global http.DefaultServeMux and binds a port, so it can only run once).
var (
	testChatSrv services.ChatService
	testAuthSrv services.AuthService
)

func TestMain(m *testing.M) {
	userRepo := repositories.NewUserRepositoryMock()
	testAuthSrv = services.NewAuthService(userRepo, "test-secret")
	testChatSrv = services.NewChatService(repositories.NewMessageRepositoryMock(), userRepo)

	wsHdl := handlers.NewWSHandler(testChatSrv, testAuthSrv)
	authHdl := handlers.NewAuthHandler(testAuthSrv)
	userHdl := handlers.NewUserHandler(testAuthSrv)
	directHdl := handlers.NewDirectHandler(testChatSrv, testAuthSrv)

	go createWSServer(testChatSrv, wsHdl, authHdl, userHdl, directHdl)
	time.Sleep(1 * time.Second) // let the server bind the port

	os.Exit(m.Run())
}

// registerAndLogin creates a user and returns a valid JWT for it.
func registerAndLogin(t *testing.T, username string) string {
	t.Helper()
	if _, err := testAuthSrv.Register(username, "1234"); err != nil {
		t.Fatalf("register %s: %v", username, err)
	}
	res, err := testAuthSrv.Login(username, "1234")
	if err != nil {
		t.Fatalf("login %s: %v", username, err)
	}
	return res.Token
}

func dial(t *testing.T, token string) *websocket.Conn {
	t.Helper()
	url := fmt.Sprintf("%s%s/ws?token=%s", host, config.WSPort, token)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	return conn
}

// sendDM reports failures with Errorf (not Fatalf) so it is safe to call from
// sender goroutines the tests spawn; callers should stop sending on false.
func sendDM(t *testing.T, conn *websocket.Conn, to, data string) bool {
	t.Helper()
	err := conn.WriteJSON(map[string]string{
		"type": string(services.MsgType_DirectMsg),
		"to":   to,
		"data": data,
	})
	if err != nil {
		t.Errorf("write to %s: %v", to, err)
		return false
	}
	return true
}

// TestDirectMessage drives the whole private-chat path: two users connect over
// websockets, one sends a direct message, and we assert it is delivered to the
// recipient with the right sender identity and then persisted in history.
func TestDirectMessage(t *testing.T) {
	aliceToken := registerAndLogin(t, "alice")
	bobToken := registerAndLogin(t, "bob")
	alice, err := testAuthSrv.ValidateToken(aliceToken)
	if err != nil {
		t.Fatalf("validate alice token: %v", err)
	}

	aliceConn := dial(t, aliceToken)
	defer aliceConn.Close()
	bobConn := dial(t, bobToken)
	defer bobConn.Close()
	time.Sleep(200 * time.Millisecond) // let both Join the server

	sendDM(t, aliceConn, "bob", "hi bob")

	// bob receives it with alice as the sender
	bobConn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got := new(services.RespMsg)
	if err := bobConn.ReadJSON(got); err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.SenderName != "alice" || got.Data != "hi bob" || got.To != "bob" {
		t.Fatalf("unexpected message: %+v", got)
	}

	// and it is persisted in the conversation history
	time.Sleep(300 * time.Millisecond) // persist runs in its own goroutine
	msgs, err := testChatSrv.GetDirectMessages(alice.ID, "bob")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(msgs) != 1 || msgs[0].SenderName != "alice" || msgs[0].Data != "hi bob" {
		t.Fatalf("history mismatch: %+v", msgs)
	}
}

// TestConcurrentDirectMessages stresses the shared server state under -race:
// N users are arranged in a ring (user i only ever messages user i+1), every
// user fires M messages concurrently, and we assert each user receives exactly
// M messages. Running this under `go test -race` is what surfaces data races on
// the clients map, the dispatch channels and the persistence goroutines.
func TestConcurrentDirectMessages(t *testing.T) {
	const (
		n = 20 // users
		m = 50 // messages each user sends -> 1000 messages in flight
	)

	names := make([]string, n)
	conns := make([]*websocket.Conn, n)
	for i := range n {
		name := fmt.Sprintf("ring%02d", i)
		names[i] = name
		conns[i] = dial(t, registerAndLogin(t, name))
		defer conns[i].Close()
	}
	time.Sleep(300 * time.Millisecond) // let every client Join

	var received atomic.Int64
	var readers sync.WaitGroup
	// one reader goroutine per connection, counting delivered messages
	for i := range n {
		readers.Add(1)
		go func(conn *websocket.Conn) {
			defer readers.Done()
			for {
				var msg services.RespMsg
				if err := conn.ReadJSON(&msg); err != nil {
					return // connection closed at the end of the test
				}
				received.Add(1)
			}
		}(conns[i])
	}

	// every user fires M messages at its ring neighbour, all at once
	var senders sync.WaitGroup
	for i := range n {
		senders.Add(1)
		go func(i int) {
			defer senders.Done()
			to := names[(i+1)%n]
			for j := range m {
				if !sendDM(t, conns[i], to, fmt.Sprintf("msg-%d-%d", i, j)) {
					return
				}
			}
		}(i)
	}
	senders.Wait()

	// wait until all N*M messages are delivered (or fail on timeout)
	want := int64(n * m)
	deadline := time.Now().Add(30 * time.Second)
	for received.Load() < want {
		if time.Now().After(deadline) {
			t.Fatalf("timeout: delivered %d/%d messages", received.Load(), want)
		}
		time.Sleep(50 * time.Millisecond)
	}

	// closing the connections unblocks the reader goroutines
	for _, c := range conns {
		c.Close()
	}
	readers.Wait()

	if got := received.Load(); got != want {
		t.Fatalf("delivered %d messages, want %d", got, want)
	}
}

// TestChurnWhileMessaging hammers the join/leave path at the same time as the
// deliver path: two stable users message each other while extra tabs of those
// SAME users keep connecting and disconnecting. Delivery iterates the clients
// map (recipient tabs + sender's other tabs) while join/leave mutates it —
// exactly the interleaving where a data race would surface under -race if the
// AcceptLoop didn't own all of that state.
func TestChurnWhileMessaging(t *testing.T) {
	const (
		msgs     = 80 // messages each stable user sends
		churners = 6  // tabs that keep connecting/disconnecting
	)

	tokenA := registerAndLogin(t, "stableA")
	tokenB := registerAndLogin(t, "stableB")

	connA := dial(t, tokenA)
	defer connA.Close()
	connB := dial(t, tokenB)
	defer connB.Close()
	time.Sleep(200 * time.Millisecond) // let both Join the server

	// stable readers count only what the peer sent them
	var gotA, gotB atomic.Int64
	var readers sync.WaitGroup
	readCounted := func(conn *websocket.Conn, from string, counter *atomic.Int64) {
		defer readers.Done()
		for {
			msg := new(services.RespMsg)
			if err := conn.ReadJSON(msg); err != nil {
				return // connection closed at the end of the test
			}
			if msg.SenderName == from {
				counter.Add(1)
			}
		}
	}
	readers.Add(2)
	go readCounted(connA, "stableB", &gotA)
	go readCounted(connB, "stableA", &gotB)

	// churners: extra tabs of stableA/stableB joining and leaving nonstop
	stopChurn := make(chan struct{})
	var churn sync.WaitGroup
	for i := range churners {
		churn.Add(1)
		go func(i int) {
			defer churn.Done()
			token := tokenA
			if i%2 == 0 {
				token = tokenB
			}
			url := fmt.Sprintf("%s%s/ws?token=%s", host, config.WSPort, token)
			for {
				select {
				case <-stopChurn:
					return
				default:
				}
				conn, _, err := websocket.DefaultDialer.Dial(url, nil)
				if err != nil {
					time.Sleep(50 * time.Millisecond)
					continue
				}
				// stay online briefly, drain whatever gets delivered, then leave
				conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						break
					}
				}
				conn.Close()
			}
		}(i)
	}

	// both stable users fire messages while the churn is running; the tiny
	// sleep stretches the send window so many join/leave cycles overlap it
	var senders sync.WaitGroup
	send := func(conn *websocket.Conn, to, prefix string) {
		defer senders.Done()
		for j := range msgs {
			if !sendDM(t, conn, to, fmt.Sprintf("%s-%d", prefix, j)) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	senders.Add(2)
	go send(connA, "stableB", "a")
	go send(connB, "stableA", "b")
	senders.Wait()

	// every stable message must arrive despite the churn
	deadline := time.Now().Add(20 * time.Second)
	for gotA.Load() < msgs || gotB.Load() < msgs {
		if time.Now().After(deadline) {
			t.Fatalf("timeout: A got %d/%d, B got %d/%d", gotA.Load(), msgs, gotB.Load(), msgs)
		}
		time.Sleep(50 * time.Millisecond)
	}

	close(stopChurn)
	churn.Wait()
	connA.Close()
	connB.Close()
	readers.Wait()

	if gotA.Load() != msgs || gotB.Load() != msgs {
		t.Fatalf("A got %d, B got %d, want %d each", gotA.Load(), gotB.Load(), msgs)
	}
}
