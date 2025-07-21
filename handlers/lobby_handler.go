package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/internal/maps"
	"powergrid/internal/network"
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
	TypeListMaps    MessageType = "LIST_MAPS"
	TypeSetReady    MessageType = "SET_READY"
	TypeStartGame   MessageType = "START_GAME"

	// Server-to-client message types
	TypeConnected     MessageType = "CONNECTED"
	TypeError         MessageType = "ERROR"
	TypeLobbyCreated  MessageType = "LOBBY_CREATED"
	TypeLobbyJoined   MessageType = "LOBBY_JOINED"
	TypeLobbyLeft     MessageType = "LOBBY_LEFT"
	TypeLobbiesListed MessageType = "LOBBIES_LISTED"
	TypeMapsListed    MessageType = "MAPS_LISTED"
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

// PlayerSession tracks a player's session state
type PlayerSession struct {
	PlayerID      string
	PlayerName    string
	Connection    *websocket.Conn
	ConnMutex     sync.Mutex // Mutex for thread-safe writes
	CreatedAt     time.Time
	LastActivity  time.Time
}

// LobbyHandler handles lobby-related WebSocket connections
type LobbyHandler struct {
	upgrader      websocket.Upgrader
	lobbyManager  *models.LobbyManager
	mapManager    *maps.MapManager
	sessions      map[string]*PlayerSession  // sessionID -> PlayerSession
	connections   map[*websocket.Conn]string // conn -> sessionID for cleanup
	mu            sync.Mutex
	logger        Logger
	cleanupStop   chan struct{} // Channel to stop cleanup routine
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
		sessions:     make(map[string]*PlayerSession),
		connections:  make(map[*websocket.Conn]string),
		logger:       &DefaultLogger{},
	}
}

// SetLogger sets a custom logger for the handler
func (h *LobbyHandler) SetLogger(logger Logger) {
	h.logger = logger
}

// SetMapManager sets the map manager for the handler
func (h *LobbyHandler) SetMapManager(mapManager *maps.MapManager) {
	h.mapManager = mapManager
}

// GetLobbyManager returns the lobby manager instance
func (h *LobbyHandler) GetLobbyManager() *models.LobbyManager {
	return h.lobbyManager
}

// StartSessionCleanup starts a goroutine that periodically cleans up inactive sessions
func (h *LobbyHandler) StartSessionCleanup(interval time.Duration, maxInactivity time.Duration) {
	h.cleanupStop = make(chan struct{})
	
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				h.cleanupInactiveSessions(maxInactivity)
			case <-h.cleanupStop:
				h.logger.Println("[backend] Session cleanup routine stopped")
				return
			}
		}
	}()
	h.logger.Printf("[backend] Session cleanup started: interval=%v, maxInactivity=%v", interval, maxInactivity)
}

// StopSessionCleanup stops the session cleanup routine
func (h *LobbyHandler) StopSessionCleanup() {
	if h.cleanupStop != nil {
		close(h.cleanupStop)
		h.cleanupStop = nil
	}
}

// GetSessionInfo returns information about current sessions for debugging
func (h *LobbyHandler) GetSessionInfo() map[string]interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	sessionInfo := make(map[string]interface{})
	sessionInfo["total_sessions"] = len(h.sessions)
	sessionInfo["total_connections"] = len(h.connections)
	
	sessions := make([]map[string]interface{}, 0, len(h.sessions))
	for sessionID, session := range h.sessions {
		sessionData := map[string]interface{}{
			"session_id":    sessionID,
			"player_id":     session.PlayerID,
			"player_name":   session.PlayerName,
			"created_at":    session.CreatedAt,
			"last_activity": session.LastActivity,
			"has_connection": session.Connection != nil,
		}
		sessions = append(sessions, sessionData)
	}
	sessionInfo["sessions"] = sessions
	
	return sessionInfo
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

	// Generate a temporary session ID for this connection
	// The actual session ID will be determined when the client sends its first message
	tempSessionID := uuid.New().String()
	h.logger.Printf("[backend] New WebSocket connection established (temp ID: %s)", tempSessionID)

	// Don't send initial connected message - wait for client to identify itself
	
	// Process messages from the client
	for {
		// Read message
		_, rawMessage, err := conn.ReadMessage()
		if err != nil {
			h.logger.Printf("[backend] Connection closed: %v", err)
			break
		}

		h.logger.Printf("[backend] Received message: %s", rawMessage)

		// Parse the message
		var message Message
		if err := json.Unmarshal(rawMessage, &message); err != nil {
			h.logger.Printf("[backend] Parse error: %v", err)
			h.sendErrorMessage(conn, tempSessionID, "Invalid message format")
			continue
		}

		// Use client's session ID if provided, otherwise use temp ID
		sessionID := message.SessionID
		if sessionID == "" {
			sessionID = tempSessionID
			h.logger.Printf("[backend] No session ID in message, using temp ID: %s", tempSessionID)
		}

		// Process the message with the determined session ID
		h.processMessage(conn, sessionID, message)
	}
}

// closeConnection handles a closed WebSocket connection
func (h *LobbyHandler) closeConnection(conn *websocket.Conn) {
	h.mu.Lock()
	sessionID, exists := h.connections[conn]
	var playerID string
	if exists {
		delete(h.connections, conn)
		// Update the session's connection to nil (player session remains for potential reconnection)
		if session, sessionExists := h.sessions[sessionID]; sessionExists {
			playerID = session.PlayerID
			session.Connection = nil
			h.logger.Printf("[backend] Connection closed for session %s (PlayerID: %s). Session retained for reconnection.", sessionID, playerID)
		}
	}
	h.mu.Unlock()

	// Note: We don't remove the player from lobbies on connection close
	// This allows for reconnection without losing lobby state
	// Players are only removed from lobbies on explicit LEAVE_LOBBY or after a timeout
	
	conn.Close()
}

// cleanupInactiveSessions removes sessions that haven't been active for a specified duration
func (h *LobbyHandler) cleanupInactiveSessions(maxInactivity time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	now := time.Now()
	var sessionsToRemove []string
	
	for sessionID, session := range h.sessions {
		if now.Sub(session.LastActivity) > maxInactivity {
			sessionsToRemove = append(sessionsToRemove, sessionID)
			
			// Remove player from lobby if they're in one
			if session.PlayerID != "" {
				lobby := h.lobbyManager.GetPlayerLobby(session.PlayerID)
				if lobby != nil {
					lobby.RemovePlayer(session.PlayerID)
					h.broadcastLobbyUpdate(lobby)
					h.lobbyManager.CleanupLobby(lobby.ID)
				}
			}
		}
	}
	
	// Remove inactive sessions
	for _, sessionID := range sessionsToRemove {
		delete(h.sessions, sessionID)
		h.logger.Printf("[backend] Cleaned up inactive session: %s", sessionID)
	}
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

// sendMessageToSession sends a message to a client's session (thread-safe)
func (h *LobbyHandler) sendMessageToSession(sessionID string, messageType MessageType, data map[string]interface{}) {
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists || session.Connection == nil {
		return
	}

	session.ConnMutex.Lock()
	defer session.ConnMutex.Unlock()

	h.sendMessage(session.Connection, messageType, sessionID, data)
}

// sendErrorMessage sends an error message to a client
func (h *LobbyHandler) sendErrorMessage(conn *websocket.Conn, sessionID string, errorMessage string) {
	h.sendMessage(conn, TypeError, sessionID, map[string]interface{}{
		"message": errorMessage,
	})
}

// processMessage processes a message from a client
func (h *LobbyHandler) processMessage(conn *websocket.Conn, sessionID string, message Message) {
	// Update last activity time for existing sessions
	h.mu.Lock()
	if session, exists := h.sessions[sessionID]; exists {
		session.LastActivity = time.Now()
		// Update connection if it's different (reconnection case)
		if session.Connection != conn {
			h.logger.Printf("[backend] Updating connection for existing session %s", sessionID)
			session.Connection = conn
			h.connections[conn] = sessionID
		}
	}
	h.mu.Unlock()
	
	h.logger.Printf("[backend] Processing message type %s with sessionID: %s", message.Type, sessionID)
	
	switch message.Type {
	case TypeConnect:
		h.handleConnect(conn, sessionID, message)

	case TypeCreateLobby:
		// Ensure session exists before processing CREATE_LOBBY
		h.mu.Lock()
		_, sessionExists := h.sessions[sessionID]
		h.mu.Unlock()
		if !sessionExists {
			h.logger.Printf("[backend] CREATE_LOBBY attempted without session for sessionID: %s", sessionID)
			h.sendErrorMessage(conn, sessionID, "No active session found. Please send CONNECT message first.")
			return
		}
		h.handleCreateLobby(conn, sessionID, message)

	case TypeJoinLobby:
		h.handleJoinLobby(conn, sessionID, message)

	case TypeLeaveLobby:
		h.handleLeaveLobby(conn, sessionID, message)

	case TypeChatMessage:
		h.handleChatMessage(conn, sessionID, message)

	case TypeListLobbies:
		h.handleListLobbies(conn, sessionID, message)

	case TypeListMaps:
		h.handleListMaps(conn, sessionID, message)

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

	h.mu.Lock()
	
	// Check if session already exists (reconnection)
	if existingSession, exists := h.sessions[sessionID]; exists {
		h.logger.Printf("[backend] Restoring existing session: %s (PlayerID: %s, PlayerName: %s)", 
			sessionID, existingSession.PlayerID, existingSession.PlayerName)
		
		// Update connection for existing session
		existingSession.Connection = conn
		existingSession.LastActivity = time.Now()
		h.connections[conn] = sessionID
		h.mu.Unlock()
		
		// Send confirmation with existing player ID and session restored flag
		h.sendMessage(conn, TypeConnected, sessionID, map[string]interface{}{
			"player_id":      existingSession.PlayerID,
			"player_name":    existingSession.PlayerName,
			"session_id":     sessionID,
			"reconnected":    true,
			"message":        "Session restored successfully",
		})
		return
	}

	// Create a new player session
	playerID := uuid.New().String()
	session := &PlayerSession{
		PlayerID:     playerID,
		PlayerName:   playerName,
		Connection:   conn,
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
	}

	// Store the session and connection mapping
	h.sessions[sessionID] = session
	h.connections[conn] = sessionID
	h.mu.Unlock()

	h.logger.Printf("[backend] New player session created: %s (PlayerID: %s, PlayerName: %s)", 
		sessionID, playerID, playerName)

	// Send confirmation with session created flag
	h.sendMessage(conn, TypeConnected, sessionID, map[string]interface{}{
		"player_id":      playerID,
		"player_name":    playerName,
		"session_id":     sessionID,
		"reconnected":    false,
		"message":        "New session created successfully",
	})
}

// handleCreateLobby handles a create lobby message
func (h *LobbyHandler) handleCreateLobby(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	totalSessions := len(h.sessions)
	h.mu.Unlock()

	if !exists {
		h.logger.Printf("[backend] ERROR: Player session not found for sessionID: %s (total sessions: %d)", sessionID, totalSessions)
		h.mu.Lock()
		h.logger.Printf("[backend] Available sessions:")
		for sid, sess := range h.sessions {
			h.logger.Printf("[backend]   - %s: PlayerID=%s, PlayerName=%s, HasConnection=%t", sid, sess.PlayerID, sess.PlayerName, sess.Connection != nil)
		}
		
		// Check if this connection has a different session mapping
		if mappedSessionID, exists := h.connections[conn]; exists {
			h.logger.Printf("[backend] Connection is mapped to different session: %s", mappedSessionID)
		} else {
			h.logger.Printf("[backend] Connection has no session mapping")
		}
		h.mu.Unlock()
		
		h.sendErrorMessage(conn, sessionID, "Player session not found. Please send CONNECT message first.")
		return
	}

	playerID := session.PlayerID
	h.logger.Printf("[backend] Player %s (SessionID: %s) creating lobby", playerID, sessionID)

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

	// Extract map ID
	mapID, _ := message.Data["map_id"].(string)
	if mapID == "" {
		mapID = "usa" // Default to USA map
	}

	// Validate map exists if map manager is available
	if h.mapManager != nil {
		if _, exists := h.mapManager.GetMap(mapID); !exists {
			h.sendErrorMessage(conn, sessionID, "Invalid map selected")
			return
		}
	}

	// Get player name from session
	playerName := session.PlayerName

	// Create a new player
	player := &models.Player{
		ID:       playerID,
		Name:     playerName,
		JoinedAt: time.Now(),
		Conn:     conn,
	}

	// Create a lobby
	lobby := h.lobbyManager.CreateLobby(lobbyName, player, maxPlayers, "", mapID)

	// Log the lobby creation
	h.logger.Printf("Created lobby: %s (ID: %s) with host: %s", lobby.Name, lobby.ID, player.Name)

	// Send confirmation with the lobby data
	h.sendMessage(conn, TypeLobbyCreated, sessionID, map[string]interface{}{
		"lobby": lobby.ToJSON(),
	})

	// Broadcast the new lobby to all connected clients
	h.mu.Lock()
	for _, session := range h.sessions {
		if session.Connection != nil {
			h.sendMessage(session.Connection, TypeLobbiesListed, "", map[string]interface{}{
				"lobbies": h.lobbyManager.ListLobbiesJSON(),
			})
		}
	}
	h.mu.Unlock()
}

// handleJoinLobby handles a join lobby message
func (h *LobbyHandler) handleJoinLobby(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player session not found")
		return
	}

	playerID := session.PlayerID

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
		// Use the player name from the session
		if session.PlayerName == "" {
			h.sendErrorMessage(conn, sessionID, "Player name is required")
			return
		}

		player = &models.Player{
			ID:       playerID,
			Name:     session.PlayerName,
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
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player session not found")
		return
	}

	playerID := session.PlayerID

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
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player session not found")
		return
	}

	playerID := session.PlayerID

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

// handleListMaps handles a list maps message
func (h *LobbyHandler) handleListMaps(conn *websocket.Conn, sessionID string, message Message) {
	if h.mapManager == nil {
		h.sendErrorMessage(conn, sessionID, "Map manager not available")
		return
	}

	// Get all available maps
	mapList := h.mapManager.GetMapList()

	// Send the list to the client
	h.sendMessage(conn, TypeMapsListed, sessionID, map[string]interface{}{
		"maps": mapList,
	})
}

// handleSetReady handles a set ready message
func (h *LobbyHandler) handleSetReady(conn *websocket.Conn, sessionID string, message Message) {
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player session not found")
		return
	}

	playerID := session.PlayerID

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
	// Get the player session
	h.mu.Lock()
	session, exists := h.sessions[sessionID]
	h.mu.Unlock()

	if !exists {
		h.sendErrorMessage(conn, sessionID, "Player session not found")
		return
	}

	playerID := session.PlayerID

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

	// Create a game instance in the game server
	gameID, err := h.createGameFromLobby(lobby)
	if err != nil {
		h.sendErrorMessage(conn, sessionID, "Failed to create game: " + err.Error())
		// Revert lobby status
		lobby.Status = models.LobbyStatusWaiting
		return
	}

	// Broadcast the game starting message to all players in the lobby
	h.broadcastGameStarting(lobby, gameID)
}

// broadcastLobbyUpdate broadcasts a lobby update to all players in the lobby
func (h *LobbyHandler) broadcastLobbyUpdate(lobby *models.Lobby) {
	// Create a JSON representation of the lobby
	lobbyJSON := lobby.ToJSON()

	// Get sessions for all players in the lobby
	h.mu.Lock()
	sessionsToSend := make([]*PlayerSession, 0, len(lobby.Players))
	for playerID := range lobby.Players {
		// Find the session for this player
		for _, session := range h.sessions {
			if session.PlayerID == playerID && session.Connection != nil {
				sessionsToSend = append(sessionsToSend, session)
				break
			}
		}
	}
	h.mu.Unlock()

	// Send the update to each player
	for _, session := range sessionsToSend {
		// Use mutex to ensure thread-safe write
		session.ConnMutex.Lock()
		h.sendMessage(session.Connection, TypeLobbyUpdated, "", map[string]interface{}{
			"lobby": lobbyJSON,
		})
		session.ConnMutex.Unlock()
	}
}

// createGameFromLobby creates a game instance from a lobby
func (h *LobbyHandler) createGameFromLobby(lobby *models.Lobby) (string, error) {
	// Create the game in the game server
	gameManager := network.Games
	newGame, err := gameManager.CreateGame(lobby.Name, lobby.MapID)
	if err != nil {
		return "", err
	}

	// Add players to the game
	colors := []string{"red", "blue", "green", "yellow", "purple", "black"}
	colorIndex := 0
	for playerID, player := range lobby.Players {
		// Assign colors in order
		color := ""
		if colorIndex < len(colors) {
			color = colors[colorIndex]
			colorIndex++
		}
		err := newGame.AddPlayer(playerID, player.Name, color, nil)
		if err != nil {
			// If we fail to add a player, remove the game
			gameManager.RemoveGame(newGame.ID)
			return "", err
		}
	}

	return newGame.ID, nil
}

// broadcastGameStarting broadcasts a game starting message to all players in the lobby
func (h *LobbyHandler) broadcastGameStarting(lobby *models.Lobby, gameID string) {
	// Create a JSON representation of the lobby
	lobbyJSON := lobby.ToJSON()

	// Get sessions for all players in the lobby
	h.mu.Lock()
	sessionsToSend := make([]*PlayerSession, 0, len(lobby.Players))
	for playerID := range lobby.Players {
		// Find the session for this player
		for _, session := range h.sessions {
			if session.PlayerID == playerID && session.Connection != nil {
				sessionsToSend = append(sessionsToSend, session)
				break
			}
		}
	}
	h.mu.Unlock()

	// Send the update to each player
	for _, session := range sessionsToSend {
		// Use mutex to ensure thread-safe write
		session.ConnMutex.Lock()
		h.sendMessage(session.Connection, TypeGameStarting, "", map[string]interface{}{
			"lobby":    lobbyJSON,
			"game_id":  gameID,
			"game_url": "/game", // This could be a different server in production
		})
		session.ConnMutex.Unlock()
	}
}
