package handlers

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// TestLogger implements Logger interface for testing
type TestLogger struct {
	logs []string
}

func (l *TestLogger) Printf(format string, v ...interface{}) {
	l.logs = append(l.logs, format)
}

func (l *TestLogger) Println(v ...interface{}) {
	l.logs = append(l.logs, "log")
}

func (l *TestLogger) Fatal(v ...interface{}) {
	panic("fatal called in test")
}

func (l *TestLogger) Fatalf(format string, v ...interface{}) {
	panic("fatalf called in test")
}

// Helper function to create a test WebSocket server
func createTestServer(t *testing.T, handler *LobbyHandler) (*httptest.Server, string) {
	server := httptest.NewServer(http.HandlerFunc(handler.HandleWebSocket))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	return server, wsURL
}

// Helper function to create a WebSocket client
func createTestClient(t *testing.T, url string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	conn, _, err := dialer.Dial(url, nil)
	return conn, err
}

// Helper function to send a message and wait for response
func sendAndReceive(t *testing.T, conn *websocket.Conn, msg Message) Message {
	// Send message
	err := conn.WriteJSON(msg)
	if err != nil {
		t.Fatalf("Failed to send message: %v", err)
	}

	// Read response
	var response Message
	err = conn.ReadJSON(&response)
	if err != nil {
		t.Fatalf("Failed to read response: %v", err)
	}

	return response
}

// TestSessionManagement tests session creation and reconnection
func TestSessionManagement(t *testing.T) {
	handler := NewLobbyHandler()
	handler.SetLogger(&TestLogger{})
	server, wsURL := createTestServer(t, handler)
	defer server.Close()

	t.Run("NewSessionCreation", func(t *testing.T) {
		conn, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Wait for initial connection message
		var connMsg Message
		err = conn.ReadJSON(&connMsg)
		if err != nil {
			t.Fatalf("Failed to read connection message: %v", err)
		}

		// Send CONNECT message
		connectMsg := Message{
			Type:      TypeConnect,
			SessionID: "test-session-123",
			Data: map[string]interface{}{
				"player_name": "TestPlayer",
			},
		}

		response := sendAndReceive(t, conn, connectMsg)

		// Verify response
		if response.Type != TypeConnected {
			t.Errorf("Expected type %s, got %s", TypeConnected, response.Type)
		}

		if response.Data["player_name"] != "TestPlayer" {
			t.Errorf("Expected player_name TestPlayer, got %v", response.Data["player_name"])
		}

		if response.Data["player_id"] == "" {
			t.Error("Expected player_id to be set")
		}
	})

	t.Run("SessionReconnection", func(t *testing.T) {
		// First connection
		conn1, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}

		// Wait for initial connection message
		var connMsg Message
		conn1.ReadJSON(&connMsg)

		// Connect with session ID
		connectMsg := Message{
			Type:      TypeConnect,
			SessionID: "reconnect-session-456",
			Data: map[string]interface{}{
				"player_name": "ReconnectPlayer",
			},
		}

		response1 := sendAndReceive(t, conn1, connectMsg)
		playerID1 := response1.Data["player_id"].(string)
		conn1.Close()

		// Second connection with same session ID
		conn2, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to reconnect: %v", err)
		}
		defer conn2.Close()

		// Wait for initial connection message
		conn2.ReadJSON(&connMsg)

		// Reconnect with same session ID
		response2 := sendAndReceive(t, conn2, connectMsg)

		// Verify same player ID
		playerID2 := response2.Data["player_id"].(string)
		if playerID1 != playerID2 {
			t.Errorf("Expected same player ID on reconnection. Got %s and %s", playerID1, playerID2)
		}

		// Verify welcome back message
		if !strings.Contains(response2.Data["message"].(string), "Welcome back") {
			t.Errorf("Expected welcome back message, got: %v", response2.Data["message"])
		}
	})
}

// TestLobbyOperations tests lobby creation, joining, and leaving
func TestLobbyOperations(t *testing.T) {
	handler := NewLobbyHandler()
	handler.SetLogger(&TestLogger{})
	server, wsURL := createTestServer(t, handler)
	defer server.Close()

	t.Run("CreateLobbyWithoutSession", func(t *testing.T) {
		conn, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Wait for initial connection message
		var connMsg Message
		conn.ReadJSON(&connMsg)

		// Try to create lobby without connecting first
		createLobbyMsg := Message{
			Type:      TypeCreateLobby,
			SessionID: "no-session-789",
			Data: map[string]interface{}{
				"lobby_name":  "Test Lobby",
				"max_players": 4,
			},
		}

		response := sendAndReceive(t, conn, createLobbyMsg)

		// Should get an error
		if response.Type != TypeError {
			t.Errorf("Expected error type, got %s", response.Type)
		}

		if !strings.Contains(response.Data["message"].(string), "No active session") {
			t.Errorf("Expected session error message, got: %v", response.Data["message"])
		}
	})

	t.Run("CreateLobbyWithSession", func(t *testing.T) {
		conn, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect: %v", err)
		}
		defer conn.Close()

		// Wait for initial connection message
		var connMsg Message
		conn.ReadJSON(&connMsg)

		sessionID := "create-lobby-session-123"

		// First connect
		connectMsg := Message{
			Type:      TypeConnect,
			SessionID: sessionID,
			Data: map[string]interface{}{
				"player_name": "LobbyCreator",
			},
		}
		sendAndReceive(t, conn, connectMsg)

		// Create lobby
		createLobbyMsg := Message{
			Type:      TypeCreateLobby,
			SessionID: sessionID,
			Data: map[string]interface{}{
				"lobby_name":  "Test Lobby",
				"max_players": 4,
				"map_id":      "usa",
			},
		}

		response := sendAndReceive(t, conn, createLobbyMsg)

		// Check if we got an error and log it
		if response.Type == TypeError {
			t.Logf("Error creating lobby: %v", response.Data["message"])
			t.Errorf("Expected type %s, got %s with error: %v", TypeLobbyCreated, response.Type, response.Data["message"])
			return
		}

		// Verify lobby created
		if response.Type != TypeLobbyCreated {
			t.Errorf("Expected type %s, got %s", TypeLobbyCreated, response.Type)
			return
		}

		if response.Data["lobby"] == nil {
			t.Error("Lobby data is nil")
			return
		}

		lobbyData := response.Data["lobby"].(map[string]interface{})
		if lobbyData["name"] != "Test Lobby" {
			t.Errorf("Expected lobby name 'Test Lobby', got %v", lobbyData["name"])
		}
	})

	t.Run("JoinAndLeaveLobby", func(t *testing.T) {
		// Create host connection
		hostConn, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect host: %v", err)
		}
		defer hostConn.Close()

		var connMsg Message
		hostConn.ReadJSON(&connMsg)

		hostSessionID := "host-session-456"
		
		// Host connects
		connectMsg := Message{
			Type:      TypeConnect,
			SessionID: hostSessionID,
			Data: map[string]interface{}{
				"player_name": "Host",
			},
		}
		sendAndReceive(t, hostConn, connectMsg)

		// Host creates lobby
		createLobbyMsg := Message{
			Type:      TypeCreateLobby,
			SessionID: hostSessionID,
			Data: map[string]interface{}{
				"lobby_name":  "Join Test Lobby",
				"max_players": 4,
			},
		}
		lobbyResponse := sendAndReceive(t, hostConn, createLobbyMsg)
		lobbyData := lobbyResponse.Data["lobby"].(map[string]interface{})
		lobbyID := lobbyData["id"].(string)

		// Create guest connection
		guestConn, err := createTestClient(t, wsURL)
		if err != nil {
			t.Fatalf("Failed to connect guest: %v", err)
		}
		defer guestConn.Close()

		guestConn.ReadJSON(&connMsg)

		guestSessionID := "guest-session-789"

		// Guest connects
		guestConnectMsg := Message{
			Type:      TypeConnect,
			SessionID: guestSessionID,
			Data: map[string]interface{}{
				"player_name": "Guest",
			},
		}
		sendAndReceive(t, guestConn, guestConnectMsg)

		// Guest joins lobby
		joinMsg := Message{
			Type:      TypeJoinLobby,
			SessionID: guestSessionID,
			Data: map[string]interface{}{
				"lobby_id": lobbyID,
			},
		}
		joinResponse := sendAndReceive(t, guestConn, joinMsg)

		if joinResponse.Type != TypeLobbyJoined {
			t.Errorf("Expected type %s, got %s", TypeLobbyJoined, joinResponse.Type)
		}

		// Guest leaves lobby
		leaveMsg := Message{
			Type:      TypeLeaveLobby,
			SessionID: guestSessionID,
		}
		leaveResponse := sendAndReceive(t, guestConn, leaveMsg)

		if leaveResponse.Type != TypeLobbyLeft {
			t.Errorf("Expected type %s, got %s", TypeLobbyLeft, leaveResponse.Type)
		}
	})
}

// TestSessionCleanup tests the session cleanup mechanism
func TestSessionCleanup(t *testing.T) {
	handler := NewLobbyHandler()
	handler.SetLogger(&TestLogger{})

	// Create a session manually
	sessionID := "cleanup-test-session"
	session := &PlayerSession{
		PlayerID:     "test-player-123",
		PlayerName:   "CleanupTest",
		Connection:   nil,
		CreatedAt:    time.Now().Add(-2 * time.Hour),
		LastActivity: time.Now().Add(-2 * time.Hour),
	}

	handler.sessions[sessionID] = session

	// Run cleanup with 1 hour timeout
	handler.cleanupInactiveSessions(1 * time.Hour)

	// Verify session was removed
	handler.mu.Lock()
	_, exists := handler.sessions[sessionID]
	handler.mu.Unlock()

	if exists {
		t.Error("Expected inactive session to be cleaned up")
	}

	// Create an active session
	activeSessionID := "active-session"
	activeSession := &PlayerSession{
		PlayerID:     "active-player-456",
		PlayerName:   "ActivePlayer",
		Connection:   nil,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	handler.sessions[activeSessionID] = activeSession

	// Run cleanup
	handler.cleanupInactiveSessions(1 * time.Hour)

	// Verify active session remains
	handler.mu.Lock()
	_, exists = handler.sessions[activeSessionID]
	handler.mu.Unlock()

	if !exists {
		t.Error("Expected active session to remain")
	}
}

// TestConcurrentConnections tests handling multiple concurrent connections
func TestConcurrentConnections(t *testing.T) {
	handler := NewLobbyHandler()
	handler.SetLogger(&TestLogger{})
	server, wsURL := createTestServer(t, handler)
	defer server.Close()

	numClients := 10
	done := make(chan bool, numClients)

	// Create multiple concurrent connections
	for i := 0; i < numClients; i++ {
		go func(clientID int) {
			conn, err := createTestClient(t, wsURL)
			if err != nil {
				t.Errorf("Client %d failed to connect: %v", clientID, err)
				done <- false
				return
			}
			defer conn.Close()

			// Wait for initial connection message
			var connMsg Message
			conn.ReadJSON(&connMsg)

			// Connect
			connectMsg := Message{
				Type:      TypeConnect,
				SessionID: fmt.Sprintf("concurrent-session-%d", clientID),
				Data: map[string]interface{}{
					"player_name": fmt.Sprintf("Player%d", clientID),
				},
			}

			response := sendAndReceive(t, conn, connectMsg)
			
			if response.Type != TypeConnected {
				t.Errorf("Client %d: Expected type %s, got %s", clientID, TypeConnected, response.Type)
				done <- false
				return
			}

			done <- true
		}(i)
	}

	// Wait for all clients to complete
	successCount := 0
	for i := 0; i < numClients; i++ {
		if <-done {
			successCount++
		}
	}

	if successCount != numClients {
		t.Errorf("Expected %d successful connections, got %d", numClients, successCount)
	}

	// Verify all sessions exist
	sessionInfo := handler.GetSessionInfo()
	totalSessions := sessionInfo["total_sessions"].(int)
	if totalSessions < numClients {
		t.Errorf("Expected at least %d sessions, got %d", numClients, totalSessions)
	}
}