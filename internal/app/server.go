package app

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/internal/game"
	"powergrid/internal/network"
	"powergrid/pkg/protocol"
)

// Server represents the Power Grid game server
type Server struct {
	httpServer *http.Server
	games      map[string]*game.Game
	gamesMutex sync.RWMutex
	upgrader   websocket.Upgrader
}

// NewServer creates a new server instance
func NewServer(port string) *Server {
	s := &Server{
		games: make(map[string]*game.Game),
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for development
			},
		},
	}

	// Create HTTP server
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	s.httpServer = &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

// Start starts the server
func (s *Server) Start() error {
	return s.httpServer.ListenAndServe()
}

// Stop stops the server
func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.httpServer.Shutdown(ctx)
}

// handleHome handles the home page
func (s *Server) handleHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"name":    "Power Grid Game Server",
		"version": "0.1.0",
		"status":  "running",
	})
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}

// handleWebSocket handles WebSocket connection requests
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	// Create a new session for this connection
	session := network.NewSession(conn)

	log.Printf("New WebSocket connection established: %s", session.ID)

	// The session will handle reading/writing to the connection
	// Our SessionManager will process the messages
}

// CreateGame creates a new game
func (s *Server) CreateGame(name, mapName string, maxPlayers int) (*game.Game, error) {
	s.gamesMutex.Lock()
	defer s.gamesMutex.Unlock()

	gameID := uuid.New().String()
	newGame, err := game.NewGame(gameID, name, mapName)
	if err != nil {
		return nil, err
	}

	s.games[gameID] = newGame
	return newGame, nil
}

// GetGame gets a game by ID
func (s *Server) GetGame(gameID string) (*game.Game, bool) {
	s.gamesMutex.RLock()
	defer s.gamesMutex.RUnlock()

	game, exists := s.games[gameID]
	return game, exists
}

// DeleteGame deletes a game
func (s *Server) DeleteGame(gameID string) {
	s.gamesMutex.Lock()
	defer s.gamesMutex.Unlock()

	delete(s.games, gameID)
}

// ListGames returns a list of all games
func (s *Server) ListGames() []protocol.GameInfo {
	s.gamesMutex.RLock()
	defer s.gamesMutex.RUnlock()

	var gamesList []protocol.GameInfo
	for id, g := range s.games {
		// Only include games that are in lobby or playing
		if g.Status == protocol.StatusLobby || g.Status == protocol.StatusPlaying {
			gameInfo := protocol.GameInfo{
				ID:         id,
				Name:       g.Name,
				Status:     g.Status,
				Map:        g.Map.Name,
				Players:    len(g.Players),
				MaxPlayers: 6, // Fixed for now, could be configurable
				CreatedAt:  g.CreatedAt.Unix(),
			}
			gamesList = append(gamesList, gameInfo)
		}
	}
	return gamesList
}

// InitMessageHandlers initializes message handlers for the session manager
func (s *Server) InitMessageHandlers() {
	// We'll implement this in the next phase as we add more functionality
}
