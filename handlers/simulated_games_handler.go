package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"powergrid/models"
	"powergrid/pkg/logger"
)

// AIClientProcess represents a running AI client process
type AIClientProcess struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Process   *exec.Cmd `json:"-"`
	ProcessID int       `json:"processId"`
	Status    string    `json:"status"` // launching, connected, ready, error
	LobbyID   string    `json:"lobbyId"`
	CreatedAt time.Time `json:"createdAt"`
}

// SimulatedGame represents a game with AI players
type SimulatedGame struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Status         string                 `json:"status"` // creating, running, paused, completed
	AIPlayerCount  int                    `json:"aiPlayerCount"`
	AIDifficulty   string                 `json:"aiDifficulty"`
	MapID          string                 `json:"mapId"`
	GameSpeed      string                 `json:"gameSpeed"`
	StartedAt      *time.Time             `json:"startedAt,omitempty"`
	EndedAt        *time.Time             `json:"endedAt,omitempty"`
	Duration       int                    `json:"duration"` // in seconds
	CurrentRound   int                    `json:"currentRound"`
	CurrentPhase   string                 `json:"currentPhase"`
	AIClients      []AIClientProcess      `json:"aiClients"`
	DecisionLog    []AIDecision           `json:"-"` // Not included in JSON responses
}

// AIDecision represents a decision made by an AI player
type AIDecision struct {
	ID           string                 `json:"id"`
	GameID       string                 `json:"gameId"`
	PlayerID     string                 `json:"playerId"`
	PlayerName   string                 `json:"playerName"`
	Timestamp    time.Time              `json:"timestamp"`
	Phase        string                 `json:"phase"`
	DecisionType string                 `json:"decisionType"`
	Decision     interface{}            `json:"decision"`
	Reasoning    string                 `json:"reasoning"`
	Factors      map[string]interface{} `json:"factors"`
}

// SimulatedGamesManager manages AI games and clients
type SimulatedGamesManager struct {
	games      map[string]*SimulatedGame
	clients    map[string]*AIClientProcess
	decisions  map[string][]AIDecision // gameID -> decisions
	mu         sync.RWMutex
	lobbyMgr   *models.LobbyManager
	logger     *logger.ColoredLogger
	wsUpgrader websocket.Upgrader
}

// NewSimulatedGamesManager creates a new simulated games manager
func NewSimulatedGamesManager(lobbyMgr *models.LobbyManager) *SimulatedGamesManager {
	return &SimulatedGamesManager{
		games:     make(map[string]*SimulatedGame),
		clients:   make(map[string]*AIClientProcess),
		decisions: make(map[string][]AIDecision),
		lobbyMgr:  lobbyMgr,
		logger:    logger.ServerLogger,
		wsUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins for admin interface
			},
		},
	}
}

// RegisterRoutes registers HTTP routes for simulated games
func (m *SimulatedGamesManager) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/admin/simulated/create-lobby", m.handleCreateLobby)
	mux.HandleFunc("/api/admin/simulated/launch-ai-client", m.handleLaunchAIClient)
	mux.HandleFunc("/api/admin/simulated/lobby/", m.handleLobbyRoutes)
	mux.HandleFunc("/api/admin/simulated/start-game/", m.handleStartGameRoute)
	mux.HandleFunc("/api/admin/simulated/control/", m.handleControlGameRoute)
	mux.HandleFunc("/api/admin/simulated/games", m.handleGamesRoute)
	mux.HandleFunc("/api/admin/simulated/games/", m.handleGameRoutes)
	mux.HandleFunc("/api/admin/simulated/ai-metrics", m.handleGetAIMetrics)
	
	// WebSocket endpoint for real-time monitoring
	mux.HandleFunc("/ws/admin/game/", m.handleGameMonitorWS)
}

// Route handlers that parse path parameters
func (m *SimulatedGamesManager) handleLobbyRoutes(w http.ResponseWriter, r *http.Request) {
	// Extract lobbyId from path
	path := r.URL.Path
	if strings.HasSuffix(path, "/ready") {
		lobbyID := strings.TrimSuffix(strings.TrimPrefix(path, "/api/admin/simulated/lobby/"), "/ready")
		r = mux.SetURLVars(r, map[string]string{"lobbyId": lobbyID})
		m.handleCheckLobbyReady(w, r)
	}
}

func (m *SimulatedGamesManager) handleStartGameRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	lobbyID := strings.TrimPrefix(r.URL.Path, "/api/admin/simulated/start-game/")
	r = mux.SetURLVars(r, map[string]string{"lobbyId": lobbyID})
	m.handleStartGame(w, r)
}

func (m *SimulatedGamesManager) handleControlGameRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	gameID := strings.TrimPrefix(r.URL.Path, "/api/admin/simulated/control/")
	r = mux.SetURLVars(r, map[string]string{"gameId": gameID})
	m.handleControlGame(w, r)
}

func (m *SimulatedGamesManager) handleGamesRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		m.handleListGames(w, r)
	} else {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (m *SimulatedGamesManager) handleGameRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	basePrefix := "/api/admin/simulated/games/"
	
	if !strings.HasPrefix(path, basePrefix) {
		http.NotFound(w, r)
		return
	}
	
	subPath := strings.TrimPrefix(path, basePrefix)
	parts := strings.Split(subPath, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		http.NotFound(w, r)
		return
	}
	
	gameID := parts[0]
	r = mux.SetURLVars(r, map[string]string{"gameId": gameID})
	
	if len(parts) == 1 {
		switch r.Method {
		case "GET":
			m.handleGetGame(w, r)
		case "DELETE":
			m.handleDeleteGame(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	} else if len(parts) == 2 {
		switch parts[1] {
		case "decisions":
			m.handleGetDecisions(w, r)
		case "export":
			m.handleExportGame(w, r)
		default:
			http.NotFound(w, r)
		}
	} else {
		http.NotFound(w, r)
	}
}

// handleCreateLobby creates a new lobby for AI-only games
func (m *SimulatedGamesManager) handleCreateLobby(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		MaxPlayers int    `json:"maxPlayers"`
		MapID      string `json:"mapId"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Create a dummy host player for AI lobbies
	hostPlayer := &models.Player{
		ID:       uuid.New().String(),
		Name:     "AI_Host",
		IsHost:   true,
		IsReady:  true,
		JoinedAt: time.Now(),
	}
	
	// Create lobby through the lobby manager
	lobby := m.lobbyMgr.CreateLobby(req.Name, hostPlayer, req.MaxPlayers, "", req.MapID)
	lobby.IsAIOnly = true // Mark as AI-only lobby
	
	// Create simulated game record
	game := &SimulatedGame{
		ID:            lobby.ID,
		Name:          req.Name,
		Status:        "creating",
		MapID:         req.MapID,
		AIClients:     make([]AIClientProcess, 0),
		CurrentRound:  0,
		CurrentPhase:  "waiting",
	}
	
	m.mu.Lock()
	m.games[lobby.ID] = game
	m.decisions[lobby.ID] = make([]AIDecision, 0)
	m.mu.Unlock()
	
	m.logger.Info("Created AI-only lobby: %s", lobby.ID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lobby)
}

// handleLaunchAIClient launches a new AI client process
func (m *SimulatedGamesManager) handleLaunchAIClient(w http.ResponseWriter, r *http.Request) {
	var req struct {
		LobbyID     string `json:"lobbyId"`
		PlayerName  string `json:"playerName"`
		Difficulty  string `json:"difficulty"`
		Personality string `json:"personality"`
		Strategy    string `json:"strategy"`
		GameSpeed   string `json:"gameSpeed"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	// Validate lobby exists
	m.mu.RLock()
	game, exists := m.games[req.LobbyID]
	m.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Lobby not found", http.StatusNotFound)
		return
	}
	
	// Calculate decision delay based on game speed
	decisionDelay := m.getDecisionDelay(req.GameSpeed)
	
	// Build command arguments
	args := []string{
		"-server", "ws://localhost:5080/ws",
		"-game", req.LobbyID,
		"-name", req.PlayerName,
		"-strategy", req.Personality, // Use personality as strategy for AI client
		"-auto-play", "true",
		"-think-time", decisionDelay,
		"-log-level", "debug",
		"-log-decisions", // Enable decision logging
	}
	
	// Launch the AI client process
	cmd := exec.Command("./ai_client", args...)
	
	// Start the process
	if err := cmd.Start(); err != nil {
		m.logger.Error("Failed to launch AI client: %v", err)
		http.Error(w, fmt.Sprintf("Failed to launch AI client: %v", err), http.StatusInternalServerError)
		return
	}
	
	// Create client record
	client := &AIClientProcess{
		ID:        uuid.New().String(),
		Name:      req.PlayerName,
		Process:   cmd,
		ProcessID: cmd.Process.Pid,
		Status:    "launching",
		LobbyID:   req.LobbyID,
		CreatedAt: time.Now(),
	}
	
	// Add to tracking
	m.mu.Lock()
	m.clients[client.ID] = client
	game.AIClients = append(game.AIClients, *client)
	game.AIPlayerCount++
	game.AIDifficulty = req.Difficulty
	game.GameSpeed = req.GameSpeed
	m.mu.Unlock()
	
	// Monitor the process
	go m.monitorAIClient(client)
	
	m.logger.Info("Launched AI client %s (PID: %d) for lobby %s", client.Name, client.ProcessID, req.LobbyID)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(client)
}

// handleCheckLobbyReady checks if all AI clients are ready
func (m *SimulatedGamesManager) handleCheckLobbyReady(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lobbyID := vars["lobbyId"]
	
	lobby, err := m.lobbyMgr.GetLobby(lobbyID)
	if err != nil || lobby == nil {
		http.Error(w, "Lobby not found", http.StatusNotFound)
		return
	}
	
	m.mu.RLock()
	game := m.games[lobbyID]
	m.mu.RUnlock()
	
	ready := len(lobby.Players) == lobby.MaxPlayers
	
	response := map[string]interface{}{
		"ready":      ready,
		"players":    len(lobby.Players),
		"maxPlayers": lobby.MaxPlayers,
	}
	
	// Update client statuses
	if ready && game != nil {
		m.mu.Lock()
		for i := range game.AIClients {
			if game.AIClients[i].Status == "launching" {
				game.AIClients[i].Status = "ready"
			}
		}
		m.mu.Unlock()
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleStartGame starts the simulated game
func (m *SimulatedGamesManager) handleStartGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	lobbyID := vars["lobbyId"]
	
	m.mu.Lock()
	game, exists := m.games[lobbyID]
	if !exists {
		m.mu.Unlock()
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	// Update game status
	now := time.Now()
	game.Status = "running"
	game.StartedAt = &now
	game.CurrentRound = 1
	game.CurrentPhase = "auction"
	m.mu.Unlock()
	
	// Start the game through network game manager
	// TODO: Implement game start logic
	// For now, just update the lobby status
	lobby, err := m.lobbyMgr.GetLobby(lobbyID)
	if err != nil || lobby == nil {
		http.Error(w, "Lobby not found", http.StatusNotFound)
		return
	}
	lobby.Status = models.LobbyStatusInGame
	
	m.logger.Info("Started simulated game: %s", lobbyID)
	
	response := map[string]interface{}{
		"gameId": lobbyID,
		"status": "started",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleControlGame controls game execution (pause, resume, stop, speed)
func (m *SimulatedGamesManager) handleControlGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	
	var req struct {
		Action string `json:"action"` // pause, resume, stop, speed
		Speed  string `json:"speed,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	m.mu.Lock()
	game, exists := m.games[gameID]
	if !exists {
		m.mu.Unlock()
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	switch req.Action {
	case "pause":
		game.Status = "paused"
		// TODO: Send pause signal to AI clients
		
	case "resume":
		game.Status = "running"
		// TODO: Send resume signal to AI clients
		
	case "stop":
		game.Status = "completed"
		now := time.Now()
		game.EndedAt = &now
		if game.StartedAt != nil {
			game.Duration = int(now.Sub(*game.StartedAt).Seconds())
		}
		// Stop all AI client processes
		for _, client := range game.AIClients {
			if clientProc, ok := m.clients[client.ID]; ok {
				m.stopAIClient(clientProc)
			}
		}
		
	case "speed":
		if req.Speed != "" {
			game.GameSpeed = req.Speed
			// TODO: Update AI client decision delays
		}
		
	default:
		m.mu.Unlock()
		http.Error(w, "Invalid action", http.StatusBadRequest)
		return
	}
	
	m.mu.Unlock()
	
	m.logger.Info("Game %s control action: %s", gameID, req.Action)
	
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

// handleListGames returns all simulated games
func (m *SimulatedGamesManager) handleListGames(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	games := make([]*SimulatedGame, 0, len(m.games))
	for _, game := range m.games {
		// Update duration for running games
		if game.Status == "running" && game.StartedAt != nil {
			game.Duration = int(time.Since(*game.StartedAt).Seconds())
		}
		games = append(games, game)
	}
	m.mu.RUnlock()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(games)
}

// handleGetGame returns details of a specific game
func (m *SimulatedGamesManager) handleGetGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	
	m.mu.RLock()
	game, exists := m.games[gameID]
	m.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	// Update duration for running games
	if game.Status == "running" && game.StartedAt != nil {
		game.Duration = int(time.Since(*game.StartedAt).Seconds())
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(game)
}

// handleGetDecisions returns AI decisions for a game
func (m *SimulatedGamesManager) handleGetDecisions(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	
	limit := 100
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	
	m.mu.RLock()
	decisions, exists := m.decisions[gameID]
	m.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	// Return latest decisions up to limit
	start := 0
	if len(decisions) > limit {
		start = len(decisions) - limit
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(decisions[start:])
}

// handleExportGame exports game data
func (m *SimulatedGamesManager) handleExportGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	format := r.URL.Query().Get("format")
	if format == "" {
		format = "json"
	}
	
	m.mu.RLock()
	game, gameExists := m.games[gameID]
	decisions, _ := m.decisions[gameID]
	m.mu.RUnlock()
	
	if !gameExists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	switch format {
	case "json":
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"game_%s.json\"", gameID))
		
		exportData := map[string]interface{}{
			"game":      game,
			"decisions": decisions,
		}
		json.NewEncoder(w).Encode(exportData)
		
	case "csv":
		// TODO: Implement CSV export
		http.Error(w, "CSV export not implemented yet", http.StatusNotImplemented)
		
	default:
		http.Error(w, "Invalid format", http.StatusBadRequest)
	}
}

// handleDeleteGame deletes a completed game
func (m *SimulatedGamesManager) handleDeleteGame(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	
	m.mu.Lock()
	game, exists := m.games[gameID]
	if !exists {
		m.mu.Unlock()
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	// Stop any running AI clients
	if game.Status == "running" {
		for _, client := range game.AIClients {
			if clientProc, ok := m.clients[client.ID]; ok {
				m.stopAIClient(clientProc)
			}
		}
	}
	
	// Remove from tracking
	delete(m.games, gameID)
	delete(m.decisions, gameID)
	m.mu.Unlock()
	
	m.logger.Info("Deleted simulated game: %s", gameID)
	
	w.WriteHeader(http.StatusOK)
}

// handleGetAIMetrics returns AI performance metrics
func (m *SimulatedGamesManager) handleGetAIMetrics(w http.ResponseWriter, r *http.Request) {
	// TODO: Implement metrics aggregation
	metrics := map[string]interface{}{
		"totalGames":     len(m.games),
		"activeGames":    m.countGamesByStatus("running"),
		"completedGames": m.countGamesByStatus("completed"),
		"totalDecisions": m.countTotalDecisions(),
		"avgDecisionTime": 2.0, // seconds
		"winRates": map[string]float64{
			"aggressive":   0.45,
			"conservative": 0.35,
			"balanced":     0.50,
			"random":       0.20,
		},
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

// handleGameMonitorWS handles WebSocket connections for real-time game monitoring
func (m *SimulatedGamesManager) handleGameMonitorWS(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	gameID := vars["gameId"]
	
	// Validate game exists
	m.mu.RLock()
	_, exists := m.games[gameID]
	m.mu.RUnlock()
	
	if !exists {
		http.Error(w, "Game not found", http.StatusNotFound)
		return
	}
	
	// Upgrade to WebSocket
	conn, err := m.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		m.logger.Error("WebSocket upgrade failed: %v", err)
		return
	}
	defer conn.Close()
	
	m.logger.Info("Admin WebSocket connected for game monitoring: %s", gameID)
	
	// Subscribe to game updates
	updates := make(chan interface{}, 100)
	m.subscribeToGameUpdates(gameID, updates)
	defer m.unsubscribeFromGameUpdates(gameID, updates)
	
	// Send updates to client
	for update := range updates {
		if err := conn.WriteJSON(update); err != nil {
			m.logger.Error("WebSocket write error: %v", err)
			break
		}
	}
}

// Helper methods

func (m *SimulatedGamesManager) getDecisionDelay(gameSpeed string) string {
	switch gameSpeed {
	case "slow":
		return "5s"
	case "normal":
		return "2s"
	case "fast":
		return "500ms"
	case "instant":
		return "0s"
	default:
		return "2s"
	}
}

func (m *SimulatedGamesManager) monitorAIClient(client *AIClientProcess) {
	// Wait for process to exit
	err := client.Process.Wait()
	
	m.mu.Lock()
	if err != nil {
		client.Status = "error"
		m.logger.Error("AI client %s exited with error: %v", client.Name, err)
	} else {
		client.Status = "completed"
		m.logger.Info("AI client %s exited normally", client.Name)
	}
	m.mu.Unlock()
}

func (m *SimulatedGamesManager) stopAIClient(client *AIClientProcess) {
	if client.Process != nil && client.Process.Process != nil {
		// Try graceful shutdown first
		client.Process.Process.Signal(os.Interrupt)
		
		// Give it time to shutdown
		done := make(chan error, 1)
		go func() {
			done <- client.Process.Wait()
		}()
		
		select {
		case <-done:
			// Process exited
		case <-time.After(5 * time.Second):
			// Force kill
			client.Process.Process.Kill()
		}
	}
}

func (m *SimulatedGamesManager) countGamesByStatus(status string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	count := 0
	for _, game := range m.games {
		if game.Status == status {
			count++
		}
	}
	return count
}

func (m *SimulatedGamesManager) countTotalDecisions() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	total := 0
	for _, decisions := range m.decisions {
		total += len(decisions)
	}
	return total
}

func (m *SimulatedGamesManager) subscribeToGameUpdates(gameID string, updates chan<- interface{}) {
	// TODO: Implement subscription to game events
	// This would connect to the game's event stream and forward updates
}

func (m *SimulatedGamesManager) unsubscribeFromGameUpdates(gameID string, updates chan<- interface{}) {
	close(updates)
}

// LogAIDecision logs a decision made by an AI player
func (m *SimulatedGamesManager) LogAIDecision(decision AIDecision) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if decisions, exists := m.decisions[decision.GameID]; exists {
		m.decisions[decision.GameID] = append(decisions, decision)
		
		// Broadcast to monitoring clients
		// TODO: Implement WebSocket broadcast
	}
}

// Shutdown stops all running AI clients and cleans up resources
func (m *SimulatedGamesManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.logger.Info("Shutting down simulated games manager...")
	
	// Stop all running games
	for gameID, game := range m.games {
		if game.Status == "running" {
			m.logger.Info("Stopping game %s", gameID)
			for _, client := range game.AIClients {
				if clientProc, ok := m.clients[client.ID]; ok {
					m.stopAIClient(clientProc)
				}
			}
		}
	}
	
	// Clear all data
	m.games = make(map[string]*SimulatedGame)
	m.clients = make(map[string]*AIClientProcess)
	m.decisions = make(map[string][]AIDecision)
	
	m.logger.Info("Simulated games manager shutdown complete")
}