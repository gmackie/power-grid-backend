package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"powergrid/internal/analytics"
	"powergrid/pkg/logger"
)

// DatabaseAnalyticsHandler handles analytics-related HTTP requests using SQLite database
type DatabaseAnalyticsHandler struct {
	service *analytics.DatabaseService
	logger  *logger.ColoredLogger
}

// NewDatabaseAnalyticsHandler creates a new database analytics handler
func NewDatabaseAnalyticsHandler(service *analytics.DatabaseService) *DatabaseAnalyticsHandler {
	return &DatabaseAnalyticsHandler{
		service: service,
		logger:  logger.CreateAILogger("AnalyticsAPI", logger.ColorCyan),
	}
}

// RegisterRoutes registers all analytics routes
func (h *DatabaseAnalyticsHandler) RegisterRoutes(mux *http.ServeMux) {
	// Player endpoints
	mux.HandleFunc("/api/players/", h.handlePlayerRequest)
	mux.HandleFunc("/api/players", h.handlePlayersRequest)
	
	// Achievement endpoints
	mux.HandleFunc("/api/achievements", h.handleAchievements)
	
	// Leaderboard endpoints
	mux.HandleFunc("/api/leaderboard", h.handleLeaderboard)
	mux.HandleFunc("/api/leaderboard/achievements", h.handleAchievementLeaderboard)
	
	// Game analytics
	mux.HandleFunc("/api/analytics/games", h.handleGameAnalytics)
	mux.HandleFunc("/api/analytics/achievements", h.handleAchievementAnalytics)
	mux.HandleFunc("/api/analytics/advanced", h.handleAdvancedAnalytics)
	mux.HandleFunc("/api/analytics/activity", h.handleActivityReport)
	mux.HandleFunc("/api/analytics/player-types", h.handlePlayerTypeDistribution)
	mux.HandleFunc("/api/analytics/maps", h.handleMapAnalytics)
	
	// Health check for API
	mux.HandleFunc("/api/health", h.handleHealth)
	
	h.logger.Info("Analytics API routes registered")
}

// Player endpoints

func (h *DatabaseAnalyticsHandler) handlePlayerRequest(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Extract player name from URL path
	path := strings.TrimPrefix(r.URL.Path, "/api/players/")
	if path == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Player name required")
		return
	}
	
	playerName := strings.Split(path, "/")[0]
	
	// Check if this is a stats request
	if strings.HasSuffix(r.URL.Path, "/stats") {
		h.handlePlayerStats(w, r, playerName)
		return
	}
	
	// Check if this is an achievements request
	if strings.HasSuffix(r.URL.Path, "/achievements") {
		h.handlePlayerAchievements(w, r, playerName)
		return
	}
	
	// Check if this is a performance metrics request
	if strings.HasSuffix(r.URL.Path, "/performance") {
		h.handlePlayerPerformance(w, r, playerName)
		return
	}
	
	// Check if this is a competitors request
	if strings.HasSuffix(r.URL.Path, "/competitors") {
		h.handlePlayerCompetitors(w, r, playerName)
		return
	}
	
	// Check if this is a progression request
	if strings.HasSuffix(r.URL.Path, "/progression") {
		h.handlePlayerProgression(w, r, playerName)
		return
	}
	
	// Default: return player info and stats
	stats, err := h.service.GetPlayerStats(playerName)
	if err != nil {
		h.logger.Error("Failed to get player stats for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, stats)
}

func (h *DatabaseAnalyticsHandler) handlePlayerStats(w http.ResponseWriter, r *http.Request, playerName string) {
	stats, err := h.service.GetPlayerStats(playerName)
	if err != nil {
		h.logger.Error("Failed to get player stats for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, stats)
}

func (h *DatabaseAnalyticsHandler) handlePlayerAchievements(w http.ResponseWriter, r *http.Request, playerName string) {
	achievements, err := h.service.GetPlayerAchievements(playerName)
	if err != nil {
		h.logger.Error("Failed to get player achievements for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"player_name":   playerName,
		"achievements":  achievements,
		"total_count":   len(achievements),
	})
}

func (h *DatabaseAnalyticsHandler) handlePlayersRequest(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// This endpoint would need to be implemented in the repository
	// For now, return an error indicating it's not implemented
	h.writeErrorResponse(w, http.StatusNotImplemented, "Endpoint not yet implemented")
}

// Achievement endpoints

func (h *DatabaseAnalyticsHandler) handleAchievements(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// This would get all available achievements
	// For now, return placeholder response
	response := map[string]interface{}{
		"message": "Achievement endpoint - not yet fully implemented",
		"status":  "placeholder",
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Leaderboard endpoints

func (h *DatabaseAnalyticsHandler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 50 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	
	leaderboard, err := h.service.GetLeaderboard(limit)
	if err != nil {
		h.logger.Error("Failed to get leaderboard: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve leaderboard")
		return
	}
	
	response := map[string]interface{}{
		"leaderboard": leaderboard,
		"limit":       limit,
		"count":       len(leaderboard),
		"updated_at":  time.Now(),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *DatabaseAnalyticsHandler) handleAchievementLeaderboard(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get limit parameter
	limitStr := r.URL.Query().Get("limit")
	limit := 25 // default for achievement leaderboard
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	
	// This would be implemented in the achievement repository
	response := map[string]interface{}{
		"message": "Achievement leaderboard endpoint - not yet fully implemented",
		"status":  "placeholder",
		"limit":   limit,
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Analytics endpoints

func (h *DatabaseAnalyticsHandler) handleGameAnalytics(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get days parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	analytics, err := h.service.GetGameAnalytics(days)
	if err != nil {
		h.logger.Error("Failed to get game analytics: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve analytics")
		return
	}
	
	response := map[string]interface{}{
		"analytics":   analytics,
		"period_days": days,
		"updated_at":  time.Now(),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

func (h *DatabaseAnalyticsHandler) handleAchievementAnalytics(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	stats, err := h.service.GetAchievementStats()
	if err != nil {
		h.logger.Error("Failed to get achievement analytics: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve achievement analytics")
		return
	}
	
	response := map[string]interface{}{
		"stats":      stats,
		"updated_at": time.Now(),
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Health endpoint

func (h *DatabaseAnalyticsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	response := map[string]interface{}{
		"status":     "healthy",
		"service":    "analytics-database",
		"timestamp":  time.Now(),
		"version":    "1.0.0",
	}
	
	h.writeJSONResponse(w, http.StatusOK, response)
}

// Helper methods

func (h *DatabaseAnalyticsHandler) setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func (h *DatabaseAnalyticsHandler) writeJSONResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response: %v", err)
	}
}

func (h *DatabaseAnalyticsHandler) writeErrorResponse(w http.ResponseWriter, status int, message string) {
	h.writeJSONResponse(w, status, map[string]interface{}{
		"error":     message,
		"status":    status,
		"timestamp": time.Now(),
	})
}

// Advanced player analytics handlers

func (h *DatabaseAnalyticsHandler) handlePlayerPerformance(w http.ResponseWriter, r *http.Request, playerName string) {
	// Get timeframe parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	metrics, err := h.service.GetPlayerPerformanceMetrics(playerName, days)
	if err != nil {
		h.logger.Error("Failed to get player performance metrics for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found or insufficient data")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"player_name":   playerName,
		"metrics":       metrics,
		"time_frame":    days,
		"generated_at":  time.Now(),
	})
}

func (h *DatabaseAnalyticsHandler) handlePlayerCompetitors(w http.ResponseWriter, r *http.Request, playerName string) {
	// Get timeframe parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	competitors, err := h.service.GetCompetitorAnalysis(playerName, days)
	if err != nil {
		h.logger.Error("Failed to get competitor analysis for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found or insufficient data")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"player_name":   playerName,
		"competitors":   competitors,
		"time_frame":    days,
		"generated_at":  time.Now(),
	})
}

func (h *DatabaseAnalyticsHandler) handlePlayerProgression(w http.ResponseWriter, r *http.Request, playerName string) {
	// Get timeframe parameter
	daysStr := r.URL.Query().Get("days")
	days := 90 // default longer period for progression
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	progression, err := h.service.GetPlayerSkillProgression(playerName, days)
	if err != nil {
		h.logger.Error("Failed to get player skill progression for %s: %v", playerName, err)
		h.writeErrorResponse(w, http.StatusNotFound, "Player not found or insufficient data")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"player_name":   playerName,
		"progression":   progression,
		"time_frame":    days,
		"generated_at":  time.Now(),
	})
}

// Advanced analytics endpoints

func (h *DatabaseAnalyticsHandler) handleAdvancedAnalytics(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get days parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	analytics, err := h.service.GetAdvancedGameAnalytics(days)
	if err != nil {
		h.logger.Error("Failed to get advanced game analytics: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve advanced analytics")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"analytics":    analytics,
		"time_frame":   days,
		"generated_at": time.Now(),
	})
}

func (h *DatabaseAnalyticsHandler) handleActivityReport(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get days parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	report, err := h.service.GetActivityReport(days)
	if err != nil {
		h.logger.Error("Failed to get activity report: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve activity report")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, report)
}

func (h *DatabaseAnalyticsHandler) handlePlayerTypeDistribution(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	distribution, err := h.service.GetPlayerTypeDistribution()
	if err != nil {
		h.logger.Error("Failed to get player type distribution: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve player type distribution")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"distribution": distribution,
		"total_players": func() int {
			total := 0
			for _, count := range distribution {
				total += count
			}
			return total
		}(),
		"generated_at": time.Now(),
	})
}

func (h *DatabaseAnalyticsHandler) handleMapAnalytics(w http.ResponseWriter, r *http.Request) {
	h.setCORSHeaders(w)
	
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}
	
	if r.Method != http.MethodGet {
		h.writeErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	
	// Get days parameter
	daysStr := r.URL.Query().Get("days")
	days := 30 // default
	if daysStr != "" {
		if parsed, err := strconv.Atoi(daysStr); err == nil && parsed > 0 && parsed <= 365 {
			days = parsed
		}
	}
	
	mapAnalytics, err := h.service.GetMapAnalytics(days)
	if err != nil {
		h.logger.Error("Failed to get map analytics: %v", err)
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve map analytics")
		return
	}
	
	h.writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"map_analytics": mapAnalytics,
		"time_frame":    days,
		"generated_at":  time.Now(),
	})
}