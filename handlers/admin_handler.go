package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"

	"powergrid/internal/network"
	"powergrid/pkg/logger"
)

// AdminHandler manages admin WebSocket connections and controls
type AdminHandler struct {
	logBroadcaster *network.LogBroadcaster
	upgrader       websocket.Upgrader
	logger         *logger.ColoredLogger
	
	// Admin control channels
	shutdownChan chan bool
	restartChan  chan bool
}

// NewAdminHandler creates a new admin handler
func NewAdminHandler(logBroadcaster *network.LogBroadcaster) *AdminHandler {
	return &AdminHandler{
		logBroadcaster: logBroadcaster,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger:       logger.NewColoredLogger("ADMIN", logger.ColorBrightRed),
		shutdownChan: make(chan bool, 1),
		restartChan:  make(chan bool, 1),
	}
}

// RegisterRoutes registers admin routes
func (ah *AdminHandler) RegisterRoutes(mux *http.ServeMux) {
	// WebSocket log streaming endpoint
	mux.HandleFunc("/admin/logs/stream", ah.handleLogStream)
	
	// Admin control endpoints
	mux.HandleFunc("/admin/control/shutdown", ah.handleShutdown)
	mux.HandleFunc("/admin/control/restart", ah.handleRestart)
	mux.HandleFunc("/admin/control/status", ah.handleStatus)
	
	// Session management endpoints
	mux.HandleFunc("/admin/sessions", ah.handleSessions)
	mux.HandleFunc("/admin/sessions/kick", ah.handleKickSession)
	
	// Log management endpoints
	mux.HandleFunc("/admin/logs/history", ah.handleLogHistory)
	mux.HandleFunc("/admin/logs/stats", ah.handleLogStats)
	
	ah.logger.Info("Admin API routes registered")
}

// handleLogStream handles WebSocket connections for log streaming
func (ah *AdminHandler) handleLogStream(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := ah.upgrader.Upgrade(w, r, nil)
	if err != nil {
		ah.logger.Error("Failed to upgrade to WebSocket: %v", err)
		return
	}
	
	clientID := uuid.New().String()
	ah.logger.Info("Admin log client connected: %s", clientID)
	
	// Parse query parameters for initial filter
	filter := ah.parseLogFilter(r)
	
	// Add client to broadcaster
	ah.logBroadcaster.AddClient(conn, clientID, filter)
	
	// Handle incoming messages (filter updates, etc.)
	go ah.handleLogClientMessages(conn, clientID)
}

// parseLogFilter parses log filter from query parameters
func (ah *AdminHandler) parseLogFilter(r *http.Request) network.LogFilter {
	query := r.URL.Query()
	
	filter := network.LogFilter{
		MinLevel: network.LogLevel(query.Get("level")),
	}
	
	if filter.MinLevel == "" {
		filter.MinLevel = network.LogLevelInfo
	}
	
	if components := query["component"]; len(components) > 0 {
		filter.Components = components
	}
	
	if sessionID := query.Get("session_id"); sessionID != "" {
		filter.SessionID = sessionID
	}
	
	if playerID := query.Get("player_id"); playerID != "" {
		filter.PlayerID = playerID
	}
	
	if gameID := query.Get("game_id"); gameID != "" {
		filter.GameID = gameID
	}
	
	if keywords := query["keyword"]; len(keywords) > 0 {
		filter.Keywords = keywords
	}
	
	return filter
}

// handleLogClientMessages handles incoming WebSocket messages from log clients
func (ah *AdminHandler) handleLogClientMessages(conn *websocket.Conn, clientID string) {
	defer func() {
		ah.logBroadcaster.RemoveClient(clientID)
		conn.Close()
	}()
	
	for {
		var message map[string]interface{}
		if err := conn.ReadJSON(&message); err != nil {
			ah.logger.Error("Failed to read message from log client %s: %v", clientID, err)
			break
		}
		
		// Handle different message types
		switch message["type"] {
		case "update_filter":
			if filterData, ok := message["filter"]; ok {
				filter := ah.parseFilterFromJSON(filterData)
				ah.logBroadcaster.UpdateClientFilter(clientID, filter)
				ah.logger.Debug("Updated filter for log client: %s", clientID)
			}
		case "ping":
			// Respond to ping
			conn.WriteJSON(map[string]interface{}{
				"type": "pong",
				"timestamp": time.Now(),
			})
		}
	}
}

// parseFilterFromJSON parses log filter from JSON data
func (ah *AdminHandler) parseFilterFromJSON(data interface{}) network.LogFilter {
	filter := network.LogFilter{
		MinLevel: network.LogLevelInfo,
	}
	
	if filterMap, ok := data.(map[string]interface{}); ok {
		if level, ok := filterMap["min_level"].(string); ok {
			filter.MinLevel = network.LogLevel(level)
		}
		
		if components, ok := filterMap["components"].([]interface{}); ok {
			for _, comp := range components {
				if compStr, ok := comp.(string); ok {
					filter.Components = append(filter.Components, compStr)
				}
			}
		}
		
		if sessionID, ok := filterMap["session_id"].(string); ok {
			filter.SessionID = sessionID
		}
		
		if playerID, ok := filterMap["player_id"].(string); ok {
			filter.PlayerID = playerID
		}
		
		if gameID, ok := filterMap["game_id"].(string); ok {
			filter.GameID = gameID
		}
		
		if keywords, ok := filterMap["keywords"].([]interface{}); ok {
			for _, keyword := range keywords {
				if keywordStr, ok := keyword.(string); ok {
					filter.Keywords = append(filter.Keywords, keywordStr)
				}
			}
		}
	}
	
	return filter
}

// handleShutdown handles server shutdown requests
func (ah *AdminHandler) handleShutdown(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	ah.logger.Warn("Admin shutdown request received")
	
	// Send shutdown signal (non-blocking)
	select {
	case ah.shutdownChan <- true:
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "shutdown_initiated",
			"timestamp": time.Now(),
		})
	default:
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "shutdown_already_pending",
		})
	}
}

// handleRestart handles server restart requests
func (ah *AdminHandler) handleRestart(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	ah.logger.Warn("Admin restart request received")
	
	// Send restart signal (non-blocking)
	select {
	case ah.restartChan <- true:
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "restart_initiated",
			"timestamp": time.Now(),
		})
	default:
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "restart_already_pending",
		})
	}
}

// handleStatus handles server status requests
func (ah *AdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := ah.logBroadcaster.GetStats()
	
	response := map[string]interface{}{
		"status": "running",
		"timestamp": time.Now(),
		"uptime": time.Since(time.Now()), // This should be calculated from server start time
		"log_streaming": stats,
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleSessions handles session management requests
func (ah *AdminHandler) handleSessions(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// TODO: Implement session listing
	// This would require integration with the session manager
	
	response := map[string]interface{}{
		"sessions": []interface{}{}, // Placeholder
		"count": 0,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleKickSession handles kicking specific sessions
func (ah *AdminHandler) handleKickSession(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var request struct {
		SessionID string `json:"session_id"`
		Reason    string `json:"reason,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	
	if request.SessionID == "" {
		http.Error(w, "session_id is required", http.StatusBadRequest)
		return
	}
	
	ah.logger.Warn("Admin kick session request: %s (reason: %s)", request.SessionID, request.Reason)
	
	// TODO: Implement session kicking
	// This would require integration with the session manager
	
	response := map[string]interface{}{
		"status": "session_kicked",
		"session_id": request.SessionID,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLogHistory handles historical log requests
func (ah *AdminHandler) handleLogHistory(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse query parameters
	filter := ah.parseLogFilter(r)
	
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 1000 {
			limit = parsed
		}
	}
	
	// Get historical logs
	logs := ah.logBroadcaster.GetHistoricalLogs(filter, limit)
	
	response := map[string]interface{}{
		"logs": logs,
		"count": len(logs),
		"filter": filter,
		"timestamp": time.Now(),
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleLogStats handles log statistics requests
func (ah *AdminHandler) handleLogStats(w http.ResponseWriter, r *http.Request) {
	ah.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := ah.logBroadcaster.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// setCORSHeaders sets CORS headers for cross-origin requests
func (ah *AdminHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

// GetShutdownChannel returns the shutdown channel for graceful shutdown
func (ah *AdminHandler) GetShutdownChannel() <-chan bool {
	return ah.shutdownChan
}

// GetRestartChannel returns the restart channel for graceful restart
func (ah *AdminHandler) GetRestartChannel() <-chan bool {
	return ah.restartChan
}