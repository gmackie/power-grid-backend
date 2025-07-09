package handlers

import (
	"log"
	"net/http"

	"github.com/gorilla/websocket"
	"powergrid/internal/network"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin for testing
		return true
	},
}

// HandleGameWebSocket handles the protocol-based WebSocket connections
func HandleGameWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create a new session
	session := network.NewSession(conn)
	log.Printf("New game WebSocket connection established: %s", session.ID)

	// The session will handle everything from here through its readPump/writePump
}