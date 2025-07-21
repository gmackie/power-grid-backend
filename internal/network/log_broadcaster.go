package network

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"powergrid/pkg/logger"
)

// LogLevel represents different log levels for streaming
type LogLevel string

const (
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelError LogLevel = "ERROR"
	LogLevelFatal LogLevel = "FATAL"
)

// LogEntry represents a single log entry for streaming
type LogEntry struct {
	Timestamp   time.Time         `json:"timestamp"`
	Level       LogLevel          `json:"level"`
	Component   string            `json:"component"`
	Message     string            `json:"message"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	PlayerID    string            `json:"player_id,omitempty"`
	GameID      string            `json:"game_id,omitempty"`
	CallSite    string            `json:"call_site,omitempty"`
}

// LogFilter defines criteria for filtering logs
type LogFilter struct {
	MinLevel    LogLevel          `json:"min_level"`
	Components  []string          `json:"components,omitempty"`
	SessionID   string            `json:"session_id,omitempty"`
	PlayerID    string            `json:"player_id,omitempty"`
	GameID      string            `json:"game_id,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
}

// LogClient represents a WebSocket client subscribed to logs
type LogClient struct {
	conn      *websocket.Conn
	filter    LogFilter
	buffer    chan LogEntry
	done      chan bool
	clientID  string
	lastPing  time.Time
}

// LogBroadcaster manages log streaming to WebSocket clients
type LogBroadcaster struct {
	clients    map[string]*LogClient
	clientsMu  sync.RWMutex
	logBuffer  []LogEntry
	bufferMu   sync.RWMutex
	maxBuffer  int
	upgrader   websocket.Upgrader
	logger     *logger.ColoredLogger
}

// NewLogBroadcaster creates a new log broadcaster
func NewLogBroadcaster(maxBuffer int) *LogBroadcaster {
	return &LogBroadcaster{
		clients:   make(map[string]*LogClient),
		logBuffer: make([]LogEntry, 0, maxBuffer),
		maxBuffer: maxBuffer,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in development
			},
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
		},
		logger: logger.NewColoredLogger("LOG_STREAM", logger.ColorBrightCyan),
	}
}

// AddLogEntry adds a new log entry to the broadcaster
func (lb *LogBroadcaster) AddLogEntry(entryData interface{}) {
	// Convert interface{} to LogEntry
	var entry LogEntry
	
	if entryMap, ok := entryData.(map[string]interface{}); ok {
		// Convert from map format
		entry.Timestamp, _ = entryMap["timestamp"].(time.Time)
		entry.Level = LogLevel(entryMap["level"].(string))
		entry.Component, _ = entryMap["component"].(string)
		entry.Message, _ = entryMap["message"].(string)
		entry.SessionID, _ = entryMap["session_id"].(string)
		entry.PlayerID, _ = entryMap["player_id"].(string)
		entry.GameID, _ = entryMap["game_id"].(string)
		entry.CallSite, _ = entryMap["call_site"].(string)
		
		if metadata, ok := entryMap["metadata"].(map[string]string); ok {
			entry.Metadata = metadata
		}
	} else if logEntry, ok := entryData.(LogEntry); ok {
		// Already a LogEntry
		entry = logEntry
	} else {
		// Fallback: create a basic entry
		entry = LogEntry{
			Timestamp: time.Now(),
			Level:     LogLevelInfo,
			Component: "UNKNOWN",
			Message:   fmt.Sprintf("%v", entryData),
		}
	}
	lb.bufferMu.Lock()
	
	// Add to buffer
	lb.logBuffer = append(lb.logBuffer, entry)
	
	// Trim buffer if too large
	if len(lb.logBuffer) > lb.maxBuffer {
		lb.logBuffer = lb.logBuffer[len(lb.logBuffer)-lb.maxBuffer:]
	}
	
	lb.bufferMu.Unlock()
	
	// Broadcast to clients
	lb.broadcastToClients(entry)
}

// AddClient adds a new WebSocket client
func (lb *LogBroadcaster) AddClient(conn *websocket.Conn, clientID string, filter LogFilter) {
	client := &LogClient{
		conn:     conn,
		filter:   filter,
		buffer:   make(chan LogEntry, 100),
		done:     make(chan bool),
		clientID: clientID,
		lastPing: time.Now(),
	}
	
	lb.clientsMu.Lock()
	lb.clients[clientID] = client
	lb.clientsMu.Unlock()
	
	lb.logger.Info("Log client connected: %s", clientID)
	
	// Send historical logs if requested
	go lb.sendHistoricalLogs(client)
	
	// Start client handler
	go lb.handleClient(client)
}

// RemoveClient removes a WebSocket client
func (lb *LogBroadcaster) RemoveClient(clientID string) {
	lb.clientsMu.Lock()
	defer lb.clientsMu.Unlock()
	
	if client, exists := lb.clients[clientID]; exists {
		close(client.done)
		client.conn.Close()
		delete(lb.clients, clientID)
		lb.logger.Info("Log client disconnected: %s", clientID)
	}
}

// UpdateClientFilter updates the filter for a specific client
func (lb *LogBroadcaster) UpdateClientFilter(clientID string, filter LogFilter) {
	lb.clientsMu.RLock()
	defer lb.clientsMu.RUnlock()
	
	if client, exists := lb.clients[clientID]; exists {
		client.filter = filter
		lb.logger.Debug("Updated filter for client: %s", clientID)
	}
}

// GetHistoricalLogs returns historical logs matching the filter
func (lb *LogBroadcaster) GetHistoricalLogs(filter LogFilter, limit int) []LogEntry {
	lb.bufferMu.RLock()
	defer lb.bufferMu.RUnlock()
	
	var filtered []LogEntry
	
	// Apply filter to historical logs
	for i := len(lb.logBuffer) - 1; i >= 0 && len(filtered) < limit; i-- {
		entry := lb.logBuffer[i]
		if lb.matchesFilter(entry, filter) {
			filtered = append([]LogEntry{entry}, filtered...) // Prepend to maintain order
		}
	}
	
	return filtered
}

// broadcastToClients sends a log entry to all matching clients
func (lb *LogBroadcaster) broadcastToClients(entry LogEntry) {
	lb.clientsMu.RLock()
	defer lb.clientsMu.RUnlock()
	
	for _, client := range lb.clients {
		if lb.matchesFilter(entry, client.filter) {
			select {
			case client.buffer <- entry:
			default:
				// Buffer full, skip this entry
				lb.logger.Warn("Log buffer full for client: %s", client.clientID)
			}
		}
	}
}

// sendHistoricalLogs sends historical logs to a new client
func (lb *LogBroadcaster) sendHistoricalLogs(client *LogClient) {
	historicalLogs := lb.GetHistoricalLogs(client.filter, 100) // Last 100 matching logs
	
	for _, entry := range historicalLogs {
		select {
		case client.buffer <- entry:
		case <-client.done:
			return
		}
	}
}

// handleClient manages a WebSocket client connection
func (lb *LogBroadcaster) handleClient(client *LogClient) {
	ticker := time.NewTicker(30 * time.Second) // Ping every 30 seconds
	defer ticker.Stop()
	
	for {
		select {
		case entry := <-client.buffer:
			if err := client.conn.WriteJSON(entry); err != nil {
				lb.logger.Error("Failed to send log to client %s: %v", client.clientID, err)
				lb.RemoveClient(client.clientID)
				return
			}
			
		case <-ticker.C:
			// Send ping
			if err := client.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				lb.logger.Error("Failed to ping client %s: %v", client.clientID, err)
				lb.RemoveClient(client.clientID)
				return
			}
			client.lastPing = time.Now()
			
		case <-client.done:
			return
		}
	}
}

// matchesFilter checks if a log entry matches the client's filter
func (lb *LogBroadcaster) matchesFilter(entry LogEntry, filter LogFilter) bool {
	// Check log level
	if !lb.levelMatches(entry.Level, filter.MinLevel) {
		return false
	}
	
	// Check components
	if len(filter.Components) > 0 {
		found := false
		for _, component := range filter.Components {
			if component == entry.Component {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check session ID
	if filter.SessionID != "" && filter.SessionID != entry.SessionID {
		return false
	}
	
	// Check player ID
	if filter.PlayerID != "" && filter.PlayerID != entry.PlayerID {
		return false
	}
	
	// Check game ID
	if filter.GameID != "" && filter.GameID != entry.GameID {
		return false
	}
	
	// Check keywords
	if len(filter.Keywords) > 0 {
		found := false
		for _, keyword := range filter.Keywords {
			if contains(entry.Message, keyword) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

// levelMatches checks if the entry level meets the minimum filter level
func (lb *LogBroadcaster) levelMatches(entryLevel, minLevel LogLevel) bool {
	levels := []LogLevel{LogLevelDebug, LogLevelInfo, LogLevelWarn, LogLevelError, LogLevelFatal}
	
	entryIndex := -1
	minIndex := -1
	
	for i, level := range levels {
		if level == entryLevel {
			entryIndex = i
		}
		if level == minLevel {
			minIndex = i
		}
	}
	
	return entryIndex >= minIndex
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     fmt.Sprintf("%s", s)[0:len(substr)] == substr))
}

// GetClientCount returns the number of connected clients
func (lb *LogBroadcaster) GetClientCount() int {
	lb.clientsMu.RLock()
	defer lb.clientsMu.RUnlock()
	return len(lb.clients)
}

// GetStats returns statistics about the log broadcaster
func (lb *LogBroadcaster) GetStats() map[string]interface{} {
	lb.clientsMu.RLock()
	clientCount := len(lb.clients)
	lb.clientsMu.RUnlock()
	
	lb.bufferMu.RLock()
	bufferSize := len(lb.logBuffer)
	lb.bufferMu.RUnlock()
	
	return map[string]interface{}{
		"connected_clients": clientCount,
		"buffer_size":      bufferSize,
		"max_buffer":       lb.maxBuffer,
	}
}