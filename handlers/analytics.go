package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"powergrid/internal/analytics"
	"powergrid/models"
	"powergrid/pkg/logger"
)

// AnalyticsHandler handles analytics-related HTTP requests
type AnalyticsHandler struct {
	service *analytics.Service
	logger  *logger.ColoredLogger
}

// NewAnalyticsHandler creates a new analytics handler
func NewAnalyticsHandler(service *analytics.Service) *AnalyticsHandler {
	return &AnalyticsHandler{
		service: service,
		logger:  logger.CreateAILogger("API", logger.ColorBrightYellow),
	}
}

// RegisterRoutes registers all analytics routes
func (h *AnalyticsHandler) RegisterRoutes(mux *http.ServeMux) {
	// Player endpoints
	mux.HandleFunc("/api/players/", h.handlePlayerRequest)
	mux.HandleFunc("/api/players", h.handlePlayersRequest)
	
	// Achievement endpoints
	mux.HandleFunc("/api/achievements", h.handleAchievements)
	
	// Leaderboard endpoint
	mux.HandleFunc("/api/leaderboard", h.handleLeaderboard)
	
	// Game analytics
	mux.HandleFunc("/api/analytics/games", h.handleGameAnalytics)
	
	// Health check for API
	mux.HandleFunc("/api/health", h.handleHealth)
}

// Player endpoints

func (h *AnalyticsHandler) handlePlayerRequest(w http.ResponseWriter, r *http.Request) {
	// Extract player name from path
	path := strings.TrimPrefix(r.URL.Path, "/api/players/")
	parts := strings.Split(path, "/")
	
	if len(parts) == 0 || parts[0] == "" {
		h.sendError(w, "Player name required", http.StatusBadRequest)
		return
	}
	
	playerName := parts[0]
	
	// Route based on remaining path
	if len(parts) == 1 {
		// GET /api/players/{name} - Get player stats
		if r.Method != http.MethodGet {
			h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		h.getPlayerStats(w, r, playerName)
	} else if len(parts) == 2 {
		switch parts[1] {
		case "achievements":
			// GET /api/players/{name}/achievements
			if r.Method != http.MethodGet {
				h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.getPlayerAchievements(w, r, playerName)
			
		case "history":
			// GET /api/players/{name}/history
			if r.Method != http.MethodGet {
				h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.getPlayerHistory(w, r, playerName)
			
		case "progress":
			// GET /api/players/{name}/progress
			if r.Method != http.MethodGet {
				h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			h.getPlayerProgress(w, r, playerName)
			
		default:
			h.sendError(w, "Not found", http.StatusNotFound)
		}
	} else {
		h.sendError(w, "Not found", http.StatusNotFound)
	}
}

func (h *AnalyticsHandler) handlePlayersRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// GET /api/players - List all players
	h.listPlayers(w, r)
}

func (h *AnalyticsHandler) getPlayerStats(w http.ResponseWriter, r *http.Request, playerName string) {
	stats, err := h.service.GetPlayerStats(playerName)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusNotFound)
		return
	}
	
	h.logger.Info("Retrieved stats for player: %s", playerName)
	h.sendJSON(w, stats)
}

func (h *AnalyticsHandler) getPlayerAchievements(w http.ResponseWriter, r *http.Request, playerName string) {
	achievements, err := h.service.GetPlayerAchievements(playerName)
	if err != nil {
		h.sendError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Calculate achievement summary
	summary := struct {
		PlayerName         string                      `json:"player_name"`
		TotalAchievements  int                         `json:"total_achievements"`
		TotalPoints        int                         `json:"total_points"`
		Achievements       []models.PlayerAchievement  `json:"achievements"`
		RecentAchievements []models.PlayerAchievement  `json:"recent_achievements"`
		CategoryBreakdown  map[string]int              `json:"category_breakdown"`
	}{
		PlayerName:         playerName,
		TotalAchievements:  len(achievements),
		Achievements:       achievements,
		CategoryBreakdown:  make(map[string]int),
	}
	
	// Calculate points and categories
	for _, ach := range achievements {
		summary.TotalPoints += ach.Achievement.Points
		summary.CategoryBreakdown[ach.Achievement.Category]++
	}
	
	// Get recent achievements (last 5)
	if len(achievements) > 5 {
		summary.RecentAchievements = achievements[len(achievements)-5:]
	} else {
		summary.RecentAchievements = achievements
	}
	
	h.logger.Info("Retrieved %d achievements for player: %s", len(achievements), playerName)
	h.sendJSON(w, summary)
}

func (h *AnalyticsHandler) getPlayerHistory(w http.ResponseWriter, r *http.Request, playerName string) {
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 20 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	history := h.service.GetGameHistory(playerName, limit)
	
	response := struct {
		PlayerName  string              `json:"player_name"`
		GameCount   int                 `json:"game_count"`
		Games       []models.GameRecord `json:"games"`
	}{
		PlayerName: playerName,
		GameCount:  len(history),
		Games:      history,
	}
	
	h.logger.Info("Retrieved %d games for player: %s", len(history), playerName)
	h.sendJSON(w, response)
}

func (h *AnalyticsHandler) getPlayerProgress(w http.ResponseWriter, r *http.Request, playerName string) {
	// This would show player improvement over time
	// For now, return a simple progress report
	stats, _ := h.service.GetPlayerStats(playerName)
	history := h.service.GetGameHistory(playerName, 50)
	
	response := models.PlayerProgressResponse{
		PlayerName: playerName,
		Progress:   make([]models.ProgressDataPoint, 0),
		Milestones: make([]models.Milestone, 0),
	}
	
	// Calculate progress over recent games
	if len(history) > 0 {
		// Group by week
		weeklyStats := make(map[string]*models.ProgressDataPoint)
		
		for _, game := range history {
			week := game.StartTime.Format("2006-01-02")
			if _, exists := weeklyStats[week]; !exists {
				weeklyStats[week] = &models.ProgressDataPoint{
					Date: game.StartTime,
				}
			}
			
			// Update weekly stats
			point := weeklyStats[week]
			point.GameCount++
			
			for _, player := range game.Players {
				if player.PlayerName == playerName {
					if player.IsWinner {
						point.WinRate = (point.WinRate*float64(point.GameCount-1) + 100) / float64(point.GameCount)
					} else {
						point.WinRate = (point.WinRate * float64(point.GameCount-1)) / float64(point.GameCount)
					}
					point.AvgCities = (point.AvgCities*float64(point.GameCount-1) + float64(player.FinalCities)) / float64(point.GameCount)
					break
				}
			}
		}
		
		// Convert to slice
		for _, point := range weeklyStats {
			response.Progress = append(response.Progress, *point)
		}
	}
	
	// Add milestones
	if stats != nil {
		if stats.GamesWon == 1 {
			response.Milestones = append(response.Milestones, models.Milestone{
				Name:        "First Victory",
				Description: "Won your first game",
				AchievedAt:  stats.CreatedAt, // Approximate
			})
		}
		if stats.GamesPlayed == 10 {
			response.Milestones = append(response.Milestones, models.Milestone{
				Name:        "10 Games Played",
				Description: "Reached 10 games milestone",
				AchievedAt:  stats.UpdatedAt,
			})
		}
	}
	
	h.sendJSON(w, response)
}

func (h *AnalyticsHandler) listPlayers(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, this would query from the service
	// For now, return a simple message
	response := struct {
		Message string `json:"message"`
		Info    string `json:"info"`
	}{
		Message: "Player list endpoint",
		Info:    "Use /api/leaderboard for top players or /api/players/{name} for specific player stats",
	}
	
	h.sendJSON(w, response)
}

// Achievement endpoints

func (h *AnalyticsHandler) handleAchievements(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Return all available achievements
	h.sendJSON(w, models.PredefinedAchievements)
}

// Leaderboard endpoint

func (h *AnalyticsHandler) handleLeaderboard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse query parameters
	limitStr := r.URL.Query().Get("limit")
	limit := 10 // default
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	leaderboard := h.service.GetLeaderboard(limit)
	
	response := struct {
		Timestamp   time.Time                 `json:"timestamp"`
		PlayerCount int                       `json:"player_count"`
		Leaderboard []models.LeaderboardEntry `json:"leaderboard"`
	}{
		Timestamp:   time.Now(),
		PlayerCount: len(leaderboard),
		Leaderboard: leaderboard,
	}
	
	h.logger.Info("Retrieved leaderboard with %d entries", len(leaderboard))
	h.sendJSON(w, response)
}

// Game analytics endpoint

func (h *AnalyticsHandler) handleGameAnalytics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.sendError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse time range from query
	var timeRange *models.TimeRange
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	if startStr != "" && endStr != "" {
		start, err1 := time.Parse(time.RFC3339, startStr)
		end, err2 := time.Parse(time.RFC3339, endStr)
		if err1 == nil && err2 == nil {
			timeRange = &models.TimeRange{
				Start: start,
				End:   end,
			}
		}
	}
	
	analytics := h.service.GetGameAnalytics(timeRange)
	
	h.logger.Info("Retrieved game analytics: %d total games", analytics.TotalGames)
	h.sendJSON(w, analytics)
}

// Health check

func (h *AnalyticsHandler) handleHealth(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Status    string    `json:"status"`
		Service   string    `json:"service"`
		Timestamp time.Time `json:"timestamp"`
	}{
		Status:    "healthy",
		Service:   "analytics-api",
		Timestamp: time.Now(),
	}
	
	h.sendJSON(w, response)
}

// Helper methods

func (h *AnalyticsHandler) sendJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	if err := json.NewEncoder(w).Encode(data); err != nil {
		h.logger.Error("Failed to encode JSON response: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *AnalyticsHandler) sendError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteHeader(code)
	
	response := struct {
		Error   string    `json:"error"`
		Message string    `json:"message"`
		Code    int       `json:"code"`
		Time    time.Time `json:"time"`
	}{
		Error:   http.StatusText(code),
		Message: message,
		Code:    code,
		Time:    time.Now(),
	}
	
	json.NewEncoder(w).Encode(response)
}