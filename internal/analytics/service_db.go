package analytics

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	"powergrid/internal/analytics/models"
	"powergrid/internal/database"
	"powergrid/internal/database/repositories"
	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// DatabaseService manages analytics using SQLite database
type DatabaseService struct {
	mu         sync.RWMutex
	db         *database.DB
	repo       *database.Repository
	logger     *logger.ColoredLogger
	
	// Active game tracking (still in-memory for performance)
	activeGames map[string]*GameTracker
}

// NewDatabaseService creates a new database-backed analytics service
func NewDatabaseService(db *database.DB) *DatabaseService {
	return &DatabaseService{
		db:          db,
		repo:        database.NewRepository(db),
		logger:      logger.CreateAILogger("AnalyticsDB", logger.ColorGreen),
		activeGames: make(map[string]*GameTracker),
	}
}

// StartGameTracking begins tracking a new game
func (s *DatabaseService) StartGameTracking(gameID, gameName, mapName string, playerNames []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.logger.Info("Starting game tracking: %s", gameID)
	
	// Create game record in database
	_, err := s.repo.Game.CreateGame(gameID, gameName, mapName, 6, len(playerNames))
	if err != nil {
		return fmt.Errorf("failed to create game record: %w", err)
	}
	
	// Create or update players and add them as participants
	for i, playerName := range playerNames {
		player, err := s.repo.Player.CreateOrUpdatePlayer(playerName)
		if err != nil {
			s.logger.Error("Failed to create/update player %s: %v", playerName, err)
			continue
		}
		
		// Add as game participant
		err = s.repo.Game.AddGameParticipant(gameID, player.ID, playerName, "", i+1)
		if err != nil {
			s.logger.Error("Failed to add participant %s: %v", playerName, err)
		}
	}
	
	// Create in-memory tracker for active game
	tracker := &GameTracker{
		GameID:     gameID,
		GameName:   gameName,
		MapName:    mapName,
		StartTime:  time.Now(),
		Players:    make(map[string]*PlayerTracker),
		LastUpdate: time.Now(),
	}
	
	for _, playerName := range playerNames {
		tracker.Players[playerName] = &PlayerTracker{
			PlayerName: playerName,
		}
	}
	
	s.activeGames[gameID] = tracker
	
	// Update game status to playing
	err = s.repo.Game.UpdateGameStatus(gameID, "playing", nil)
	if err != nil {
		s.logger.Error("Failed to update game status: %v", err)
	}
	
	s.logger.Debug("Game tracking started for %s with %d players", gameID, len(playerNames))
	return nil
}

// UpdateGameState updates the state of an active game
func (s *DatabaseService) UpdateGameState(gameID string, state *protocol.GameStatePayload) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tracker, exists := s.activeGames[gameID]
	if !exists {
		return fmt.Errorf("game not being tracked: %s", gameID)
	}
	
	tracker.LastUpdate = time.Now()
	tracker.RoundCount = state.CurrentRound
	
	// Update player trackers with current state
	for _, player := range state.Players {
		if playerTracker, exists := tracker.Players[player.Name]; exists {
			playerTracker.MaxCities = max(playerTracker.MaxCities, len(player.Cities))
			playerTracker.MaxPlants = max(playerTracker.MaxPlants, len(player.PowerPlants))
			playerTracker.MaxMoney = max(playerTracker.MaxMoney, player.Money)
			
			// Calculate total resources
			totalResources := 0
			for _, count := range player.Resources {
				totalResources += count
			}
			playerTracker.MaxResources = max(playerTracker.MaxResources, totalResources)
		}
	}
	
	// Log game event
	phaseStr := string(state.CurrentPhase)
	err := s.repo.Game.LogGameEvent(gameID, nil, "state_update", state, &state.CurrentRound, phaseStr)
	if err != nil {
		s.logger.Warn("Failed to log game event: %v", err)
	}
	
	return nil
}

// CompleteGame finishes tracking a game and calculates final statistics
func (s *DatabaseService) CompleteGame(gameID string, finalState *protocol.GameStatePayload, winner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.logger.Info("Completing game tracking: %s", gameID)
	
	tracker, exists := s.activeGames[gameID]
	if !exists {
		return fmt.Errorf("game not being tracked: %s", gameID)
	}
	
	// Find winner player ID
	var winnerPlayerID *int
	if winner != "" {
		player, err := s.repo.Player.GetPlayer(winner)
		if err == nil {
			winnerPlayerID = &player.ID
		}
	}
	
	// Update game status to completed
	err := s.repo.Game.UpdateGameStatus(gameID, "completed", winnerPlayerID)
	if err != nil {
		return fmt.Errorf("failed to update game status: %w", err)
	}
	
	// Update total rounds
	err = s.repo.Game.UpdateGameRounds(gameID, tracker.RoundCount)
	if err != nil {
		s.logger.Warn("Failed to update game rounds: %v", err)
	}
	
	// Update participant results
	position := 1
	for _, player := range finalState.Players {
		dbPlayer, err := s.repo.Player.GetPlayer(player.Name)
		if err != nil {
			s.logger.Error("Failed to get player %s: %v", player.Name, err)
			continue
		}
		
		// Calculate total resources
		totalResources := 0
		for _, count := range player.Resources {
			totalResources += count
		}
		
		// Count powered cities (assume all plants can be used)
		poweredCities := 0
		for _, plant := range player.PowerPlants {
			poweredCities += plant.Capacity
		}
		poweredCities = min(poweredCities, len(player.Cities))
		
		isWinner := player.Name == winner
		
		err = s.repo.Game.UpdateGameParticipantResults(
			gameID,
			dbPlayer.ID,
			position, // final position
			len(player.Cities),
			len(player.PowerPlants),
			player.Money,
			totalResources,
			poweredCities,
			isWinner,
		)
		if err != nil {
			s.logger.Error("Failed to update participant results for %s: %v", player.Name, err)
		}
		
		position++
	}
	
	// Log completion event
	err = s.repo.Game.LogGameEvent(gameID, nil, "game_completed", map[string]interface{}{
		"winner":       winner,
		"duration":     time.Since(tracker.StartTime).Minutes(),
		"total_rounds": tracker.RoundCount,
	}, &tracker.RoundCount, "completed")
	if err != nil {
		s.logger.Warn("Failed to log completion event: %v", err)
	}
	
	// Process achievements
	s.processAchievements(gameID, finalState, winner)
	
	// Remove from active tracking
	delete(s.activeGames, gameID)
	
	s.logger.Debug("Game tracking completed for %s", gameID)
	return nil
}

// GetPlayerStats retrieves statistics for a player
func (s *DatabaseService) GetPlayerStats(playerName string) (*models.PlayerStats, error) {
	player, err := s.repo.Player.GetPlayer(playerName)
	if err != nil {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	return s.repo.Player.GetPlayerStats(player.ID)
}

// GetPlayerAchievements retrieves achievements for a player
func (s *DatabaseService) GetPlayerAchievements(playerName string) ([]*models.PlayerAchievement, error) {
	player, err := s.repo.Player.GetPlayer(playerName)
	if err != nil {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	return s.repo.Achievement.GetPlayerAchievements(player.ID)
}

// GetLeaderboard retrieves the player leaderboard
func (s *DatabaseService) GetLeaderboard(limit int) ([]*models.LeaderboardEntry, error) {
	return s.repo.Player.GetLeaderboard(limit)
}

// GetGameAnalytics retrieves game analytics for the specified number of days
func (s *DatabaseService) GetGameAnalytics(days int) (*models.GameAnalytics, error) {
	analytics, err := s.repo.Game.GetGameAnalytics(days)
	if err != nil {
		return nil, err
	}
	
	// Get recent games
	recentGames, err := s.repo.Game.GetRecentGames(10, "completed")
	if err != nil {
		s.logger.Warn("Failed to get recent games: %v", err)
	} else {
		analytics.RecentGames = recentGames
	}
	
	return analytics, nil
}

// GetAchievementStats retrieves achievement statistics
func (s *DatabaseService) GetAchievementStats() (*models.AchievementStats, error) {
	return s.repo.Achievement.GetAchievementStats()
}

// GetPlayerPerformanceMetrics retrieves comprehensive player performance analytics
func (s *DatabaseService) GetPlayerPerformanceMetrics(playerName string, timeFrameDays int) (*repositories.PlayerPerformanceMetrics, error) {
	player, err := s.repo.Player.GetPlayer(playerName)
	if err != nil {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	return s.repo.Analytics.GetPlayerPerformanceMetrics(player.ID, timeFrameDays)
}

// GetAdvancedGameAnalytics retrieves comprehensive game analytics
func (s *DatabaseService) GetAdvancedGameAnalytics(timeFrameDays int) (*repositories.GameAnalyticsAdvanced, error) {
	return s.repo.Analytics.GetAdvancedGameAnalytics(timeFrameDays)
}

// GetCompetitorAnalysis retrieves head-to-head analysis for a player
func (s *DatabaseService) GetCompetitorAnalysis(playerName string, timeFrameDays int) ([]repositories.CompetitorMatchup, error) {
	player, err := s.repo.Player.GetPlayer(playerName)
	if err != nil {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	threshold := time.Now().AddDate(0, 0, -timeFrameDays)
	return s.repo.Analytics.GetCompetitorAnalysis(player.ID, threshold), nil
}

// GetPlayerSkillProgression retrieves skill development over time
func (s *DatabaseService) GetPlayerSkillProgression(playerName string, timeFrameDays int) ([]repositories.PlayerSkillProgressPoint, error) {
	player, err := s.repo.Player.GetPlayer(playerName)
	if err != nil {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	return s.repo.Analytics.GetSkillProgression(player.ID, timeFrameDays), nil
}

// GetMapAnalytics retrieves detailed map performance analytics
func (s *DatabaseService) GetMapAnalytics(timeFrameDays int) (map[string]repositories.MapAnalytics, error) {
	threshold := time.Now().AddDate(0, 0, -timeFrameDays)
	return s.repo.Analytics.GetMapAnalytics(threshold), nil
}

// GetPlayerTypeDistribution returns distribution of player types
func (s *DatabaseService) GetPlayerTypeDistribution() (map[string]int, error) {
	query := `
		SELECT 
			p.name,
			COALESCE(ps.games_played, 0) as games_played,
			COALESCE(ps.win_rate, 0) as win_rate,
			COALESCE(ps.avg_final_cities, 0) as avg_cities,
			COALESCE(ps.avg_final_money, 0) as avg_money
		FROM players p
		LEFT JOIN player_statistics ps ON p.id = ps.player_id
		WHERE COALESCE(ps.games_played, 0) >= 5
	`
	
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get player data: %w", err)
	}
	defer rows.Close()
	
	typeDistribution := make(map[string]int)
	
	for rows.Next() {
		var name string
		var gamesPlayed int
		var winRate, avgCities, avgMoney float64
		
		if err := rows.Scan(&name, &gamesPlayed, &winRate, &avgCities, &avgMoney); err != nil {
			continue
		}
		
		// Simple classification logic
		playerType := s.classifyPlayerByStats(gamesPlayed, winRate, avgCities, avgMoney)
		typeDistribution[playerType]++
	}
	
	return typeDistribution, nil
}

// Helper method to classify players by their statistics
func (s *DatabaseService) classifyPlayerByStats(gamesPlayed int, winRate, avgCities, avgMoney float64) string {
	if gamesPlayed < 5 {
		return "novice"
	}
	
	if winRate >= 0.6 {
		if avgCities >= 15 {
			return "expansion_master"
		} else if avgMoney >= 100 {
			return "economic_powerhouse"
		} else {
			return "strategic_dominator"
		}
	} else if winRate >= 0.4 {
		if avgCities >= 12 {
			return "aggressive_expander"
		} else {
			return "tactical_player"
		}
	} else if winRate >= 0.2 {
		if avgMoney >= 80 {
			return "resource_optimizer"
		} else {
			return "developing_player"
		}
	}
	return "casual_player"
}

// GetActivityReport generates a comprehensive activity report
func (s *DatabaseService) GetActivityReport(timeFrameDays int) (map[string]interface{}, error) {
	threshold := time.Now().AddDate(0, 0, -timeFrameDays)
	
	report := make(map[string]interface{})
	
	// Get basic activity metrics
	basicQuery := `
		SELECT 
			COUNT(DISTINCT g.id) as total_games,
			COUNT(DISTINCT gp.player_id) as unique_players,
			AVG(g.duration_minutes) as avg_game_duration,
			COUNT(DISTINCT DATE(g.created_at)) as active_days
		FROM games g
		LEFT JOIN game_participants gp ON g.id = gp.game_id
		WHERE g.created_at >= ?
	`
	
	var totalGames, uniquePlayers, activeDays int
	var avgDuration sql.NullFloat64
	
	err := s.db.QueryRow(basicQuery, threshold).Scan(&totalGames, &uniquePlayers, &avgDuration, &activeDays)
	if err != nil {
		return nil, fmt.Errorf("failed to get activity metrics: %w", err)
	}
	
	report["total_games"] = totalGames
	report["unique_players"] = uniquePlayers
	report["active_days"] = activeDays
	if avgDuration.Valid {
		report["avg_game_duration"] = avgDuration.Float64
	} else {
		report["avg_game_duration"] = 0.0
	}
	
	// Get daily activity
	dailyQuery := `
		SELECT 
			DATE(created_at) as date,
			COUNT(*) as games_count
		FROM games
		WHERE created_at >= ?
		GROUP BY DATE(created_at)
		ORDER BY date DESC
		LIMIT 30
	`
	
	rows, err := s.db.Query(dailyQuery, threshold)
	if err == nil {
		defer rows.Close()
		
		var dailyActivity []map[string]interface{}
		for rows.Next() {
			var date string
			var count int
			if err := rows.Scan(&date, &count); err == nil {
				dailyActivity = append(dailyActivity, map[string]interface{}{
					"date":  date,
					"games": count,
				})
			}
		}
		report["daily_activity"] = dailyActivity
	}
	
	// Get player type distribution
	typeDistribution, err := s.GetPlayerTypeDistribution()
	if err == nil {
		report["player_type_distribution"] = typeDistribution
	}
	
	report["generated_at"] = time.Now()
	report["time_frame_days"] = timeFrameDays
	
	return report, nil
}

// processAchievements checks and awards achievements for completed game
func (s *DatabaseService) processAchievements(gameID string, finalState *protocol.GameStatePayload, winner string) {
	// This is a simplified version - in a full implementation, we'd check all achievement criteria
	for _, player := range finalState.Players {
		dbPlayer, err := s.repo.Player.GetPlayer(player.Name)
		if err != nil {
			continue
		}
		
		// Check for first win achievement
		if player.Name == winner {
			s.checkAndAwardAchievement(dbPlayer.ID, "first_win", gameID)
		}
		
		// Check for city builder achievement
		if len(player.Cities) >= 15 {
			s.checkAndAwardAchievement(dbPlayer.ID, "city_builder", gameID)
		}
		
		// Check for money bags achievement
		if player.Money >= 200 {
			s.checkAndAwardAchievement(dbPlayer.ID, "money_bags", gameID)
		}
		
		// Check for plant collector achievement
		if len(player.PowerPlants) >= 5 {
			s.checkAndAwardAchievement(dbPlayer.ID, "plant_collector", gameID)
		}
	}
}

// checkAndAwardAchievement checks if player has achievement and awards it if not
func (s *DatabaseService) checkAndAwardAchievement(playerID int, achievementID string, gameID string) {
	achievement, err := s.repo.Achievement.GetAchievement(achievementID)
	if err != nil {
		s.logger.Debug("Achievement not found: %s", achievementID)
		return
	}
	
	// Check if player already has this achievement
	playerAchievements, err := s.repo.Achievement.GetPlayerAchievements(playerID)
	if err != nil {
		return
	}
	
	for _, pa := range playerAchievements {
		if pa.AchievementID == achievement.ID && pa.IsCompleted {
			return // Already has this achievement
		}
	}
	
	// Get game's internal ID
	var internalGameID *int
	if gameID != "" {
		game, err := s.repo.Game.GetGame(gameID)
		if err == nil {
			internalGameID = &game.ID
		}
	}
	
	// Award achievement
	_, err = s.repo.Achievement.CreatePlayerAchievement(playerID, achievement.ID, internalGameID, 1, 1)
	if err != nil {
		s.logger.Error("Failed to award achievement %s to player %d: %v", achievementID, playerID, err)
	} else {
		s.logger.Info("Awarded achievement %s to player %d", achievementID, playerID)
	}
}

// InitializeAchievements creates predefined achievements in the database
func (s *DatabaseService) InitializeAchievements() error {
	s.logger.Info("Initializing predefined achievements...")
	
	// Use the predefined achievements from the original models
	predefinedAchievements := []struct {
		ID          string
		Name        string
		Description string
		Icon        string
		Category    string
		Points      int
		Criteria    string
	}{
		{"first_win", "First Victory", "Win your first game", "🏆", "victory", 10, "games_won >= 1"},
		{"winning_streak_3", "Hat Trick", "Win 3 games in a row", "🎩", "victory", 25, "winning_streak >= 3"},
		{"master_strategist", "Master Strategist", "Win 50 games", "🧠", "victory", 100, "games_won >= 50"},
		{"money_bags", "Money Bags", "End a game with 200+ elektro", "💰", "economic", 20, "final_money >= 200"},
		{"resource_hoarder", "Resource Hoarder", "Own 20+ resources at once", "📦", "economic", 15, "max_resources >= 20"},
		{"city_builder", "City Builder", "Build in 15+ cities in a single game", "🏙️", "expansion", 25, "max_cities >= 15"},
		{"plant_collector", "Plant Collector", "Own 5 power plants at once", "🏭", "plants", 20, "max_plants >= 5"},
		{"regular_player", "Regular Player", "Play 25 games", "🎮", "participation", 15, "games_played >= 25"},
		{"dedicated_player", "Dedicated Player", "Play 100 games", "🌟", "participation", 50, "games_played >= 100"},
		{"speed_demon", "Speed Demon", "Win a game in under 30 minutes", "🏎️", "special", 35, "speed_win"},
	}
	
	for _, ach := range predefinedAchievements {
		// Check if achievement already exists
		existing, err := s.repo.Achievement.GetAchievement(ach.ID)
		if err == nil && existing != nil {
			s.logger.Debug("Achievement %s already exists, skipping", ach.ID)
			continue
		}
		
		_, err = s.repo.Achievement.CreateAchievement(
			ach.ID, ach.Name, ach.Description, ach.Category, 
			ach.Icon, ach.Criteria, ach.Points,
		)
		if err != nil {
			s.logger.Error("Failed to create achievement %s: %v", ach.ID, err)
		} else {
			s.logger.Debug("Created achievement: %s", ach.Name)
		}
	}
	
	s.logger.Info("Achievement initialization completed")
	return nil
}

// Close closes the database service
func (s *DatabaseService) Close() error {
	s.logger.Info("Closing database analytics service")
	return s.repo.Close()
}

// GetDatabaseStats returns database performance statistics
func (s *DatabaseService) GetDatabaseStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Get basic database stats
	stats["db_stats"] = s.db.GetStats()
	
	// Get connection pool stats
	if poolStats := s.db.GetPoolStats(); poolStats != nil {
		stats["pool_stats"] = poolStats
	}
	
	// Get optimizer stats
	if optimizerStats := s.db.GetOptimizerStats(); optimizerStats != nil {
		stats["optimizer_stats"] = optimizerStats
	}
	
	// Get database size information
	if sizes, err := s.db.GetDatabaseSize(); err == nil {
		stats["database_size"] = sizes
	}
	
	// Get table sizes
	if tableSizes, err := s.db.GetTableSizes(); err == nil {
		stats["table_sizes"] = tableSizes
	}
	
	return stats
}

// OptimizeDatabase performs database optimization
func (s *DatabaseService) OptimizeDatabase() error {
	s.logger.Info("Starting database optimization")
	return s.db.OptimizeNow()
}

// GetQueryPlan analyzes a query's execution plan
func (s *DatabaseService) GetQueryPlan(query string, args ...interface{}) (*database.QueryPlan, error) {
	return s.db.GetQueryPlan(query, args...)
}

// GetIndexUsage returns index usage statistics
func (s *DatabaseService) GetIndexUsage() ([]database.IndexUsage, error) {
	return s.db.GetIndexUsage()
}

// Helper functions
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}