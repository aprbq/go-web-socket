package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aprbq/go-web-socket/config"
	"github.com/aprbq/go-web-socket/handlers"
	"github.com/aprbq/go-web-socket/repositories"
	"github.com/aprbq/go-web-socket/services"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// cors lets chat.html (opened from file://) call the REST endpoints
func cors(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		h(w, r)
	}
}

func preflight(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.WriteHeader(http.StatusNoContent)
}

// handleREST registers a "METHOD /path" route with CORS plus its OPTIONS preflight
func handleREST(pattern string, h http.HandlerFunc) {
	_, path, _ := strings.Cut(pattern, " ")
	http.HandleFunc("OPTIONS "+path, preflight)
	http.HandleFunc(pattern, cors(h))
}

func createWSServer(
	chatSrv services.ChatService,
	wsHandler handlers.WSHandler,
	authHandler handlers.AuthHandler,
	userHandler handlers.UserHandler,
	directHandler handlers.DirectHandler,
) {
	go chatSrv.AcceptLoop()

	handleREST("POST /register", authHandler.Register)
	handleREST("POST /login", authHandler.Login)
	handleREST("GET /users", userHandler.SearchUsers)
	handleREST("GET /dm/history", directHandler.GetHistory)
	handleREST("GET /dm/conversations", directHandler.GetConversations)
	http.HandleFunc("/ws", wsHandler.HandleWS)

	fmt.Printf("starting server on port: %s\n", config.WSPort)
	log.Fatal(http.ListenAndServe(config.WSPort, nil))
}

func main() {
	config.Load()

	db, err := gorm.Open(postgres.Open(config.DBDsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("unable to connect to the database: %v", err)
	}

	// composition root — the only place that knows the concrete implementations
	userRepo := repositories.NewUserRepositoryDB(db)
	msgRepo := repositories.NewMessageRepositoryDB(db)

	authSrv := services.NewAuthService(userRepo, config.JWTSecret)
	chatSrv := services.NewChatService(msgRepo, userRepo)

	authHandler := handlers.NewAuthHandler(authSrv)
	userHandler := handlers.NewUserHandler(authSrv)
	directHandler := handlers.NewDirectHandler(chatSrv, authSrv)
	wsHandler := handlers.NewWSHandler(chatSrv, authSrv)

	createWSServer(chatSrv, wsHandler, authHandler, userHandler, directHandler)
}
