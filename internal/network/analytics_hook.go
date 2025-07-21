package network

import (
	"powergrid/internal/analytics"
	"powergrid/internal/game"
	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// AnalyticsHook integrates analytics with game events
type AnalyticsHook struct {
	service *analytics.Service
	logger  *logger.ColoredLogger
}

// NewAnalyticsHook creates a new analytics hook
func NewAnalyticsHook(service *analytics.Service) *AnalyticsHook {
	return &AnalyticsHook{
		service: service,
		logger:  logger.GameLogger,
	}
}

// SetupGameAnalytics configures a game to send analytics events
func (h *AnalyticsHook) SetupGameAnalytics(game *game.Game) {
	// Set the analytics handler on the game
	game.SetAnalyticsHandler(func(event string, data interface{}) {
		h.handleGameEvent(game.ID, event, data)
	})
}

// OnGameStart is called when a game starts
func (h *AnalyticsHook) OnGameStart(gameID, gameName, mapName string, players []string) {
	h.service.TrackGameStart(gameID, gameName, mapName, players)
	h.logger.Info("Analytics: Started tracking game %s", gameID)
}

// OnGameStateUpdate is called when game state changes
func (h *AnalyticsHook) OnGameStateUpdate(gameID string, state *protocol.GameStatePayload) {
	h.service.TrackGameState(gameID, state)
}

// OnGameEnd is called when a game ends
func (h *AnalyticsHook) OnGameEnd(gameID string, winner string, finalState *protocol.GameStatePayload) {
	h.service.TrackGameEnd(gameID, winner, finalState)
	h.logger.Info("Analytics: Recorded game end for %s, winner: %s", gameID, winner)
}

// handleGameEvent processes individual game events
func (h *AnalyticsHook) handleGameEvent(gameID, event string, data interface{}) {
	// Log specific events that might be useful for analytics
	switch event {
	case "player_joined":
		h.logger.Debug("Analytics: Player joined game %s", gameID)
	case "phase_changed":
		h.logger.Debug("Analytics: Phase changed in game %s", gameID)
	case "auction_completed":
		h.logger.Debug("Analytics: Auction completed in game %s", gameID)
	case "resources_purchased":
		h.logger.Debug("Analytics: Resources purchased in game %s", gameID)
	case "city_built":
		h.logger.Debug("Analytics: City built in game %s", gameID)
	case "cities_powered":
		h.logger.Debug("Analytics: Cities powered in game %s", gameID)
	}
}