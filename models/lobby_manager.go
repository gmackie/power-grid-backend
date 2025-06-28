package models

import (
	"fmt"
	"log"
	"sync"
)

// LobbyManager manages all active lobbies in the system
type LobbyManager struct {
	lobbies map[string]*Lobby
	mu      sync.RWMutex
}

// NewLobbyManager creates a new lobby manager
func NewLobbyManager() *LobbyManager {
	return &LobbyManager{
		lobbies: make(map[string]*Lobby),
	}
}

// CreateLobby creates a new lobby with the given parameters
func (lm *LobbyManager) CreateLobby(name string, host *Player, maxPlayers int, password string) *Lobby {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	// Create a new lobby
	lobby := NewLobby(name, host, maxPlayers, password)

	// Store the lobby
	lm.lobbies[lobby.ID] = lobby

	// Log the creation
	log.Printf("Created new lobby: %s (ID: %s) with host: %s", name, lobby.ID, host.Name)

	return lobby
}

// GetLobby retrieves a lobby by ID
func (lm *LobbyManager) GetLobby(lobbyID string) (*Lobby, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lobby, exists := lm.lobbies[lobbyID]
	if !exists {
		return nil, fmt.Errorf("lobby not found: %s", lobbyID)
	}

	return lobby, nil
}

// DeleteLobby removes a lobby
func (lm *LobbyManager) DeleteLobby(lobbyID string) error {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	if _, exists := lm.lobbies[lobbyID]; !exists {
		return fmt.Errorf("lobby not found: %s", lobbyID)
	}

	delete(lm.lobbies, lobbyID)
	return nil
}

// ListLobbies returns a list of all lobbies
func (lm *LobbyManager) ListLobbies() []*Lobby {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lobbies := make([]*Lobby, 0, len(lm.lobbies))
	for _, lobby := range lm.lobbies {
		lobbies = append(lobbies, lobby)
	}

	return lobbies
}

// ListLobbiesJSON returns a JSON-friendly representation of all lobbies
func (lm *LobbyManager) ListLobbiesJSON() []map[string]interface{} {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lobbiesJSON := make([]map[string]interface{}, 0, len(lm.lobbies))
	for _, lobby := range lm.lobbies {
		// Create a simplified version of the lobby
		lobbyJSON := map[string]interface{}{
			"id":           lobby.ID,
			"name":         lobby.Name,
			"status":       lobby.Status,
			"player_count": len(lobby.Players),
			"max_players":  lobby.MaxPlayers,
			"has_password": lobby.Password != "",
			"created_at":   lobby.CreatedAt,
		}
		lobbiesJSON = append(lobbiesJSON, lobbyJSON)
	}

	return lobbiesJSON
}

// CleanupLobby removes empty lobbies or updates lobby state when needed
func (lm *LobbyManager) CleanupLobby(lobbyID string) {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lobby, exists := lm.lobbies[lobbyID]
	if !exists {
		return
	}

	// Check if lobby is empty
	if len(lobby.Players) == 0 {
		delete(lm.lobbies, lobbyID)
	}
}

// GetPlayerLobby finds the lobby containing a specific player
func (lm *LobbyManager) GetPlayerLobby(playerID string) *Lobby {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	for _, lobby := range lm.lobbies {
		lobby.mu.Lock()
		_, exists := lobby.Players[playerID]
		lobby.mu.Unlock()

		if exists {
			return lobby
		}
	}

	return nil
}
