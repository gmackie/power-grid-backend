package testutil

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"powergrid/handlers"
)

// MessageMatcher is a function that checks if a message matches certain criteria
type MessageMatcher func(msg handlers.Message) bool

// WaitForMessageFunc waits for a message matching the given criteria
func WaitForMessageFunc(conn *websocket.Conn, matcher MessageMatcher, timeout time.Duration) (*handlers.Message, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		// Set read deadline
		conn.SetReadDeadline(deadline)
		
		var msg handlers.Message
		err := conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				return nil, fmt.Errorf("connection closed unexpectedly: %v", err)
			}
			continue
		}
		
		if matcher(msg) {
			return &msg, nil
		}
	}
	
	return nil, fmt.Errorf("timeout waiting for message")
}

// WaitForMessageType waits for a specific message type
func WaitForMessageType(conn *websocket.Conn, msgType handlers.MessageType, timeout time.Duration) (*handlers.Message, error) {
	return WaitForMessageFunc(conn, func(msg handlers.Message) bool {
		return msg.Type == msgType
	}, timeout)
}

// AssertMessageType checks if a message has the expected type
func AssertMessageType(t *testing.T, msg *handlers.Message, expected handlers.MessageType) {
	t.Helper()
	if msg.Type != expected {
		t.Errorf("Expected message type %s, got %s", expected, msg.Type)
	}
}

// AssertNoError fails the test if err is not nil
func AssertNoError(t *testing.T, err error, msg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: %v", msg, err)
	}
}

// AssertError fails the test if err is nil
func AssertError(t *testing.T, err error, msg string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected error but got nil", msg)
	}
}

// CreateTestMessage creates a test message
func CreateTestMessage(msgType handlers.MessageType, sessionID string, data map[string]interface{}) handlers.Message {
	return handlers.Message{
		Type:      msgType,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}
}

// SendAndWait sends a message and waits for a response of a specific type
func SendAndWait(t *testing.T, conn *websocket.Conn, msg handlers.Message, expectedType handlers.MessageType, timeout time.Duration) *handlers.Message {
	t.Helper()
	
	err := conn.WriteJSON(msg)
	AssertNoError(t, err, "Failed to send message")
	
	response, err := WaitForMessageType(conn, expectedType, timeout)
	AssertNoError(t, err, fmt.Sprintf("Failed to receive %s message", expectedType))
	
	return response
}

// ExtractLobbyID extracts lobby ID from a LOBBY_CREATED message
func ExtractLobbyID(t *testing.T, msg *handlers.Message) string {
	t.Helper()
	
	lobbyData, ok := msg.Data["lobby"].(map[string]interface{})
	if !ok {
		t.Fatal("No lobby data in message")
	}
	
	lobbyID, ok := lobbyData["id"].(string)
	if !ok {
		t.Fatal("No lobby ID in lobby data")
	}
	
	return lobbyID
}

// ExtractPlayerID extracts player ID from a CONNECTED message
func ExtractPlayerID(t *testing.T, msg *handlers.Message) string {
	t.Helper()
	
	playerID, ok := msg.Data["player_id"].(string)
	if !ok {
		t.Fatal("No player_id in message")
	}
	
	return playerID
}

// GameStateValidator validates game state consistency
type GameStateValidator struct {
	t *testing.T
}

// NewGameStateValidator creates a new game state validator
func NewGameStateValidator(t *testing.T) *GameStateValidator {
	return &GameStateValidator{t: t}
}

// ValidatePhaseTransition validates that phase transitions are legal
func (v *GameStateValidator) ValidatePhaseTransition(oldPhase, newPhase string) {
	v.t.Helper()
	
	validTransitions := map[string][]string{
		"PLAYER_ORDER":  {"AUCTION"},
		"AUCTION":       {"RESOURCES"},
		"RESOURCES":     {"BUILDING"},
		"BUILDING":      {"BUREAUCRACY"},
		"BUREAUCRACY":   {"AUCTION", "GAME_END"},
	}
	
	validNext, exists := validTransitions[oldPhase]
	if !exists {
		v.t.Errorf("Unknown phase: %s", oldPhase)
		return
	}
	
	valid := false
	for _, next := range validNext {
		if next == newPhase {
			valid = true
			break
		}
	}
	
	if !valid {
		v.t.Errorf("Invalid phase transition from %s to %s", oldPhase, newPhase)
	}
}

// ValidatePlayerMoney validates that player money never goes negative
func (v *GameStateValidator) ValidatePlayerMoney(players []interface{}) {
	v.t.Helper()
	
	for _, p := range players {
		player, ok := p.(map[string]interface{})
		if !ok {
			continue
		}
		
		money, ok := player["money"].(float64)
		if !ok {
			continue
		}
		
		if money < 0 {
			v.t.Errorf("Player %s has negative money: %.2f", player["id"], money)
		}
	}
}

// TestGameRecorder records all game messages for analysis
type TestGameRecorder struct {
	Messages []handlers.Message
	mu       sync.RWMutex
}

// NewTestGameRecorder creates a new game recorder
func NewTestGameRecorder() *TestGameRecorder {
	return &TestGameRecorder{
		Messages: make([]handlers.Message, 0),
	}
}

// Record adds a message to the recording
func (r *TestGameRecorder) Record(msg handlers.Message) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Messages = append(r.Messages, msg)
}

// GetMessages returns all recorded messages
func (r *TestGameRecorder) GetMessages() []handlers.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	messages := make([]handlers.Message, len(r.Messages))
	copy(messages, r.Messages)
	return messages
}

// FindMessages returns all messages matching a type
func (r *TestGameRecorder) FindMessages(msgType handlers.MessageType) []handlers.Message {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	var matches []handlers.Message
	for _, msg := range r.Messages {
		if msg.Type == msgType {
			matches = append(matches, msg)
		}
	}
	return matches
}

// PrintGameLog prints a formatted game log for debugging
func (r *TestGameRecorder) PrintGameLog(t *testing.T) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	t.Log("=== Game Message Log ===")
	for i, msg := range r.Messages {
		data, _ := json.MarshalIndent(msg.Data, "  ", "  ")
		t.Logf("%d. [%s] Type: %s\n  Data: %s", i+1, time.Unix(msg.Timestamp, 0).Format("15:04:05"), msg.Type, string(data))
	}
	t.Log("======================")
}