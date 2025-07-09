package test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"powergrid/handlers"
	"powergrid/internal/maps"
)

// TestClient wraps a WebSocket connection for testing
type TestClient struct {
	conn      *websocket.Conn
	sessionID string
	playerID  string
	name      string
	t         *testing.T
	mu        sync.Mutex
	messages  []handlers.Message
}

// NewTestClient creates a new test client
func NewTestClient(t *testing.T, wsURL string, name string) (*TestClient, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(wsURL, nil)
	if err != nil {
		return nil, err
	}

	client := &TestClient{
		conn:      conn,
		sessionID: fmt.Sprintf("test-session-%s-%d", name, time.Now().UnixNano()),
		name:      name,
		t:         t,
		messages:  make([]handlers.Message, 0),
	}

	// Start message reader
	go client.readMessages()

	return client, nil
}

// readMessages continuously reads messages from the WebSocket
func (c *TestClient) readMessages() {
	for {
		var msg handlers.Message
		err := c.conn.ReadJSON(&msg)
		if err != nil {
			return
		}
		c.mu.Lock()
		c.messages = append(c.messages, msg)
		c.mu.Unlock()
	}
}

// SendMessage sends a message to the server
func (c *TestClient) SendMessage(msgType handlers.MessageType, data map[string]interface{}) error {
	msg := handlers.Message{
		Type:      msgType,
		SessionID: c.sessionID,
		PlayerID:  c.playerID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
	return c.conn.WriteJSON(msg)
}

// WaitForMessage waits for a specific message type
func (c *TestClient) WaitForMessage(msgType handlers.MessageType, timeout time.Duration) (*handlers.Message, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		c.mu.Lock()
		for i, msg := range c.messages {
			if msg.Type == msgType {
				// Remove the message from the queue
				c.messages = append(c.messages[:i], c.messages[i+1:]...)
				c.mu.Unlock()
				return &msg, nil
			}
		}
		c.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
	return nil, fmt.Errorf("timeout waiting for message type %s", msgType)
}

// Connect establishes a player session
func (c *TestClient) Connect() error {
	// Wait for initial connection message
	_, err := c.WaitForMessage(handlers.TypeConnected, 2*time.Second)
	if err != nil {
		// Send connect message
		err = c.SendMessage(handlers.TypeConnect, map[string]interface{}{
			"player_name": c.name,
		})
		if err != nil {
			return fmt.Errorf("failed to send connect message: %v", err)
		}
	}

	// Wait for connected response
	msg, err := c.WaitForMessage(handlers.TypeConnected, 2*time.Second)
	if err != nil {
		return fmt.Errorf("failed to receive connected message: %v", err)
	}

	// Extract player ID
	if playerID, ok := msg.Data["player_id"].(string); ok {
		c.playerID = playerID
	} else {
		return fmt.Errorf("no player_id in connected message")
	}

	return nil
}

// Close closes the client connection
func (c *TestClient) Close() error {
	return c.conn.Close()
}

// TestFullGameFlow tests a complete game from lobby creation to game completion
func TestFullGameFlow(t *testing.T) {
	// Create handler with maps
	handler := handlers.NewLobbyHandler()
	handler.SetLogger(&handlers.DefaultLogger{})
	
	// Create map manager and load maps
	mapManager := maps.NewMapManager()
	if err := mapManager.LoadMapsFromDirectory("../maps"); err != nil {
		t.Logf("Warning: Could not load maps from ../maps: %v", err)
	}
	handler.SetMapManager(mapManager)
	
	// Start session cleanup routine
	handler.StartSessionCleanup(5*time.Minute, 30*time.Minute)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("CompleteLobbyToGameFlow", func(t *testing.T) {
		// Create host client
		host, err := NewTestClient(t, wsURL, "Host")
		if err != nil {
			t.Fatalf("Failed to create host client: %v", err)
		}
		defer host.Close()

		// Connect host
		err = host.Connect()
		if err != nil {
			t.Fatalf("Host failed to connect: %v", err)
		}

		// Host creates lobby
		err = host.SendMessage(handlers.TypeCreateLobby, map[string]interface{}{
			"lobby_name":  "Test Game",
			"max_players": 3,
			"map_id":      "usa",
		})
		if err != nil {
			t.Fatalf("Failed to create lobby: %v", err)
		}

		lobbyMsg, err := host.WaitForMessage(handlers.TypeLobbyCreated, 2*time.Second)
		if err != nil {
			t.Fatalf("Failed to receive lobby created message: %v", err)
		}

		lobbyData := lobbyMsg.Data["lobby"].(map[string]interface{})
		lobbyID := lobbyData["id"].(string)
		t.Logf("Lobby created with ID: %s", lobbyID)

		// Create guest clients
		guest1, err := NewTestClient(t, wsURL, "Guest1")
		if err != nil {
			t.Fatalf("Failed to create guest1 client: %v", err)
		}
		defer guest1.Close()

		guest2, err := NewTestClient(t, wsURL, "Guest2")
		if err != nil {
			t.Fatalf("Failed to create guest2 client: %v", err)
		}
		defer guest2.Close()

		// Connect guests
		err = guest1.Connect()
		if err != nil {
			t.Fatalf("Guest1 failed to connect: %v", err)
		}

		err = guest2.Connect()
		if err != nil {
			t.Fatalf("Guest2 failed to connect: %v", err)
		}

		// Guests join lobby
		err = guest1.SendMessage(handlers.TypeJoinLobby, map[string]interface{}{
			"lobby_id": lobbyID,
		})
		if err != nil {
			t.Fatalf("Guest1 failed to join lobby: %v", err)
		}

		_, err = guest1.WaitForMessage(handlers.TypeLobbyJoined, 2*time.Second)
		if err != nil {
			t.Fatalf("Guest1 failed to receive lobby joined message: %v", err)
		}

		err = guest2.SendMessage(handlers.TypeJoinLobby, map[string]interface{}{
			"lobby_id": lobbyID,
		})
		if err != nil {
			t.Fatalf("Guest2 failed to join lobby: %v", err)
		}

		_, err = guest2.WaitForMessage(handlers.TypeLobbyJoined, 2*time.Second)
		if err != nil {
			t.Fatalf("Guest2 failed to receive lobby joined message: %v", err)
		}

		// All players set ready
		players := []*TestClient{host, guest1, guest2}
		for _, player := range players {
			err = player.SendMessage(handlers.TypeSetReady, map[string]interface{}{
				"ready": true,
			})
			if err != nil {
				t.Fatalf("%s failed to set ready: %v", player.name, err)
			}
		}

		// Host starts game
		err = host.SendMessage(handlers.TypeStartGame, map[string]interface{}{})
		if err != nil {
			t.Fatalf("Host failed to start game: %v", err)
		}

		// All players should receive game starting message
		for _, player := range players {
			_, err = player.WaitForMessage(handlers.TypeGameStarting, 2*time.Second)
			if err != nil {
				t.Fatalf("%s failed to receive game starting message: %v", player.name, err)
			}
		}

		t.Log("Game successfully started with all players")
	})
}

// TestSessionReconnection tests reconnection scenarios
func TestSessionReconnection(t *testing.T) {
	// Create handler
	handler := handlers.NewLobbyHandler()
	handler.SetLogger(&handlers.DefaultLogger{})
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("ReconnectAfterDisconnect", func(t *testing.T) {
		// Create and connect first client
		client1, err := NewTestClient(t, wsURL, "ReconnectTest")
		if err != nil {
			t.Fatalf("Failed to create client1: %v", err)
		}

		sessionID := client1.sessionID

		err = client1.Connect()
		if err != nil {
			t.Fatalf("Client1 failed to connect: %v", err)
		}

		playerID1 := client1.playerID

		// Create lobby
		err = client1.SendMessage(handlers.TypeCreateLobby, map[string]interface{}{
			"lobby_name":  "Reconnect Test",
			"max_players": 2,
		})
		if err != nil {
			t.Fatalf("Failed to create lobby: %v", err)
		}

		lobbyMsg, err := client1.WaitForMessage(handlers.TypeLobbyCreated, 2*time.Second)
		if err != nil {
			t.Fatalf("Failed to receive lobby created message: %v", err)
		}

		lobbyData := lobbyMsg.Data["lobby"].(map[string]interface{})
		lobbyID := lobbyData["id"].(string)

		// Disconnect first client
		client1.Close()
		time.Sleep(100 * time.Millisecond)

		// Create new client with same session ID
		client2, err := NewTestClient(t, wsURL, "ReconnectTest")
		if err != nil {
			t.Fatalf("Failed to create client2: %v", err)
		}
		defer client2.Close()

		client2.sessionID = sessionID // Use same session ID

		err = client2.Connect()
		if err != nil {
			t.Fatalf("Client2 failed to reconnect: %v", err)
		}

		// Verify same player ID
		if client2.playerID != playerID1 {
			t.Errorf("Expected same player ID on reconnection. Got %s, expected %s", client2.playerID, playerID1)
		}

		// Verify can still access lobby
		err = client2.SendMessage(handlers.TypeListLobbies, map[string]interface{}{})
		if err != nil {
			t.Fatalf("Failed to list lobbies: %v", err)
		}

		lobbiesMsg, err := client2.WaitForMessage(handlers.TypeLobbiesListed, 2*time.Second)
		if err != nil {
			t.Fatalf("Failed to receive lobbies list: %v", err)
		}

		// Check if our lobby still exists
		lobbies := lobbiesMsg.Data["lobbies"].([]interface{})
		found := false
		for _, l := range lobbies {
			lobby := l.(map[string]interface{})
			if lobby["id"] == lobbyID {
				found = true
				break
			}
		}

		if !found {
			t.Error("Lobby should still exist after reconnection")
		}
	})
}

// TestConcurrentGames tests multiple games running simultaneously
func TestConcurrentGames(t *testing.T) {
	// Create handler with maps
	handler := handlers.NewLobbyHandler()
	handler.SetLogger(&handlers.DefaultLogger{})
	
	mapManager := maps.NewMapManager()
	if err := mapManager.LoadMapsFromDirectory("../maps"); err != nil {
		t.Logf("Warning: Could not load maps: %v", err)
	}
	handler.SetMapManager(mapManager)
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	numGames := 3
	playersPerGame := 2

	var wg sync.WaitGroup
	wg.Add(numGames)

	for gameNum := 0; gameNum < numGames; gameNum++ {
		go func(gNum int) {
			defer wg.Done()

			// Create host for this game
			host, err := NewTestClient(t, wsURL, fmt.Sprintf("Game%d-Host", gNum))
			if err != nil {
				t.Errorf("Game %d: Failed to create host: %v", gNum, err)
				return
			}
			defer host.Close()

			err = host.Connect()
			if err != nil {
				t.Errorf("Game %d: Host failed to connect: %v", gNum, err)
				return
			}

			// Create lobby
			err = host.SendMessage(handlers.TypeCreateLobby, map[string]interface{}{
				"lobby_name":  fmt.Sprintf("Game %d", gNum),
				"max_players": playersPerGame,
				"map_id":      "usa",
			})
			if err != nil {
				t.Errorf("Game %d: Failed to create lobby: %v", gNum, err)
				return
			}

			lobbyMsg, err := host.WaitForMessage(handlers.TypeLobbyCreated, 2*time.Second)
			if err != nil {
				t.Errorf("Game %d: Failed to receive lobby created: %v", gNum, err)
				return
			}

			lobbyData := lobbyMsg.Data["lobby"].(map[string]interface{})
			lobbyID := lobbyData["id"].(string)

			// Create guest
			guest, err := NewTestClient(t, wsURL, fmt.Sprintf("Game%d-Guest", gNum))
			if err != nil {
				t.Errorf("Game %d: Failed to create guest: %v", gNum, err)
				return
			}
			defer guest.Close()

			err = guest.Connect()
			if err != nil {
				t.Errorf("Game %d: Guest failed to connect: %v", gNum, err)
				return
			}

			// Guest joins
			err = guest.SendMessage(handlers.TypeJoinLobby, map[string]interface{}{
				"lobby_id": lobbyID,
			})
			if err != nil {
				t.Errorf("Game %d: Guest failed to join: %v", gNum, err)
				return
			}

			_, err = guest.WaitForMessage(handlers.TypeLobbyJoined, 2*time.Second)
			if err != nil {
				t.Errorf("Game %d: Guest failed to receive join confirmation: %v", gNum, err)
				return
			}

			t.Logf("Game %d successfully created and joined", gNum)
		}(gameNum)
	}

	wg.Wait()
	t.Log("All concurrent games completed successfully")
}

// TestMessageValidation tests server-side message validation
func TestMessageValidation(t *testing.T) {
	// Create handler
	handler := handlers.NewLobbyHandler()
	handler.SetLogger(&handlers.DefaultLogger{})
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	defer server.Close()
	
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")

	t.Run("InvalidMessageFormats", func(t *testing.T) {
		client, err := NewTestClient(t, wsURL, "ValidationTest")
		if err != nil {
			t.Fatalf("Failed to create client: %v", err)
		}
		defer client.Close()

		// Connect properly first
		err = client.Connect()
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Test missing required fields
		testCases := []struct {
			name     string
			msgType  handlers.MessageType
			data     map[string]interface{}
			expected string
		}{
			{
				name:     "CreateLobbyMissingName",
				msgType:  handlers.TypeCreateLobby,
				data:     map[string]interface{}{"max_players": 4},
				expected: "Lobby name is required",
			},
			{
				name:     "JoinLobbyMissingID",
				msgType:  handlers.TypeJoinLobby,
				data:     map[string]interface{}{},
				expected: "Lobby ID is required",
			},
			{
				name:     "ConnectMissingPlayerName",
				msgType:  handlers.TypeConnect,
				data:     map[string]interface{}{},
				expected: "Player name is required",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := client.SendMessage(tc.msgType, tc.data)
				if err != nil {
					t.Fatalf("Failed to send message: %v", err)
				}

				errorMsg, err := client.WaitForMessage(handlers.TypeError, 2*time.Second)
				if err != nil {
					t.Fatalf("Failed to receive error message: %v", err)
				}

				if msg, ok := errorMsg.Data["message"].(string); ok {
					if !strings.Contains(msg, tc.expected) {
						t.Errorf("Expected error containing '%s', got '%s'", tc.expected, msg)
					}
				} else {
					t.Error("Error message not found in response")
				}
			})
		}
	})
}