package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/aprbq/go-web-socket/config"
	"github.com/aprbq/go-web-socket/handlers"
	"github.com/aprbq/go-web-socket/services"
)

func createWSServer(chatSrv services.ChatService, wsHandler handlers.WSHandler) {
	go chatSrv.AcceptLoop()
	http.HandleFunc("/", wsHandler.HandleWS)

	fmt.Printf("starting server on port: %s\n", config.WSPort)
	log.Fatal(http.ListenAndServe(config.WSPort, nil))
}

// TODO
// [x] HTTP server
// [x] Upgrade it to WS once client connects
// [x] Add WS client
// [x] Add newly connected ws to server
// [x] Remove client on disconnect
// [x] Send broadcast msg -> no race conditions
// -----
// [x] join room
// [x] leave room
// [x] Send room msg -> no race conditions
// -----
// [] test performance -> channels vs locks
func main() {
	chatSrv := services.NewChatService()
	wsHandler := handlers.NewWSHandler(chatSrv)

	createWSServer(chatSrv, wsHandler)
}
