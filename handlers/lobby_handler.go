package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/models"
)

// Logger interface for custom logging
type Logger interface {
	Printf(format string, v ...interface{})
	Println(v ...interface{})
	Fatal(v ...interface{})
	Fatalf(format string, v ...interface{})
}

// DefaultLogger implements Logger using standard log package
type DefaultLogger struct{}

func (l *DefaultLogger) Printf(format string, v ...interface{}) {
	log.Printf(format, v...)
}

func (l *DefaultLogger) Println(v ...interface{}) {
	log.Println(v...)
}

func (l *DefaultLogger) Fatal(v ...interface{}) {
	log.Fatal(v...)
}

func (l *DefaultLogger) Fatalf(format string, v ...interface{}) {
	log.Fatalf(format, v...)
}

// MessageType represents the type of message sent between client and server
type MessageType string

const (
	// Client-to-server message types
	TypeConnect     MessageType = "CONNECT"
	TypeDisconnect  MessageType = "DISCONNECT"
	TypeCreateLobby MessageType = "CREATE_LOBBY"
	TypeJoinLobby   MessageType = "JOIN_LOBBY"
	TypeLeaveLobby  MessageType = "LEAVE_LOBBY"
	TypeChatMessage MessageType = "CHAT_MESSAGE"
	TypeListLobbies MessageType = "LIST_LOBBIES"
	TypeSetReady    MessageType = "SET_READY"
	TypeStartGame   MessageType = "START_GAME"

	// Server-to-client message types
	TypeConnected     MessageType = "CONNECTED"
	TypeError         MessageType = "ERROR"
	TypeLobbyCreated  MessageType = "LOBBY_CREATED"
	TypeLobbyJoined   MessageType = "LOBBY_JOINED"
	TypeLobbyLeft     MessageType = "LOBBY_LEFT"
	TypeLobbiesListed MessageType = "LOBBIES_LISTED"
	TypeLobbyUpdated  MessageType = "LOBBY_UPDATED"
	TypeReadyUpdated  MessageType = "READY_UPDATED"
	TypeGameStarting  MessageType = "GAME_STARTING"
)

// Message represents a message exchanged between client and server
type Message struct {
	Type      MessageType            `json:"type"`
	SessionID string                 `json:"session_id,omitempty"`
	PlayerID  string                 `json:"player_id,omitempty"`
	Timestamp int64                  `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// LobbyHandler handles lobby-related WebSocket connections
type LobbyHandler struct {
	upgrader     websocket.Upgrader
	lobbyManager *models.LobbyManager
	clients      map[*websocket.Conn]string // conn -> playerID
	mu           sync.Mutex
	logger       Logger
}

// NewLobbyHandler creates a new lobby handler
func NewLobbyHandler() *LobbyHandler {
	return &LobbyHandler{
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all connections for testing
			},
		},
		lobbyManager: models.NewLobbyManager(),
		clients:      make(map[*websocket.Conn]string),
		logger:       &DefaultLogger{},
	}
}

// SetLogger sets a custom logger for the handler
func (h *LobbyHandler) SetLogger(logger Logger) {
	h.logger = logger
}

// HandleWebSocket handles WebSocket connections for lobby-related operations
func (h *LobbyHandler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.logger.Println("Upgrade error:", err)
		return
	}
	defer h.closeConnection(conn)

	// Generate a session ID
	sessionID := uuid.New().String()
	h.logger.Printf("New client connected with session ID: %s", sessionID)

	// Send connected message
	h.sendConnectedMessage(conn, sessionID)

	// Process messages from the client
	for {
		// Read message
		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			h.logger.Println("Read error:", err)
			break
		}

		h.logger.Printf("Received message: %s", rawMessage)

		// Parse the message
		var message Message
		if err := json.Unmarshal(rawMessage, &message); err != nil {
			h.logger.Println("Parse error:", err)
			h.sendErrorMessage(conn, sessionID, "Invalid message format")
			continue
		}

		// Process the message
		h.processMessage(conn, sessionID, message)
	}
}

// closeConnection handles a closed WebSocket connection
func (h *LobbyHandler) closeConnection(conn *websocket.Conn) {
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	if exists {
		delete(h.clients, conn)
	}
	h.mu.Unlock()

	// If the player was in a lobby, remove them
	if exists {
		lobby := h.lobbyManager.GetPlayerLobby(playerID)
		if lobby != nil {
			lobby.RemovePlayer(playerID)
			h.broadcastLobbyUpdate(lobby)
			h.lobbyManager.CleanupLobby(lobby.ID)
		}
	}

	conn.Close()
}

// sendMessage sends a message to a client
func (h *LobbyHandler) sendMessage(conn *websocket.Conn, messageType MessageType, sessionID string, data map[string]interface{}) {
	message := Message{
		Type:      messageType,
		SessionID: sessionID,
		Timestamp: time.Now().Unix(),
		Data:      data,
	}

	if err := conn.WriteJSON(message); err != nil {
		h.logger.Println("Write error:", err)
	}
}

// sendConnectedMessage sends a connected message to a client
func (h *LobbyHandler) sendConnectedMessage(conn *websocket.Conn, sessionID string) {
	h.sendMessage(conn, TypeConnected, sessionID, map[string]interface{}{
		"message": "Welcome to Power Grid Game Server",
	})
}

// sendErrorMessage sends an error message to a client
func (h *LobbyHandler) sendErrorMessage(conn *websocket.Conn, sessionID string, errorMessage string) {
	h.sendMessage(conn, TypeError, sessionID, map[string]interface{}{
		"message": errorMessage,
	})
}

// processMessage processes a message from a client
func (h *LobbyHandler) processMessage(conn *websocket.Conn, sessionID string, message Message) {
	switch message.Type {
	case TypeConnect:
		h.handleConnect(conn, sessionID, message)

	case TypeCreateLobby:
		h.handleCreateLobby(conn, sessionID, message)

	case TypeJoinLobby:
		h.handleJoinLobby(conn, sessionID, message)

	case TypeLeaveLobby:
		h.handleLeaveLobby(conn, sessionID, message)

	case TypeChatMessage:
		h.handleChatMessage(conn, sessionID, message)

	case TypeListLobbies:
		h.handleListLobbies(conn, sessionID, message)

	case TypeSetReady:
		h.handleSetReady(conn, sessionID, message)

	case TypeStartGame:
		h.handleStartGame(conn, sessionID, message)

	default:
		h.sendErrorMessage(conn, sessionID, "Unknown message type")
	}
}

// handleConnect handles a connect message
func (h *LobbyHandler) handleConnect(conn *websocket.Conn, sessionID string, message Message) {
	// Extract player name from message
	playerName, ok := message.Data["player_name"].(string)
	if !ok || playerName == "" {
		h.sendErrorMessage(conn, sessionID, "Player name is required")
		return
	}

	// Create a new player
	playerID := uuid.New().String()

	// Associate the connection with the player ID
	h.mu.Lock()
	h.clients[conn] = playerID
	h.mu.Unlock()

	// Send confirmation
	h.sendMessage(conn, TypeConnected, sessionID, map[string]interface{}{
		"player_id":   playerID,
		"player_name": playerName,
	})
}

// handleCreateLobby handles a create lobby message
func (h *LobbyHandler) handleCreateLobby(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Extract lobby data
	lobbyName, ok := message.Data["lobby_name"].(string)
	if !ok || lobbyName == "" {
		h.sendErrorMessage(conn, sessionID, "Lobby name is required")
		return
	}

	// Extract optional parameters
	maxPlayersFloat, _ := message.Data["max_players"].(float64)
	maxPlayers := int(maxPlayersFloat)
	if maxPlayers <= 0 {
		maxPlayers = 6 // Default to 6 players (Power Grid allows 2-6 players)
	}

	// Extract player name
	playerName, ok := message.Data["player_name"].(string)
	if !ok || playerName == "" {
		h.sendErrorMessage(conn, sessionID, "Player name is required")
		return
	}

	// Create a new player
	player := &models.Player{
		ID:       playerID,
		Name:     playerName,
		JoinedAt: time.Now(),
		Conn:     conn,
	}

	// Create a lobby
	lobby := h.lobbyManager.CreateLobby(lobbyName, player, maxPlayers, "")

	// Log the lobby creation
	h.logger.Printf("Created lobby: %s (ID: %s) with host: %s", lobby.Name, lobby.ID, player.Name)

	// Send confirmation with the lobby data
	h.sendMessage(conn, TypeLobbyCreated, sessionID, map[string]interface{}{
		"lobby": lobby.ToJSON(),
	})

	// Broadcast the new lobby to all connected clients
	for client := range h.clients {
		h.sendMessage(client, TypeLobbiesListed, "", map[string]interface{}{
			"lobbies": h.lobbyManager.ListLobbiesJSON(),
		})
	}
}

// handleJoinLobby handles a join lobby message
func (h *LobbyHandler) handleJoinLobby(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Extract lobby ID
	lobbyID, ok := message.Data["lobby_id"].(string)
	if !ok || lobbyID == "" {
		h.sendErrorMessage(conn, sessionID, "Lobby ID is required")
		return
	}

	// Extract password if provided
	password, _ := message.Data["password"].(string)

	// Get the lobby
	lobby, err := h.lobbyManager.GetLobby(lobbyID)
	if err != nil {
		h.sendErrorMessage(conn, sessionID, "Lobby not found")
		return
	}

	// Check password if required
	if lobby.Password != "" && lobby.Password != password {
		h.sendErrorMessage(conn, sessionID, "Incorrect password")
		return
	}

	// Get the player info
	var player *models.Player
	for _, l := range h.lobbyManager.ListLobbies() {
		for id, p := range l.Players {
			if id == playerID {
				player = p
				break
			}
		}

		if player != nil {
			break
		}
	}

	if player == nil {
		// Create a new player if not found
		playerName, ok := message.Data["player_name"].(string)
		if !ok || playerName == "" {
			h.sendErrorMessage(conn, sessionID, "Player name is required")
			return
		}

		player = &models.Player{
			ID:       playerID,
			Name:     playerName,
			JoinedAt: time.Now(),
			Conn:     conn,
		}
	}

	// Add the player to the lobby
	if !lobby.AddPlayer(player) {
		h.sendErrorMessage(conn, sessionID, "Failed to join lobby")
		return
	}

	// Send confirmation
	h.sendMessage(conn, TypeLobbyJoined, sessionID, map[string]interface{}{
		"lobby": lobby.ToJSON(),
	})

	// Broadcast the lobby update to all players in the lobby
	h.broadcastLobbyUpdate(lobby)
}

// handleLeaveLobby handles a leave lobby message
func (h *LobbyHandler) handleLeaveLobby(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Find the player's lobby
	lobby := h.lobbyManager.GetPlayerLobby(playerID)
	if lobby == nil {
		h.sendErrorMessage(conn, sessionID, "Player not in a lobby")
		return
	}

	// Remove the player from the lobby
	lobby.RemovePlayer(playerID)

	// Send confirmation
	h.sendMessage(conn, TypeLobbyLeft, sessionID, map[string]interface{}{
		"lobby_id": lobby.ID,
	})

	// Broadcast the lobby update to all players in the lobby
	h.broadcastLobbyUpdate(lobby)

	// Clean up the lobby if empty
	h.lobbyManager.CleanupLobby(lobby.ID)
}

// handleChatMessage handles a chat message
func (h *LobbyHandler) handleChatMessage(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Extract message content
	content, ok := message.Data["content"].(string)
	if !ok || content == "" {
		h.sendErrorMessage(conn, sessionID, "Message content is required")
		return
	}

	// Find the player's lobby
	lobby := h.lobbyManager.GetPlayerLobby(playerID)
	if lobby == nil {
		h.sendErrorMessage(conn, sessionID, "Player not in a lobby")
		return
	}

	// Add the message to the lobby
	lobby.AddMessage(playerID, content)

	// Broadcast the lobby update to all players in the lobby
	h.broadcastLobbyUpdate(lobby)
}

// handleListLobbies handles a list lobbies message
func (h *LobbyHandler) handleListLobbies(conn *websocket.Conn, sessionID string, message Message) {
	// List all lobbies
	lobbies := h.lobbyManager.ListLobbiesJSON()

	// Send the list to the client
	h.sendMessage(conn, TypeLobbiesListed, sessionID, map[string]interface{}{
		"lobbies": lobbies,
	})
}

// handleSetReady handles a set ready message
func (h *LobbyHandler) handleSetReady(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Extract ready status
	readyStatus, ok := message.Data["ready"].(bool)
	if !ok {
		h.sendErrorMessage(conn, sessionID, "Ready status is required")
		return
	}

	// Find the player's lobby
	lobby := h.lobbyManager.GetPlayerLobby(playerID)
	if lobby == nil {
		h.sendErrorMessage(conn, sessionID, "Player not in a lobby")
		return
	}

	// Update the ready status
	if !lobby.SetPlayerReady(playerID, readyStatus) {
		h.sendErrorMessage(conn, sessionID, "Failed to update ready status")
		return
	}

	// Send confirmation
	h.sendMessage(conn, TypeReadyUpdated, sessionID, map[string]interface{}{
		"player_id": playerID,
		"ready":     readyStatus,
	})

	// Broadcast the lobby update to all players in the lobby
	h.broadcastLobbyUpdate(lobby)
}

// handleStartGame handles a start game message
func (h *LobbyHandler) handleStartGame(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player ID
	h.mu.Lock()
	playerID, exists := h.clients[conn]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player not found")
		return
	}

	// Find the player's lobby
	lobby := h.lobbyManager.GetPlayerLobby(playerID)
	if lobby == nil {
		h.sendErrorMessage(conn, sessionID, "Player not in a lobby")
		return
	}

	// Check if the player is the host
	var isHost bool
	for id, player := range lobby.Players {
		if id == playerID && player.IsHost {
			isHost = true
			break
		}
	}

	if !isHost {
		h.sendErrorMessage(conn, sessionID, "Only the host can start the game")
		return
	}

	// Attempt to start the game
	if !lobby.StartGame() {
		h.sendErrorMessage(conn, sessionID, "Cannot start the game")
		return
	}

	// Broadcast the game starting message to all players in the lobby
	h.broadcastGameStarting(lobby)
}

// broadcastLobbyUpdate broadcasts a lobby update to all players in the lobby
func (h *LobbyHandler) broadcastLobbyUpdate(lobby *models.Lobby) {
	// Create a JSON representation of the lobby
	lobbyJSON := lobby.ToJSON()

	// Get a copy of the players to avoid holding the lock during sends
	playersWithConns := make([]*websocket.Conn, 0, len(lobby.Players))

	// Get all player connections
	for _, player := range lobby.Players {
		if player.Conn != nil {
			playersWithConns = append(playersWithConns, player.Conn)
		}
	}

	// Send the update to each player
	for _, conn := range playersWithConns {
		h.sendMessage(conn, TypeLobbyUpdated, "", map[string]interface{}{
			"lobby": lobbyJSON,
		})
	}
}

// broadcastGameStarting broadcasts a game starting message to all players in the lobby
func (h *LobbyHandler) broadcastGameStarting(lobby *models.Lobby) {
	// Create a JSON representation of the lobby
	lobbyJSON := lobby.ToJSON()

	// Get a copy of the players to avoid holding the lock during sends
	playersWithConns := make([]*websocket.Conn, 0, len(lobby.Players))

	// Get all player connections
	for _, player := range lobby.Players {
		if player.Conn != nil {
			playersWithConns = append(playersWithConns, player.Conn)
		}
	}

	// Send the update to each player
	for _, conn := range playersWithConns {
		h.sendMessage(conn, TypeGameStarting, "", map[string]interface{}{
			"lobby": lobbyJSON,
		})
	}
}
