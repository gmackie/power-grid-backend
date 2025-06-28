package models

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// GameStatus represents the current state of a game
type GameStatus string

const (
	GameStatusSetup  GameStatus = "setup"
	GameStatusActive GameStatus = "active"
	GameStatusEnded  GameStatus = "ended"
)

// Game represents an active game session
type Game struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Status    GameStatus `json:"status"`
	Players   []*Player  `json:"players"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	mu        sync.Mutex `json:"-"` // For thread safety
}

// NewGame creates a new game from a lobby
func NewGame(lobby *Lobby) *Game {
	game := &Game{
		ID:        uuid.New().String(),
		Name:      lobby.Name,
		Status:    GameStatusSetup,
		Players:   make([]*Player, 0),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Add players from the lobby
	for _, player := range lobby.Players {
		// Clone player to avoid shared references
		gamePlayers := &Player{
			ID:       player.ID,
			Name:     player.Name,
			IsHost:   player.IsHost,
			IsReady:  player.IsReady,
			JoinedAt: player.JoinedAt,
			Conn:     player.Conn,
		}
		game.Players = append(game.Players, gamePlayers)
	}

	return game
}

// ToJSON returns a JSON-safe representation of the game
func (g *Game) ToJSON() map[string]interface{} {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Create a copy of the players without the websocket connection
	players := make([]map[string]interface{}, 0, len(g.Players))
	for _, player := range g.Players {
		players = append(players, map[string]interface{}{
			"id":        player.ID,
			"name":      player.Name,
			"is_host":   player.IsHost,
			"is_ready":  player.IsReady,
			"joined_at": player.JoinedAt,
		})
	}

	// Return the JSON-safe game
	return map[string]interface{}{
		"id":         g.ID,
		"name":       g.Name,
		"status":     g.Status,
		"players":    players,
		"created_at": g.CreatedAt,
		"updated_at": g.UpdatedAt,
	}
}
