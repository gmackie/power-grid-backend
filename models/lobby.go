package models

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// LobbyStatus represents the current state of a lobby
type LobbyStatus string

const (
	LobbyStatusWaiting  LobbyStatus = "waiting"
	LobbyStatusStarting LobbyStatus = "starting"
	LobbyStatusInGame   LobbyStatus = "in_game"
	LobbyStatusEnded    LobbyStatus = "ended"
)

// Player represents a player in the game
type Player struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	IsHost   bool            `json:"is_host"`
	IsReady  bool            `json:"is_ready"`
	JoinedAt time.Time       `json:"joined_at"`
	Conn     *websocket.Conn `json:"-"` // Not serialized
}

// Message represents a message in the lobby
type Message struct {
	ID         string    `json:"id"`
	PlayerID   string    `json:"player_id"`
	PlayerName string    `json:"player_name"`
	Content    string    `json:"content"`
	CreatedAt  time.Time `json:"created_at"`
}

// Lobby represents a game lobby
type Lobby struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Status     LobbyStatus        `json:"status"`
	Players    map[string]*Player `json:"players"`
	Messages   []Message          `json:"messages"`
	MaxPlayers int                `json:"max_players"`
	MapID      string             `json:"map_id"`
	CreatedAt  time.Time          `json:"created_at"`
	IsAIOnly   bool               `json:"is_ai_only"`
	UpdatedAt  time.Time          `json:"updated_at"`
	Password   string             `json:"-"` // Not serialized to JSON
	mu         sync.Mutex         `json:"-"` // For thread safety
}

// NewLobby creates a new lobby
func NewLobby(name string, host *Player, maxPlayers int, password string, mapID string) *Lobby {
	// Create the lobby
	lobby := &Lobby{
		ID:         uuid.New().String(),
		Name:       name,
		Status:     LobbyStatusWaiting,
		Players:    make(map[string]*Player),
		Messages:   []Message{},
		MaxPlayers: maxPlayers,
		MapID:      mapID,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Password:   password,
	}

	// Set the host
	host.IsHost = true
	host.IsReady = true
	lobby.Players[host.ID] = host

	// Add welcome message
	lobby.AddSystemMessage("Lobby created. Waiting for players...")

	// Log the creation
	log.Printf("Initialized new lobby: %s (ID: %s) with host: %s", name, lobby.ID, host.Name)

	return lobby
}

// AddPlayer adds a player to the lobby
func (l *Lobby) AddPlayer(player *Player) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if the lobby is full
	if len(l.Players) >= l.MaxPlayers {
		return false
	}

	// Check if player is already in the lobby
	if _, exists := l.Players[player.ID]; exists {
		return false
	}

	l.Players[player.ID] = player
	l.UpdatedAt = time.Now()

	// Add system message
	l.AddSystemMessageLocked(player.Name + " joined the lobby")

	return true
}

// RemovePlayer removes a player from the lobby
func (l *Lobby) RemovePlayer(playerID string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	player, exists := l.Players[playerID]
	if !exists {
		return
	}

	// Add system message before removing
	l.AddSystemMessageLocked(player.Name + " left the lobby")

	// Delete the player
	delete(l.Players, playerID)
	l.UpdatedAt = time.Now()

	// If the host left, assign a new host
	if player.IsHost && len(l.Players) > 0 {
		// Find the player who joined earliest
		var newHost *Player
		var earliestJoin time.Time

		for _, p := range l.Players {
			if newHost == nil || p.JoinedAt.Before(earliestJoin) {
				newHost = p
				earliestJoin = p.JoinedAt
			}
		}

		if newHost != nil {
			newHost.IsHost = true
			l.AddSystemMessageLocked(newHost.Name + " is now the host")
		}
	}
}

// AddMessage adds a message to the lobby
func (l *Lobby) AddMessage(playerID string, content string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	player, exists := l.Players[playerID]
	if !exists {
		return
	}

	message := Message{
		ID:         uuid.New().String(),
		PlayerID:   playerID,
		PlayerName: player.Name,
		Content:    content,
		CreatedAt:  time.Now(),
	}

	l.Messages = append(l.Messages, message)
	l.UpdatedAt = time.Now()
}

// AddSystemMessage adds a system message to the lobby
func (l *Lobby) AddSystemMessage(content string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.AddSystemMessageLocked(content)
}

// AddSystemMessageLocked adds a system message to the lobby (with lock already acquired)
func (l *Lobby) AddSystemMessageLocked(content string) {
	message := Message{
		ID:         uuid.New().String(),
		PlayerID:   "system",
		PlayerName: "System",
		Content:    content,
		CreatedAt:  time.Now(),
	}

	l.Messages = append(l.Messages, message)
	l.UpdatedAt = time.Now()
}

// SetPlayerReady sets a player's ready status
func (l *Lobby) SetPlayerReady(playerID string, isReady bool) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	player, exists := l.Players[playerID]
	if !exists {
		return false
	}

	// Update ready status
	player.IsReady = isReady
	l.UpdatedAt = time.Now()

	// Add system message
	if isReady {
		l.AddSystemMessageLocked(player.Name + " is ready")
	} else {
		l.AddSystemMessageLocked(player.Name + " is not ready")
	}

	return true
}

// CanStartGame checks if the game can be started
func (l *Lobby) CanStartGame() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Need at least 2 players
	if len(l.Players) < 2 {
		return false
	}

	// All players must be ready
	for _, player := range l.Players {
		if !player.IsReady {
			return false
		}
	}

	return true
}

// StartGame transitions the lobby to game state
func (l *Lobby) StartGame() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Check if game can be started
	if len(l.Players) < 2 {
		return false
	}

	// Check if all players are ready
	for _, player := range l.Players {
		if !player.IsReady {
			return false
		}
	}

	// Update lobby status
	l.Status = LobbyStatusStarting
	l.UpdatedAt = time.Now()

	// Add system message
	l.AddSystemMessageLocked("Game is starting...")

	return true
}

// GetPlayerByConnection finds a player by their websocket connection
func (l *Lobby) GetPlayerByConnection(conn *websocket.Conn) *Player {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, player := range l.Players {
		if player.Conn == conn {
			return player
		}
	}

	return nil
}

// ToJSON returns a JSON-safe representation of the lobby (without connections)
func (l *Lobby) ToJSON() map[string]interface{} {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create a copy of the players without the websocket connection
	players := make(map[string]interface{})
	for id, player := range l.Players {
		players[id] = map[string]interface{}{
			"id":        player.ID,
			"name":      player.Name,
			"is_host":   player.IsHost,
			"is_ready":  player.IsReady,
			"joined_at": player.JoinedAt,
		}
	}

	// Return the JSON-safe lobby
	return map[string]interface{}{
		"id":          l.ID,
		"name":        l.Name,
		"status":      l.Status,
		"players":     players,
		"messages":    l.Messages,
		"max_players": l.MaxPlayers,
		"map_id":      l.MapID,
		"created_at":  l.CreatedAt,
		"updated_at":  l.UpdatedAt,
	}
}
