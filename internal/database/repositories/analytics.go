package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"powergrid/pkg/logger"
)

// AnalyticsRepository handles advanced analytics queries
type AnalyticsRepository struct {
	db     *sql.DB
	logger *logger.ColoredLogger
}

// NewAnalyticsRepository creates a new analytics repository
func NewAnalyticsRepository(db *sql.DB) *AnalyticsRepository {
	return &AnalyticsRepository{
		db:     db,
		logger: logger.CreateAILogger("AnalyticsRepo", logger.ColorBrightGreen),
	}
}

// PlayerPerformanceMetrics represents detailed player performance analytics
type PlayerPerformanceMetrics struct {
	PlayerID                int                            `json:"player_id"`
	PlayerName              string                         `json:"player_name"`
	TotalGames              int                            `json:"total_games"`
	WinRate                 float64                        `json:"win_rate"`
	AvgPosition             float64                        `json:"avg_position"`
	AvgCitiesBuilt          float64                        `json:"avg_cities_built"`
	AvgPlantsOwned          float64                        `json:"avg_plants_owned"`
	AvgFinalMoney           float64                        `json:"avg_final_money"`
	ResourceEfficiency      float64                        `json:"resource_efficiency"`
	ExpansionRate           float64                        `json:"expansion_rate"`
	EconomicPerformance     float64                        `json:"economic_performance"`
	StrategicConsistency    float64                        `json:"strategic_consistency"`
	RecentFormTrend         string                         `json:"recent_form_trend"`
	MapPreferences          map[string]float64             `json:"map_preferences"`
	PlayerTypeClassification string                        `json:"player_type_classification"`
	SkillProgression        []PlayerSkillProgressPoint    `json:"skill_progression"`
	CompetitorAnalysis      []CompetitorMatchup           `json:"competitor_analysis"`
}

// PlayerSkillProgressPoint represents a point in time for skill tracking
type PlayerSkillProgressPoint struct {
	Date            time.Time `json:"date"`
	GamesPlayed     int       `json:"games_played"`
	WinRate         float64   `json:"win_rate"`
	AvgPosition     float64   `json:"avg_position"`
	SkillRating     float64   `json:"skill_rating"`
	ConfidenceLevel float64   `json:"confidence_level"`
}

// CompetitorMatchup represents head-to-head analysis
type CompetitorMatchup struct {
	OpponentID     int     `json:"opponent_id"`
	OpponentName   string  `json:"opponent_name"`
	GamesPlayed    int     `json:"games_played"`
	Wins           int     `json:"wins"`
	Losses         int     `json:"losses"`
	WinRate        float64 `json:"win_rate"`
	AvgPositionDiff float64 `json:"avg_position_diff"`
	Dominance      string  `json:"dominance"`
}

// GameAnalyticsAdvanced represents comprehensive game analytics
type GameAnalyticsAdvanced struct {
	TimeFrameDays           int                            `json:"time_frame_days"`
	TotalGames              int                            `json:"total_games"`
	CompletedGames          int                            `json:"completed_games"`
	CompletionRate          float64                        `json:"completion_rate"`
	AvgGameDuration         float64                        `json:"avg_game_duration"`
	GameDurationDistribution map[string]int                `json:"game_duration_distribution"`
	PlayerCountDistribution map[int]int                    `json:"player_count_distribution"`
	MapPopularity           map[string]MapAnalytics        `json:"map_popularity"`
	PeakPlayTimes           map[int]int                    `json:"peak_play_times"`
	SeasonalTrends          []SeasonalTrendPoint           `json:"seasonal_trends"`
	CompetitivenessMetrics  CompetitivenessAnalysis        `json:"competitiveness_metrics"`
	PowerPlantAnalytics     PowerPlantMarketAnalysis       `json:"power_plant_analytics"`
	ResourceMarketAnalytics ResourceMarketAnalysis         `json:"resource_market_analytics"`
	StrategyAnalysis        StrategyEffectivenessAnalysis  `json:"strategy_analysis"`
}

// MapAnalytics represents detailed map performance data
type MapAnalytics struct {
	MapName                string  `json:"map_name"`
	GamesPlayed            int     `json:"games_played"`
	AvgGameDuration        float64 `json:"avg_game_duration"`
	AvgFinalCities         float64 `json:"avg_final_cities"`
	AvgFinalMoney          float64 `json:"avg_final_money"`
	CompetitivenessScore   float64 `json:"competitiveness_score"`
	PopularityRank         int     `json:"popularity_rank"`
	BalanceScore           float64 `json:"balance_score"`
	NewPlayerFriendliness  float64 `json:"new_player_friendliness"`
}

// SeasonalTrendPoint represents analytics over time
type SeasonalTrendPoint struct {
	Date                time.Time `json:"date"`
	GamesPlayed         int       `json:"games_played"`
	UniquePlayersActive int       `json:"unique_players_active"`
	AvgGameDuration     float64   `json:"avg_game_duration"`
	CompletionRate      float64   `json:"completion_rate"`
}

// CompetitivenessAnalysis measures how competitive games are
type CompetitivenessAnalysis struct {
	AvgMarginOfVictory      float64 `json:"avg_margin_of_victory"`
	CloseGamePercentage     float64 `json:"close_game_percentage"`
	DominantWinPercentage   float64 `json:"dominant_win_percentage"`
	ComebackPercentage      float64 `json:"comeback_percentage"`
	PositionVariabilityScore float64 `json:"position_variability_score"`
	SkillSpreadIndex        float64 `json:"skill_spread_index"`
}

// PowerPlantMarketAnalysis analyzes power plant market dynamics
type PowerPlantMarketAnalysis struct {
	MostPopularPlants      []PowerPlantPopularity `json:"most_popular_plants"`
	AvgAuctionPrices       map[int]float64        `json:"avg_auction_prices"`
	PlantEfficiencyRatings map[int]float64        `json:"plant_efficiency_ratings"`
	EarlyGamePreferences   []int                  `json:"early_game_preferences"`
	LateGamePreferences    []int                  `json:"late_game_preferences"`
	ResourceTypePopularity map[string]float64     `json:"resource_type_popularity"`
}

// PowerPlantPopularity represents plant usage statistics
type PowerPlantPopularity struct {
	PlantID          int     `json:"plant_id"`
	PlantNumber      int     `json:"plant_number"`
	ResourceType     string  `json:"resource_type"`
	Capacity         int     `json:"capacity"`
	TimesOwned       int     `json:"times_owned"`
	AvgAuctionPrice  float64 `json:"avg_auction_price"`
	WinRateWithPlant float64 `json:"win_rate_with_plant"`
	EfficiencyRating float64 `json:"efficiency_rating"`
}

// ResourceMarketAnalysis analyzes resource market trends
type ResourceMarketAnalysis struct {
	ResourceConsumption     map[string]float64     `json:"resource_consumption"`
	AvgResourcePrices       map[string]float64     `json:"avg_resource_prices"`
	ResourceScarcityEvents  []ResourceScarcityEvent `json:"resource_scarcity_events"`
	PlayerResourceStrategies map[string]float64     `json:"player_resource_strategies"`
	ResourceEfficiencyScores map[string]float64     `json:"resource_efficiency_scores"`
}

// ResourceScarcityEvent represents times when resources were scarce
type ResourceScarcityEvent struct {
	ResourceType string    `json:"resource_type"`
	Date         time.Time `json:"date"`
	ScarcityLevel float64   `json:"scarcity_level"`
	Impact       string    `json:"impact"`
	Duration     int       `json:"duration_minutes"`
}

// StrategyEffectivenessAnalysis analyzes different strategic approaches
type StrategyEffectivenessAnalysis struct {
	EarlyExpansionWinRate    float64 `json:"early_expansion_win_rate"`
	ResourceHoardingWinRate  float64 `json:"resource_hoarding_win_rate"`
	PlantSpecializationWinRate float64 `json:"plant_specialization_win_rate"`
	BalancedApproachWinRate  float64 `json:"balanced_approach_win_rate"`
	AggressiveBiddingWinRate float64 `json:"aggressive_bidding_win_rate"`
	ConservativeWinRate      float64 `json:"conservative_win_rate"`
}

// GetPlayerPerformanceMetrics retrieves comprehensive player performance analytics
func (r *AnalyticsRepository) GetPlayerPerformanceMetrics(playerID int, timeFrameDays int) (*PlayerPerformanceMetrics, error) {
	metrics := &PlayerPerformanceMetrics{
		PlayerID:           playerID,
		MapPreferences:     make(map[string]float64),
		SkillProgression:   make([]PlayerSkillProgressPoint, 0),
		CompetitorAnalysis: make([]CompetitorMatchup, 0),
	}

	// Get player basic info
	var playerName string
	err := r.db.QueryRow("SELECT name FROM players WHERE id = ?", playerID).Scan(&playerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get player name: %w", err)
	}
	metrics.PlayerName = playerName

	// Calculate time threshold
	threshold := time.Now().AddDate(0, 0, -timeFrameDays)

	// Get basic performance metrics
	query := `
		SELECT 
			COUNT(*) as total_games,
			COUNT(CASE WHEN is_winner = TRUE THEN 1 END) as wins,
			AVG(final_position) as avg_position,
			AVG(final_cities) as avg_cities,
			AVG(final_plants) as avg_plants,
			AVG(final_money) as avg_money
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
	`
	
	var wins int
	err = r.db.QueryRow(query, playerID, threshold).Scan(
		&metrics.TotalGames,
		&wins,
		&metrics.AvgPosition,
		&metrics.AvgCitiesBuilt,
		&metrics.AvgPlantsOwned,
		&metrics.AvgFinalMoney,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic metrics: %w", err)
	}

	if metrics.TotalGames > 0 {
		metrics.WinRate = float64(wins) / float64(metrics.TotalGames)
	}

	// Calculate advanced metrics
	metrics.ResourceEfficiency = r.calculateResourceEfficiency(playerID, threshold)
	metrics.ExpansionRate = r.calculateExpansionRate(playerID, threshold)
	metrics.EconomicPerformance = r.calculateEconomicPerformance(playerID, threshold)
	metrics.StrategicConsistency = r.calculateStrategicConsistency(playerID, threshold)
	metrics.RecentFormTrend = r.calculateRecentFormTrend(playerID, 10) // Last 10 games
	metrics.PlayerTypeClassification = r.classifyPlayerType(metrics)

	// Get map preferences
	metrics.MapPreferences = r.getMapPreferences(playerID, threshold)

	// Get skill progression
	metrics.SkillProgression = r.GetSkillProgression(playerID, timeFrameDays)

	// Get competitor analysis
	metrics.CompetitorAnalysis = r.GetCompetitorAnalysis(playerID, threshold)

	return metrics, nil
}

// GetAdvancedGameAnalytics retrieves comprehensive game analytics
func (r *AnalyticsRepository) GetAdvancedGameAnalytics(timeFrameDays int) (*GameAnalyticsAdvanced, error) {
	analytics := &GameAnalyticsAdvanced{
		TimeFrameDays:           timeFrameDays,
		GameDurationDistribution: make(map[string]int),
		PlayerCountDistribution: make(map[int]int),
		MapPopularity:           make(map[string]MapAnalytics),
		PeakPlayTimes:           make(map[int]int),
		SeasonalTrends:          make([]SeasonalTrendPoint, 0),
	}

	threshold := time.Now().AddDate(0, 0, -timeFrameDays)

	// Get basic game statistics
	query := `
		SELECT 
			COUNT(*) as total_games,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_games,
			AVG(CASE WHEN status = 'completed' THEN duration_minutes END) as avg_duration
		FROM games
		WHERE created_at >= ?
	`
	
	err := r.db.QueryRow(query, threshold).Scan(
		&analytics.TotalGames,
		&analytics.CompletedGames,
		&analytics.AvgGameDuration,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic game analytics: %w", err)
	}

	if analytics.TotalGames > 0 {
		analytics.CompletionRate = float64(analytics.CompletedGames) / float64(analytics.TotalGames)
	}

	// Get game duration distribution
	analytics.GameDurationDistribution = r.getGameDurationDistribution(threshold)

	// Get player count distribution
	analytics.PlayerCountDistribution = r.getPlayerCountDistribution(threshold)

	// Get map analytics
	analytics.MapPopularity = r.GetMapAnalytics(threshold)

	// Get peak play times
	analytics.PeakPlayTimes = r.getPeakPlayTimes(threshold)

	// Get seasonal trends
	analytics.SeasonalTrends = r.getSeasonalTrends(timeFrameDays)

	// Get competitiveness metrics
	analytics.CompetitivenessMetrics = r.getCompetitivenessMetrics(threshold)

	// Get power plant analytics
	analytics.PowerPlantAnalytics = r.getPowerPlantAnalytics(threshold)

	// Get resource market analytics
	analytics.ResourceMarketAnalytics = r.getResourceMarketAnalytics(threshold)

	// Get strategy analysis
	analytics.StrategyAnalysis = r.getStrategyAnalysis(threshold)

	return analytics, nil
}

// Helper methods for advanced calculations

func (r *AnalyticsRepository) calculateResourceEfficiency(playerID int, threshold time.Time) float64 {
	query := `
		SELECT 
			AVG(CAST(final_money AS FLOAT) / GREATEST(final_resources, 1)) as efficiency
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
	`
	
	var efficiency sql.NullFloat64
	r.db.QueryRow(query, playerID, threshold).Scan(&efficiency)
	
	if efficiency.Valid {
		return efficiency.Float64
	}
	return 0.0
}

func (r *AnalyticsRepository) calculateExpansionRate(playerID int, threshold time.Time) float64 {
	query := `
		SELECT 
			AVG(CAST(final_cities AS FLOAT) / GREATEST(g.total_rounds, 1)) as expansion_rate
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed' AND g.total_rounds > 0
	`
	
	var rate sql.NullFloat64
	r.db.QueryRow(query, playerID, threshold).Scan(&rate)
	
	if rate.Valid {
		return rate.Float64
	}
	return 0.0
}

func (r *AnalyticsRepository) calculateEconomicPerformance(playerID int, threshold time.Time) float64 {
	query := `
		SELECT 
			AVG(CAST(final_money AS FLOAT) / GREATEST(final_cities * 10, 1)) as economic_performance
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
	`
	
	var performance sql.NullFloat64
	r.db.QueryRow(query, playerID, threshold).Scan(&performance)
	
	if performance.Valid {
		return performance.Float64
	}
	return 0.0
}

func (r *AnalyticsRepository) calculateStrategicConsistency(playerID int, threshold time.Time) float64 {
	// Calculate variance in performance metrics as inverse of consistency
	query := `
		SELECT 
			AVG(final_position) as avg_pos,
			AVG(final_cities) as avg_cities,
			AVG(final_plants) as avg_plants,
			COUNT(*) as games
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
	`
	
	var avgPos, avgCities, avgPlants float64
	var games int
	err := r.db.QueryRow(query, playerID, threshold).Scan(&avgPos, &avgCities, &avgPlants, &games)
	if err != nil || games == 0 {
		return 0.0
	}

	// Calculate variance
	varQuery := `
		SELECT 
			AVG(POWER(final_position - ?, 2)) as pos_var,
			AVG(POWER(final_cities - ?, 2)) as cities_var,
			AVG(POWER(final_plants - ?, 2)) as plants_var
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
	`
	
	var posVar, citiesVar, plantsVar float64
	err = r.db.QueryRow(varQuery, avgPos, avgCities, avgPlants, playerID, threshold).Scan(&posVar, &citiesVar, &plantsVar)
	if err != nil {
		return 0.0
	}

	// Return inverse of combined variance (higher = more consistent)
	totalVar := posVar + citiesVar + plantsVar
	if totalVar == 0 {
		return 100.0
	}
	return 100.0 / (1.0 + totalVar)
}

func (r *AnalyticsRepository) calculateRecentFormTrend(playerID int, lastNGames int) string {
	query := `
		SELECT final_position, is_winner
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.status = 'completed'
		ORDER BY g.ended_at DESC
		LIMIT ?
	`
	
	rows, err := r.db.Query(query, playerID, lastNGames)
	if err != nil {
		return "unknown"
	}
	defer rows.Close()

	var positions []float64
	var wins int
	
	for rows.Next() {
		var position float64
		var isWinner bool
		if err := rows.Scan(&position, &isWinner); err != nil {
			continue
		}
		positions = append(positions, position)
		if isWinner {
			wins++
		}
	}

	if len(positions) == 0 {
		return "insufficient_data"
	}

	// Calculate trend
	if len(positions) >= 3 {
		recent := positions[0]
		older := positions[len(positions)-1]
		if recent < older {
			return "improving"
		} else if recent > older {
			return "declining"
		}
	}

	winRate := float64(wins) / float64(len(positions))
	if winRate >= 0.6 {
		return "excellent"
	} else if winRate >= 0.4 {
		return "good"
	} else if winRate >= 0.2 {
		return "average"
	}
	return "struggling"
}

func (r *AnalyticsRepository) classifyPlayerType(metrics *PlayerPerformanceMetrics) string {
	if metrics.TotalGames < 5 {
		return "novice"
	}

	if metrics.WinRate >= 0.6 {
		if metrics.AvgCitiesBuilt >= 15 {
			return "expansion_master"
		} else if metrics.AvgFinalMoney >= 100 {
			return "economic_powerhouse"
		} else {
			return "strategic_dominator"
		}
	} else if metrics.WinRate >= 0.4 {
		if metrics.StrategicConsistency >= 70 {
			return "consistent_performer"
		} else if metrics.ExpansionRate >= 1.5 {
			return "aggressive_expander"
		} else {
			return "tactical_player"
		}
	} else if metrics.WinRate >= 0.2 {
		if metrics.ResourceEfficiency >= 5.0 {
			return "resource_optimizer"
		} else {
			return "developing_player"
		}
	}
	return "casual_player"
}

func (r *AnalyticsRepository) getMapPreferences(playerID int, threshold time.Time) map[string]float64 {
	query := `
		SELECT 
			g.map_name,
			COUNT(*) as games,
			AVG(CASE WHEN gp.is_winner = TRUE THEN 1.0 ELSE 0.0 END) as win_rate
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE gp.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
		GROUP BY g.map_name
		HAVING games >= 3
	`
	
	rows, err := r.db.Query(query, playerID, threshold)
	if err != nil {
		return make(map[string]float64)
	}
	defer rows.Close()

	preferences := make(map[string]float64)
	for rows.Next() {
		var mapName string
		var games int
		var winRate float64
		if err := rows.Scan(&mapName, &games, &winRate); err != nil {
			continue
		}
		preferences[mapName] = winRate
	}

	return preferences
}

func (r *AnalyticsRepository) GetSkillProgression(playerID int, timeFrameDays int) []PlayerSkillProgressPoint {
	// Implementation would track skill over time
	// This is a simplified version
	return []PlayerSkillProgressPoint{}
}

func (r *AnalyticsRepository) GetCompetitorAnalysis(playerID int, threshold time.Time) []CompetitorMatchup {
	query := `
		SELECT 
			opp.player_id,
			opp.player_name,
			COUNT(*) as games,
			COUNT(CASE WHEN me.is_winner = TRUE THEN 1 END) as wins,
			COUNT(CASE WHEN me.is_winner = FALSE THEN 1 END) as losses,
			AVG(me.final_position - opp.final_position) as avg_position_diff
		FROM game_participants me
		JOIN game_participants opp ON me.game_id = opp.game_id AND me.player_id != opp.player_id
		JOIN games g ON me.game_id = g.id
		WHERE me.player_id = ? AND g.ended_at >= ? AND g.status = 'completed'
		GROUP BY opp.player_id, opp.player_name
		HAVING games >= 3
		ORDER BY games DESC
		LIMIT 10
	`
	
	rows, err := r.db.Query(query, playerID, threshold)
	if err != nil {
		return []CompetitorMatchup{}
	}
	defer rows.Close()

	var matchups []CompetitorMatchup
	for rows.Next() {
		var matchup CompetitorMatchup
		err := rows.Scan(
			&matchup.OpponentID,
			&matchup.OpponentName,
			&matchup.GamesPlayed,
			&matchup.Wins,
			&matchup.Losses,
			&matchup.AvgPositionDiff,
		)
		if err != nil {
			continue
		}
		
		matchup.WinRate = float64(matchup.Wins) / float64(matchup.GamesPlayed)
		
		if matchup.WinRate >= 0.7 {
			matchup.Dominance = "dominant"
		} else if matchup.WinRate >= 0.3 {
			matchup.Dominance = "competitive"
		} else {
			matchup.Dominance = "struggling"
		}
		
		matchups = append(matchups, matchup)
	}

	return matchups
}

// Additional helper methods for game analytics would be implemented here
// These are simplified stubs for the core structure

func (r *AnalyticsRepository) getGameDurationDistribution(threshold time.Time) map[string]int {
	// Implementation would categorize games by duration
	return make(map[string]int)
}

func (r *AnalyticsRepository) getPlayerCountDistribution(threshold time.Time) map[int]int {
	// Implementation would show distribution of player counts
	return make(map[int]int)
}

func (r *AnalyticsRepository) GetMapAnalytics(threshold time.Time) map[string]MapAnalytics {
	// Implementation would analyze each map's performance
	return make(map[string]MapAnalytics)
}

func (r *AnalyticsRepository) getPeakPlayTimes(threshold time.Time) map[int]int {
	// Implementation would show peak hours
	return make(map[int]int)
}

func (r *AnalyticsRepository) getSeasonalTrends(timeFrameDays int) []SeasonalTrendPoint {
	// Implementation would show trends over time
	return []SeasonalTrendPoint{}
}

func (r *AnalyticsRepository) getCompetitivenessMetrics(threshold time.Time) CompetitivenessAnalysis {
	// Implementation would calculate competitiveness metrics
	return CompetitivenessAnalysis{}
}

func (r *AnalyticsRepository) getPowerPlantAnalytics(threshold time.Time) PowerPlantMarketAnalysis {
	// Implementation would analyze power plant market
	return PowerPlantMarketAnalysis{}
}

func (r *AnalyticsRepository) getResourceMarketAnalytics(threshold time.Time) ResourceMarketAnalysis {
	// Implementation would analyze resource market
	return ResourceMarketAnalysis{}
}

func (r *AnalyticsRepository) getStrategyAnalysis(threshold time.Time) StrategyEffectivenessAnalysis {
	// Implementation would analyze strategy effectiveness
	return StrategyEffectivenessAnalysis{}
}