package game

import (
	"testing"
	"powergrid/pkg/protocol"
)

// TestGameCreation tests basic game creation
func TestGameCreation(t *testing.T) {
	game, err := NewGame("test-game", "Test Game", "usa")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Verify game ID
	if game.ID != "test-game" {
		t.Errorf("Expected game ID 'test-game', got %s", game.ID)
	}

	// Verify game name
	if game.Name != "Test Game" {
		t.Errorf("Expected game name 'Test Game', got %s", game.Name)
	}

	// Verify initial phase
	if game.CurrentPhase != protocol.PhasePlayerOrder {
		t.Errorf("Expected initial phase %s, got %s", protocol.PhasePlayerOrder, game.CurrentPhase)
	}

	// Verify initial status
	if game.Status != protocol.StatusLobby {
		t.Errorf("Expected initial status %s, got %s", protocol.StatusLobby, game.Status)
	}

	// Verify initial turn and round
	if game.CurrentTurn != 0 {
		t.Errorf("Expected initial turn 0, got %d", game.CurrentTurn)
	}

	if game.CurrentRound != 1 {
		t.Errorf("Expected initial round 1, got %d", game.CurrentRound)
	}

	// Verify players map is initialized
	if game.Players == nil {
		t.Error("Players map should be initialized")
	}

	if len(game.Players) != 0 {
		t.Errorf("Expected 0 initial players, got %d", len(game.Players))
	}
}

// TestGameStateAccess tests concurrent access to game state
func TestGameStateAccess(t *testing.T) {
	game, err := NewGame("concurrent-test", "Concurrent Test", "usa")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Test that we can safely access game state
	done := make(chan bool, 3)

	// Goroutine 1: Read game state
	go func() {
		for i := 0; i < 10; i++ {
			_ = game.ID
			_ = game.Name
			_ = game.CurrentPhase
		}
		done <- true
	}()

	// Goroutine 2: Read players
	go func() {
		for i := 0; i < 10; i++ {
			_ = len(game.Players)
			_ = game.TurnOrder
		}
		done <- true
	}()

	// Goroutine 3: Read status
	go func() {
		for i := 0; i < 10; i++ {
			_ = game.Status
			_ = game.CurrentTurn
			_ = game.CurrentRound
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}

	// If we get here without deadlock or panic, basic thread safety is working
	t.Log("Concurrent access test passed")
}

// TestGameID tests game ID validation
func TestGameID(t *testing.T) {
	// Test valid game creation
	game, err := NewGame("valid-id-123", "Valid Game", "usa")
	if err != nil {
		t.Errorf("Should allow valid game ID: %v", err)
	}

	if game.ID != "valid-id-123" {
		t.Errorf("Game ID not set correctly")
	}
}

// TestGameProtocolIntegration tests integration with protocol types
func TestGameProtocolIntegration(t *testing.T) {
	_, err := NewGame("protocol-test", "Protocol Test", "usa")
	if err != nil {
		t.Fatalf("Failed to create game: %v", err)
	}

	// Test that all protocol phases are valid
	validPhases := []protocol.GamePhase{
		protocol.PhasePlayerOrder,
		protocol.PhaseAuction,
		protocol.PhaseBuyResources,
		protocol.PhaseBuildCities,
		protocol.PhaseBureaucracy,
		protocol.PhaseGameEnd,
	}

	for _, phase := range validPhases {
		// Just verify the constants exist and are not empty
		if string(phase) == "" {
			t.Errorf("Phase %s should not be empty", phase)
		}
	}

	// Test that all protocol statuses are valid
	validStatuses := []protocol.GameStatus{
		protocol.StatusLobby,
		protocol.StatusPlaying,
		protocol.StatusFinished,
	}

	for _, status := range validStatuses {
		// Just verify the constants exist and are not empty
		if string(status) == "" {
			t.Errorf("Status %s should not be empty", status)
		}
	}
}