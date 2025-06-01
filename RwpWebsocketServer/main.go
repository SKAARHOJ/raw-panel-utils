package main

import (
	"flag"
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
	log "github.com/s00500/env_logger"
)

const (
	authClientId     = "admin"
	authClientSecret = "password"
)

var authRequired = authClientId != "" || authClientSecret != ""

var debugWS bool // global flag for WebSocket debug mode

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins — adjust for security in production!
	},
}

func handleWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade failed:", err)
		return
	}
	defer conn.Close()

	if authRequired {
		if !authenticateWebSocket(conn) {
			return
		}
	}

	sendReadyMessage(conn)

	ws := NewWSConnection(conn)
	defer ws.Shutdown()

	<-ws.ctx.Done()
	log.Println("WebSocket connection closed, shutting down...")
}

func main() {
	flag.BoolVar(&debugWS, "debugWS", false, "Print all WebSocket messages as JSON for debugging")
	flag.Parse()

	http.HandleFunc("/ws", handleWS)

	port := 8080
	log.Printf("WebSocket server listening on :%d/ws", port)
	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal("Server failed:", err)
	}
}
