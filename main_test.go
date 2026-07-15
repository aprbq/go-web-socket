package main

import (
	"fmt"
	"log"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var host = "ws://"

type TestConfig struct {
	clientCount int
	wg          *sync.WaitGroup
}

func DialServer(wg *sync.WaitGroup) {

	dialer := websocket.DefaultDialer

	conn, _, err := dialer.Dial(fmt.Sprintf("%s%s", host, WSPort), nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()
	defer wg.Done()

	fmt.Println("connected to the server ", conn.LocalAddr().String())
	time.Sleep(2 * time.Second)

}

func TestConnection(t *testing.T) {

	go createWSServer()
	time.Sleep(1 * time.Second)

	tc := TestConfig{
		clientCount: 50,
		wg:          new(sync.WaitGroup),
	}

	tc.wg.Add(tc.clientCount)

	for range tc.clientCount {
		go DialServer(tc.wg)
	}
	tc.wg.Wait()
	fmt.Println("exiting test")
	// go func() {
	// 	for {

	// 	}
	// }()
}
