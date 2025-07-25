package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
)

type WebSocketMessage struct {
	Type      string      `json:"type"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp int64       `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run test_lobby_data.go <server_port>")
	}

	port := os.Args[1]
	serverURL := fmt.Sprintf("localhost:%s", port)

	// Connect to WebSocket
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/ws"}
	fmt.Printf("Connecting to %s\n", u.String())

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Failed to connect:", err)
	}
	defer conn.Close()

	// Handle Ctrl+C
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	done := make(chan struct{})

	// Message reader
	go func() {
		defer close(done)
		for {
			var message WebSocketMessage
			err := conn.ReadJSON(&message)
			if err != nil {
				log.Println("Read error:", err)
				return
			}
			
			fmt.Printf("\n=== RECEIVED MESSAGE ===\n")
			fmt.Printf("Type: %s\n", message.Type)
			if message.Data != nil {
				dataBytes, _ := json.MarshalIndent(message.Data, "", "  ")
				fmt.Printf("Data: %s\n", string(dataBytes))
			}
			fmt.Printf("========================\n\n")

			// If this is lobby data, log the structure
			if message.Type == "LOBBIES_LISTED" {
				if data, ok := message.Data.(map[string]interface{}); ok {
					if lobbies, exists := data["lobbies"]; exists {
						fmt.Printf("\n=== LOBBY DATA STRUCTURE ===\n")
						lobbiesBytes, _ := json.MarshalIndent(lobbies, "", "  ")
						fmt.Printf("Lobbies: %s\n", string(lobbiesBytes))
						fmt.Printf("============================\n\n")
					}
				}
			}
		}
	}()

	// Send messages
	sendMessage := func(msgType string, data interface{}) {
		message := WebSocketMessage{
			Type:      msgType,
			Timestamp: time.Now().Unix(),
			Data:      data,
		}
		err := conn.WriteJSON(message)
		if err != nil {
			log.Printf("Failed to send %s: %v", msgType, err)
		} else {
			fmt.Printf("Sent %s message\n", msgType)
		}
	}

	// 1. Connect as a player
	fmt.Println("Registering player...")
	sendMessage("CONNECT", map[string]string{
		"player_name": "TestPlayer",
	})
	time.Sleep(1 * time.Second)

	// 2. Create a lobby
	fmt.Println("Creating lobby...")
	sendMessage("CREATE_LOBBY", map[string]interface{}{
		"lobby_name":  "Test Lobby Data Capture",
		"max_players": 6,
		"map_id":      "usa",
	})
	time.Sleep(1 * time.Second)

	// 3. List lobbies to see the structure
	fmt.Println("Listing lobbies...")
	sendMessage("LIST_LOBBIES", map[string]interface{}{})
	time.Sleep(1 * time.Second)

	// Wait for interrupt or done
	select {
	case <-done:
		fmt.Println("Connection closed")
	case <-interrupt:
		fmt.Println("Interrupt signal received")
		
		// Send close message
		err := conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		if err != nil {
			log.Println("Write close error:", err)
			return
		}
		
		select {
		case <-done:
		case <-time.After(time.Second):
		}
	}
}