package logger

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

// LogBroadcaster interface for streaming logs
type LogBroadcaster interface {
	AddLogEntry(entry interface{})
}

// StreamingLogger extends ColoredLogger with streaming capabilities
type StreamingLogger struct {
	*ColoredLogger
	broadcaster LogBroadcaster
	sessionID   string
	playerID    string
	gameID      string
	metadata    map[string]string
}

// NewStreamingLogger creates a new streaming logger
func NewStreamingLogger(context, color string, broadcaster LogBroadcaster) *StreamingLogger {
	return &StreamingLogger{
		ColoredLogger: NewColoredLogger(context, color),
		broadcaster:   broadcaster,
		metadata:      make(map[string]string),
	}
}

// SetSessionID sets the session ID for all log entries
func (sl *StreamingLogger) SetSessionID(sessionID string) {
	sl.sessionID = sessionID
}

// SetPlayerID sets the player ID for all log entries
func (sl *StreamingLogger) SetPlayerID(playerID string) {
	sl.playerID = playerID
}

// SetGameID sets the game ID for all log entries
func (sl *StreamingLogger) SetGameID(gameID string) {
	sl.gameID = gameID
}

// SetMetadata adds metadata to all log entries
func (sl *StreamingLogger) SetMetadata(key, value string) {
	sl.metadata[key] = value
}

// ClearMetadata removes all metadata
func (sl *StreamingLogger) ClearMetadata() {
	sl.metadata = make(map[string]string)
}

// getCallSite returns the caller information
func (sl *StreamingLogger) getCallSite() string {
	if pc, file, line, ok := runtime.Caller(4); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			parts := strings.Split(file, "/")
			fileName := parts[len(parts)-1]
			return fmt.Sprintf("%s:%d", fileName, line)
		}
	}
	return ""
}

// streamLog creates and sends a log entry to the broadcaster
func (sl *StreamingLogger) streamLog(level LogLevel, format string, args ...interface{}) {
	if sl.broadcaster == nil {
		return
	}
	
	message := fmt.Sprintf(format, args...)
	
	// Copy metadata to avoid race conditions
	metadata := make(map[string]string)
	for k, v := range sl.metadata {
		metadata[k] = v
	}
	
	// Create entry compatible with network.LogEntry
	entry := map[string]interface{}{
		"timestamp": time.Now(),
		"level":     level.String(),
		"component": sl.context,
		"message":   message,
		"metadata":  metadata,
		"session_id": sl.sessionID,
		"player_id":  sl.playerID,
		"game_id":    sl.gameID,
		"call_site":  sl.getCallSite(),
	}
	
	sl.broadcaster.AddLogEntry(entry)
}

// Override all logging methods to include streaming
func (sl *StreamingLogger) Debug(format string, args ...interface{}) {
	sl.ColoredLogger.Debug(format, args...)
	sl.streamLog(DEBUG, format, args...)
}

func (sl *StreamingLogger) Info(format string, args ...interface{}) {
	sl.ColoredLogger.Info(format, args...)
	sl.streamLog(INFO, format, args...)
}

func (sl *StreamingLogger) Warn(format string, args ...interface{}) {
	sl.ColoredLogger.Warn(format, args...)
	sl.streamLog(WARN, format, args...)
}

func (sl *StreamingLogger) Error(format string, args ...interface{}) {
	sl.ColoredLogger.Error(format, args...)
	sl.streamLog(ERROR, format, args...)
}

func (sl *StreamingLogger) Fatal(format string, args ...interface{}) {
	sl.ColoredLogger.Fatal(format, args...)
	sl.streamLog(FATAL, format, args...)
}

// Enhanced logging methods with context
func (sl *StreamingLogger) DebugWithContext(sessionID, playerID, gameID string, format string, args ...interface{}) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	sl.Debug(format, args...)
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

func (sl *StreamingLogger) InfoWithContext(sessionID, playerID, gameID string, format string, args ...interface{}) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	sl.Info(format, args...)
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

func (sl *StreamingLogger) WarnWithContext(sessionID, playerID, gameID string, format string, args ...interface{}) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	sl.Warn(format, args...)
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

func (sl *StreamingLogger) ErrorWithContext(sessionID, playerID, gameID string, format string, args ...interface{}) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	sl.Error(format, args...)
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

// Game event logging methods
func (sl *StreamingLogger) LogGameEvent(gameID, event string, metadata map[string]string) {
	sl.SetGameID(gameID)
	for k, v := range metadata {
		sl.SetMetadata(k, v)
	}
	sl.Info("Game event: %s", event)
	sl.ClearMetadata()
}

func (sl *StreamingLogger) LogPlayerAction(sessionID, playerID, gameID, action string, metadata map[string]string) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	for k, v := range metadata {
		sl.SetMetadata(k, v)
	}
	sl.Info("Player action: %s", action)
	sl.ClearMetadata()
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

func (sl *StreamingLogger) LogAIDecision(sessionID, playerID, gameID, strategy, decision string, metadata map[string]string) {
	oldSession, oldPlayer, oldGame := sl.sessionID, sl.playerID, sl.gameID
	sl.sessionID, sl.playerID, sl.gameID = sessionID, playerID, gameID
	
	sl.SetMetadata("strategy", strategy)
	for k, v := range metadata {
		sl.SetMetadata(k, v)
	}
	sl.Info("AI decision: %s", decision)
	sl.ClearMetadata()
	
	sl.sessionID, sl.playerID, sl.gameID = oldSession, oldPlayer, oldGame
}

func (sl *StreamingLogger) LogSimulationEvent(event string, metadata map[string]string) {
	for k, v := range metadata {
		sl.SetMetadata(k, v)
	}
	sl.Info("Simulation event: %s", event)
	sl.ClearMetadata()
}

// Global streaming loggers
var (
	globalBroadcaster LogBroadcaster
	
	StreamingServerLogger *StreamingLogger
	StreamingClientLogger *StreamingLogger
	StreamingGameLogger   *StreamingLogger
	StreamingAILogger     *StreamingLogger
	StreamingTestLogger   *StreamingLogger
)

// InitStreamingLoggers initializes all streaming loggers with a broadcaster
func InitStreamingLoggers(broadcaster LogBroadcaster, level LogLevel, showCaller bool) {
	globalBroadcaster = broadcaster
	
	StreamingServerLogger = NewStreamingLogger("SERVER", ColorBrightGreen, broadcaster)
	StreamingClientLogger = NewStreamingLogger("CLIENT", ColorBrightBlue, broadcaster)
	StreamingGameLogger = NewStreamingLogger("GAME", ColorBrightPurple, broadcaster)
	StreamingAILogger = NewStreamingLogger("AI", ColorBrightCyan, broadcaster)
	StreamingTestLogger = NewStreamingLogger("TEST", ColorBrightYellow, broadcaster)
	
	loggers := []*StreamingLogger{
		StreamingServerLogger,
		StreamingClientLogger,
		StreamingGameLogger,
		StreamingAILogger,
		StreamingTestLogger,
	}
	
	for _, logger := range loggers {
		logger.SetLevel(level)
		logger.SetShowCaller(showCaller)
	}
}

// CreateStreamingPlayerLogger creates a streaming logger for a specific player
func CreateStreamingPlayerLogger(playerName, color string) *StreamingLogger {
	logger := NewStreamingLogger(fmt.Sprintf("PLAYER:%s", playerName), color, globalBroadcaster)
	return logger
}

// CreateStreamingAILogger creates a streaming logger for a specific AI strategy
func CreateStreamingAILogger(strategy, color string) *StreamingLogger {
	logger := NewStreamingLogger(fmt.Sprintf("AI:%s", strategy), color, globalBroadcaster)
	return logger
}