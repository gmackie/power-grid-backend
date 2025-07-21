package analytics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"powergrid/models"
	"powergrid/pkg/logger"
	"powergrid/pkg/protocol"
)

// Service manages analytics and achievements
type Service struct {
	mu                sync.RWMutex
	dataDir           string
	logger            *logger.ColoredLogger
	
	// In-memory storage (in production, use a database)
	playerStats       map[string]*models.PlayerStats
	gameRecords       []models.GameRecord
	playerAchievements map[string][]models.PlayerAchievement
	achievements      map[string]models.Achievement
	
	// Active game tracking
	activeGames       map[string]*GameTracker
}

// GameTracker tracks an active game for analytics
type GameTracker struct {
	GameID      string
	GameName    string
	MapName     string
	StartTime   time.Time
	Players     map[string]*PlayerTracker
	RoundCount  int
	LastUpdate  time.Time
}

// PlayerTracker tracks a player in an active game
type PlayerTracker struct {
	PlayerName      string
	MaxCities       int
	MaxPlants       int
	MaxMoney        int
	MaxResources    int
	ResourcesUsed   int
	PlantTypes      map[string]bool
	RegionControl   map[string]int
	Position        int
}

// NewService creates a new analytics service
func NewService(dataDir string) *Service {
	service := &Service{
		dataDir:            dataDir,
		logger:             logger.CreateAILogger("Analytics", logger.ColorBrightPurple),
		playerStats:        make(map[string]*models.PlayerStats),
		gameRecords:        make([]models.GameRecord, 0),
		playerAchievements: make(map[string][]models.PlayerAchievement),
		achievements:       make(map[string]models.Achievement),
		activeGames:        make(map[string]*GameTracker),
	}
	
	// Initialize achievements
	for _, achievement := range models.PredefinedAchievements {
		service.achievements[achievement.ID] = achievement
	}
	
	// Load existing data
	service.loadData()
	
	// Start periodic save
	go service.periodicSave()
	
	return service
}

// TrackGameStart records the start of a new game
func (s *Service) TrackGameStart(gameID, gameName, mapName string, players []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tracker := &GameTracker{
		GameID:     gameID,
		GameName:   gameName,
		MapName:    mapName,
		StartTime:  time.Now(),
		Players:    make(map[string]*PlayerTracker),
		LastUpdate: time.Now(),
	}
	
	for _, playerName := range players {
		tracker.Players[playerName] = &PlayerTracker{
			PlayerName: playerName,
			PlantTypes: make(map[string]bool),
			RegionControl: make(map[string]int),
		}
		
		// Initialize player stats if not exists
		if _, exists := s.playerStats[playerName]; !exists {
			s.playerStats[playerName] = &models.PlayerStats{
				PlayerName: playerName,
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			}
		}
	}
	
	s.activeGames[gameID] = tracker
	s.logger.Info("Started tracking game %s with %d players", gameID, len(players))
}

// TrackGameState updates analytics based on game state
func (s *Service) TrackGameState(gameID string, state *protocol.GameStatePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tracker, exists := s.activeGames[gameID]
	if !exists {
		return
	}
	
	tracker.RoundCount = state.CurrentRound
	tracker.LastUpdate = time.Now()
	
	// Update player tracking
	for playerID, player := range state.Players {
		if playerTracker, ok := tracker.Players[player.Name]; ok {
			// Update maximums
			if len(player.Cities) > playerTracker.MaxCities {
				playerTracker.MaxCities = len(player.Cities)
			}
			if len(player.PowerPlants) > playerTracker.MaxPlants {
				playerTracker.MaxPlants = len(player.PowerPlants)
			}
			if player.Money > playerTracker.MaxMoney {
				playerTracker.MaxMoney = player.Money
			}
			
			// Track resources
			totalResources := 0
			for _, count := range player.Resources {
				totalResources += count
			}
			if totalResources > playerTracker.MaxResources {
				playerTracker.MaxResources = totalResources
			}
			
			// Track plant diversity
			for _, plant := range player.PowerPlants {
				if plant.ResourceType != "" {
					playerTracker.PlantTypes[plant.ResourceType] = true
				}
			}
			
			// Track position in turn order
			for i, pid := range state.TurnOrder {
				if pid == playerID {
					playerTracker.Position = i + 1
					break
				}
			}
		}
	}
}

// TrackGameEnd records the completion of a game
func (s *Service) TrackGameEnd(gameID string, winner string, finalState *protocol.GameStatePayload) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	tracker, exists := s.activeGames[gameID]
	if !exists {
		return
	}
	
	// Create game record
	endTime := time.Now()
	duration := int(endTime.Sub(tracker.StartTime).Minutes())
	
	gameRecord := models.GameRecord{
		ID:          fmt.Sprintf("%s_%d", gameID, time.Now().Unix()),
		GameID:      gameID,
		GameName:    tracker.GameName,
		MapName:     tracker.MapName,
		StartTime:   tracker.StartTime,
		EndTime:     endTime,
		Duration:    duration,
		Winner:      winner,
		TotalRounds: tracker.RoundCount,
		Players:     make([]models.PlayerGameResult, 0),
		CreatedAt:   time.Now(),
	}
	
	// Process each player's results
	for playerName, playerTracker := range tracker.Players {
		var playerInfo protocol.PlayerInfo
		for _, p := range finalState.Players {
			if p.Name == playerName {
				playerInfo = p
				break
			}
		}
		
		isWinner := playerName == winner
		
		result := models.PlayerGameResult{
			PlayerName:    playerName,
			Position:      playerTracker.Position,
			FinalCities:   len(playerInfo.Cities),
			FinalPlants:   len(playerInfo.PowerPlants),
			FinalMoney:    playerInfo.Money,
			PoweredCities: playerInfo.PoweredCities,
			ResourcesUsed: playerTracker.ResourcesUsed,
			IsWinner:      isWinner,
		}
		
		gameRecord.Players = append(gameRecord.Players, result)
		
		// Update player stats
		stats := s.playerStats[playerName]
		if stats == nil {
			stats = &models.PlayerStats{
				PlayerName: playerName,
				CreatedAt:  time.Now(),
			}
			s.playerStats[playerName] = stats
		}
		
		stats.GamesPlayed++
		if isWinner {
			stats.GamesWon++
		}
		stats.WinRate = float64(stats.GamesWon) / float64(stats.GamesPlayed) * 100
		stats.TotalCities += len(playerInfo.Cities)
		stats.TotalPlants += len(playerInfo.PowerPlants)
		stats.TotalMoney += playerInfo.Money
		stats.TotalResources += playerTracker.ResourcesUsed
		stats.PlayTime += duration
		stats.LastPlayed = endTime
		stats.UpdatedAt = time.Now()
		
		// Check achievements
		s.checkAchievements(playerName, &gameRecord, playerTracker, isWinner)
	}
	
	// Store game record
	s.gameRecords = append(s.gameRecords, gameRecord)
	
	// Remove from active games
	delete(s.activeGames, gameID)
	
	s.logger.Info("Recorded game end for %s, winner: %s, duration: %d minutes", 
		gameID, winner, duration)
}

// GetPlayerStats returns statistics for a player
func (s *Service) GetPlayerStats(playerName string) (*models.PlayerStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats, exists := s.playerStats[playerName]
	if !exists {
		return nil, fmt.Errorf("player not found: %s", playerName)
	}
	
	return stats, nil
}

// GetPlayerAchievements returns achievements for a player
func (s *Service) GetPlayerAchievements(playerName string) ([]models.PlayerAchievement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	achievements, exists := s.playerAchievements[playerName]
	if !exists {
		return []models.PlayerAchievement{}, nil
	}
	
	// Populate achievement details
	for i := range achievements {
		if ach, ok := s.achievements[achievements[i].AchievementID]; ok {
			achievements[i].Achievement = ach
		}
	}
	
	return achievements, nil
}

// GetLeaderboard returns the top players
func (s *Service) GetLeaderboard(limit int) []models.LeaderboardEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	entries := make([]models.LeaderboardEntry, 0)
	rank := 1
	
	// Convert to slice for sorting
	for _, stats := range s.playerStats {
		if stats.GamesPlayed >= 5 { // Minimum games for leaderboard
			entry := models.LeaderboardEntry{
				Rank:       rank,
				PlayerName: stats.PlayerName,
				Score:      s.calculateScore(stats),
				GamesWon:   stats.GamesWon,
				WinRate:    stats.WinRate,
				LastPlayed: stats.LastPlayed,
			}
			entries = append(entries, entry)
			rank++
		}
	}
	
	// Sort by score
	sortLeaderboard(entries)
	
	// Update ranks after sorting
	for i := range entries {
		entries[i].Rank = i + 1
	}
	
	// Limit results
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}
	
	return entries
}

// GetGameHistory returns recent games for a player
func (s *Service) GetGameHistory(playerName string, limit int) []models.GameRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	history := make([]models.GameRecord, 0)
	
	// Find games with this player (reverse order for most recent first)
	for i := len(s.gameRecords) - 1; i >= 0; i-- {
		game := s.gameRecords[i]
		for _, player := range game.Players {
			if player.PlayerName == playerName {
				history = append(history, game)
				break
			}
		}
		
		if limit > 0 && len(history) >= limit {
			break
		}
	}
	
	return history
}

// GetGameAnalytics returns analytics for all games
func (s *Service) GetGameAnalytics(timeRange *models.TimeRange) models.GameAnalyticsResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	response := models.GameAnalyticsResponse{
		PopularMaps: make(map[string]int),
		PeakHours:   make(map[int]int),
		RecentGames: make([]models.GameRecord, 0),
	}
	
	totalDuration := 0
	validGames := 0
	
	for _, game := range s.gameRecords {
		// Apply time filter if provided
		if timeRange != nil {
			if game.StartTime.Before(timeRange.Start) || game.StartTime.After(timeRange.End) {
				continue
			}
		}
		
		response.TotalGames++
		response.PopularMaps[game.MapName]++
		response.PeakHours[game.StartTime.Hour()]++
		
		if game.Duration > 0 {
			totalDuration += game.Duration
			validGames++
		}
		
		// Add to recent games (last 10)
		if len(response.RecentGames) < 10 {
			response.RecentGames = append(response.RecentGames, game)
		}
	}
	
	if validGames > 0 {
		response.AverageGameTime = totalDuration / validGames
	}
	
	return response
}

// Private helper methods

func (s *Service) checkAchievements(playerName string, gameRecord *models.GameRecord, 
	tracker *PlayerTracker, isWinner bool) {
	
	stats := s.playerStats[playerName]
	if stats == nil {
		return
	}
	
	// Get existing achievements
	earnedAchievements := make(map[string]bool)
	if achievements, exists := s.playerAchievements[playerName]; exists {
		for _, ach := range achievements {
			earnedAchievements[ach.AchievementID] = true
		}
	} else {
		s.playerAchievements[playerName] = make([]models.PlayerAchievement, 0)
	}
	
	// Check each achievement
	for id, achievement := range s.achievements {
		if earnedAchievements[id] {
			continue // Already earned
		}
		
		earned := false
		
		switch achievement.ID {
		case "first_win":
			earned = stats.GamesWon >= 1
		case "master_strategist":
			earned = stats.GamesWon >= 50
		case "money_bags":
			earned = tracker.MaxMoney >= 200
		case "city_builder":
			earned = tracker.MaxCities >= 15
		case "plant_collector":
			earned = tracker.MaxPlants >= 5
		case "regular_player":
			earned = stats.GamesPlayed >= 25
		case "dedicated_player":
			earned = stats.GamesPlayed >= 100
		case "diversified":
			earned = len(tracker.PlantTypes) >= 4
		case "underdog":
			earned = isWinner && tracker.Position == len(gameRecord.Players)
		case "speed_demon":
			earned = isWinner && gameRecord.Duration < 30
		}
		
		if earned {
			playerAch := models.PlayerAchievement{
				PlayerName:    playerName,
				AchievementID: id,
				Achievement:   achievement,
				EarnedAt:      time.Now(),
				GameID:        gameRecord.GameID,
			}
			
			s.playerAchievements[playerName] = append(s.playerAchievements[playerName], playerAch)
			s.logger.Info("Player %s earned achievement: %s", playerName, achievement.Name)
		}
	}
}

func (s *Service) calculateScore(stats *models.PlayerStats) int {
	// Composite score based on various factors
	score := stats.GamesWon * 100
	score += int(stats.WinRate * 10)
	score += stats.TotalCities
	score += stats.TotalPlants * 5
	
	// Achievement bonus
	if achievements, exists := s.playerAchievements[stats.PlayerName]; exists {
		for _, ach := range achievements {
			score += ach.Achievement.Points
		}
	}
	
	return score
}

func sortLeaderboard(entries []models.LeaderboardEntry) {
	// Sort by score descending
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			if entries[j].Score > entries[i].Score {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

// Data persistence

func (s *Service) loadData() {
	// Load player stats
	statsFile := filepath.Join(s.dataDir, "player_stats.json")
	if data, err := os.ReadFile(statsFile); err == nil {
		json.Unmarshal(data, &s.playerStats)
		s.logger.Info("Loaded %d player stats", len(s.playerStats))
	}
	
	// Load game records
	recordsFile := filepath.Join(s.dataDir, "game_records.json")
	if data, err := os.ReadFile(recordsFile); err == nil {
		json.Unmarshal(data, &s.gameRecords)
		s.logger.Info("Loaded %d game records", len(s.gameRecords))
	}
	
	// Load achievements
	achievementsFile := filepath.Join(s.dataDir, "player_achievements.json")
	if data, err := os.ReadFile(achievementsFile); err == nil {
		json.Unmarshal(data, &s.playerAchievements)
		s.logger.Info("Loaded achievements for %d players", len(s.playerAchievements))
	}
}

func (s *Service) saveData() {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	// Ensure directory exists
	os.MkdirAll(s.dataDir, 0755)
	
	// Save player stats
	if data, err := json.MarshalIndent(s.playerStats, "", "  "); err == nil {
		statsFile := filepath.Join(s.dataDir, "player_stats.json")
		os.WriteFile(statsFile, data, 0644)
	}
	
	// Save game records
	if data, err := json.MarshalIndent(s.gameRecords, "", "  "); err == nil {
		recordsFile := filepath.Join(s.dataDir, "game_records.json")
		os.WriteFile(recordsFile, data, 0644)
	}
	
	// Save achievements
	if data, err := json.MarshalIndent(s.playerAchievements, "", "  "); err == nil {
		achievementsFile := filepath.Join(s.dataDir, "player_achievements.json")
		os.WriteFile(achievementsFile, data, 0644)
	}
}

func (s *Service) periodicSave() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		s.saveData()
		s.logger.Debug("Saved analytics data")
	}
}