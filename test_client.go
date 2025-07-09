package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/websocket"
	"powergrid/pkg/protocol"
)

type TestClient struct {
	conn     *websocket.Conn
	playerID string
	name     string
	color    string
	gameID   string
}

func NewTestClient(serverURL, playerID, name, color, gameID string) (*TestClient, error) {
	u := url.URL{Scheme: "ws", Host: serverURL, Path: "/game"}
	
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %v", err)
	}

	client := &TestClient{
		conn:     conn,
		playerID: playerID,
		name:     name,
		color:    color,
		gameID:   gameID,
	}

	return client, nil
}

func (c *TestClient) Close() {
	c.conn.Close()
}

func (c *TestClient) SendMessage(msgType protocol.MessageType, payload interface{}) error {
	message := protocol.NewMessage(msgType, payload)
	return c.conn.WriteJSON(message)
}

func (c *TestClient) ReadMessage() (*protocol.Message, error) {
	var message protocol.Message
	err := c.conn.ReadJSON(&message)
	return &message, err
}

func (c *TestClient) JoinGame() error {
	payload := protocol.JoinGamePayload{
		GameID:     c.gameID,
		PlayerName: c.name,
		Color:      c.color,
	}
	return c.SendMessage(protocol.MsgJoinGame, payload)
}

func (c *TestClient) StartListening() {
	go func() {
		for {
			message, err := c.ReadMessage()
			if err != nil {
				log.Printf("Client %s read error: %v", c.name, err)
				return
			}
			c.handleMessage(message)
		}
	}()
}

func (c *TestClient) handleMessage(message *protocol.Message) {
	switch message.Type {
	case protocol.MsgGameState:
		// Extract game ID from the game state
		if gameState, ok := message.Payload.(map[string]interface{}); ok {
			if gameID, exists := gameState["game_id"]; exists {
				if gameIDStr, ok := gameID.(string); ok {
					c.gameID = gameIDStr
					log.Printf("Client %s received game state for game %s", c.name, c.gameID)
					return
				}
			}
		}
		log.Printf("Client %s received game state", c.name)
	case protocol.MsgPhaseChange:
		log.Printf("Client %s: Phase changed", c.name)
	case protocol.MsgTurnChange:
		log.Printf("Client %s: Turn changed", c.name)
	case protocol.MsgError:
		log.Printf("Client %s received error: %v", c.name, message.Payload)
	default:
		log.Printf("Client %s received unknown message type: %s", c.name, message.Type)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: go run test_client.go <server_address>")
	}

	serverAddr := os.Args[1]
	gameID := "test-game-1"

	// Create test clients
	clients := make([]*TestClient, 3)
	playerData := []struct {
		id, name, color string
	}{
		{"player1", "Alice", "red"},
		{"player2", "Bob", "blue"},
		{"player3", "Charlie", "green"},
	}

	// Connect clients
	for i, data := range playerData {
		client, err := NewTestClient(serverAddr, data.id, data.name, data.color, gameID)
		if err != nil {
			log.Fatalf("Failed to create client %d: %v", i, err)
		}
		clients[i] = client
		client.StartListening()
	}

	// Clean up on exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		for _, client := range clients {
			client.Close()
		}
		os.Exit(0)
	}()

	// Wait a moment for connections to establish
	time.Sleep(100 * time.Millisecond)

	// Test sequence
	log.Println("Starting test sequence...")

	// 1. Create game
	log.Println("Creating game...")
	createPayload := protocol.CreateGamePayload{
		Name:       "Test Game",
		Map:        "usa",
		MaxPlayers: 6,
	}
	err := clients[0].SendMessage(protocol.MsgCreateGame, createPayload)
	if err != nil {
		log.Fatalf("Failed to create game: %v", err)
	}
	time.Sleep(500 * time.Millisecond) // Wait for game creation response

	// 2. Join players
	log.Println("Players joining...")
	for i, client := range clients {
		err := client.JoinGame()
		if err != nil {
			log.Fatalf("Player %d failed to join: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 3. Start game
	log.Println("Starting game...")
	err = clients[0].SendMessage(protocol.MsgStartGame, struct{}{})
	if err != nil {
		log.Fatalf("Failed to start game: %v", err)
	}
	time.Sleep(200 * time.Millisecond)

	// 4. Test auction phase - nominate a plant
	log.Println("Testing auction phase...")
	nominatePayload := protocol.BidPlantPayload{
		PlantID: 3, // Assuming plant 3 exists
		Bid:     3, // Minimum bid
	}
	err = clients[0].SendMessage(protocol.MsgBidPlant, nominatePayload)
	if err != nil {
		log.Fatalf("Failed to nominate plant: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// Other players pass
	passPayload := protocol.BidPlantPayload{
		PlantID: 3,
		Bid:     0, // 0 means pass
	}
	for i := 1; i < len(clients); i++ {
		err = clients[i].SendMessage(protocol.MsgBidPlant, passPayload)
		if err != nil {
			log.Printf("Player %d failed to pass: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	// 5. Test resource buying phase
	log.Println("Testing resource buying phase...")
	time.Sleep(200 * time.Millisecond)

	buyResourcesPayload := protocol.BuyResourcesPayload{
		Resources: map[string]int{"Coal": 2},
	}
	err = clients[0].SendMessage(protocol.MsgBuyResources, buyResourcesPayload)
	if err != nil {
		log.Printf("Failed to buy resources: %v", err)
	}
	time.Sleep(100 * time.Millisecond)

	// End turns for other phases
	for i := 0; i < len(clients); i++ {
		err = clients[i].SendMessage(protocol.MsgEndTurn, struct{}{})
		if err != nil {
			log.Printf("Player %d failed to end turn: %v", i, err)
		}
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("Test sequence completed. Press Ctrl+C to exit.")

	// Keep the program running to observe messages
	select {}
}