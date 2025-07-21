package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"powergrid/internal/analytics/models"
	"powergrid/pkg/logger"
)

// PlayerRepository handles database operations for players
type PlayerRepository struct {
	db     *sql.DB
	logger *logger.ColoredLogger
}

// NewPlayerRepository creates a new player repository
func NewPlayerRepository(db *sql.DB) *PlayerRepository {
	return &PlayerRepository{
		db:     db,
		logger: logger.CreateAILogger("PlayerRepo", logger.ColorBrightPurple),
	}
}

// CreateOrUpdatePlayer creates a new player or updates existing one
func (r *PlayerRepository) CreateOrUpdatePlayer(name string) (*models.Player, error) {
	query := `
		INSERT INTO players (name, first_seen, last_seen, updated_at)
		VALUES (?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT(name) DO UPDATE SET
			last_seen = CURRENT_TIMESTAMP,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, name, first_seen, last_seen, total_games, total_wins, 
				  total_playtime_minutes, favorite_map, preferred_color, created_at, updated_at
	`
	
	var player models.Player
	err := r.db.QueryRow(query, name).Scan(
		&player.ID,
		&player.Name,
		&player.FirstSeen,
		&player.LastSeen,
		&player.TotalGames,
		&player.TotalWins,
		&player.TotalPlaytimeMinutes,
		&player.FavoriteMap,
		&player.PreferredColor,
		&player.CreatedAt,
		&player.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create/update player %s: %w", name, err)
	}
	
	r.logger.Debug("Created/updated player: %s (ID: %d)", player.Name, player.ID)
	return &player, nil
}

// GetPlayer retrieves a player by name
func (r *PlayerRepository) GetPlayer(name string) (*models.Player, error) {
	query := `
		SELECT id, name, first_seen, last_seen, total_games, total_wins,
			   total_playtime_minutes, favorite_map, preferred_color, created_at, updated_at
		FROM players WHERE name = ?
	`
	
	var player models.Player
	err := r.db.QueryRow(query, name).Scan(
		&player.ID,
		&player.Name,
		&player.FirstSeen,
		&player.LastSeen,
		&player.TotalGames,
		&player.TotalWins,
		&player.TotalPlaytimeMinutes,
		&player.FavoriteMap,
		&player.PreferredColor,
		&player.CreatedAt,
		&player.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found: %s", name)
		}
		return nil, fmt.Errorf("failed to get player %s: %w", name, err)
	}
	
	return &player, nil
}

// GetPlayerByID retrieves a player by ID
func (r *PlayerRepository) GetPlayerByID(id int) (*models.Player, error) {
	query := `
		SELECT id, name, first_seen, last_seen, total_games, total_wins,
			   total_playtime_minutes, favorite_map, preferred_color, created_at, updated_at
		FROM players WHERE id = ?
	`
	
	var player models.Player
	err := r.db.QueryRow(query, id).Scan(
		&player.ID,
		&player.Name,
		&player.FirstSeen,
		&player.LastSeen,
		&player.TotalGames,
		&player.TotalWins,
		&player.TotalPlaytimeMinutes,
		&player.FavoriteMap,
		&player.PreferredColor,
		&player.CreatedAt,
		&player.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("player not found with ID: %d", id)
		}
		return nil, fmt.Errorf("failed to get player %d: %w", id, err)
	}
	
	return &player, nil
}

// GetAllPlayers retrieves all players with optional limit and offset
func (r *PlayerRepository) GetAllPlayers(limit, offset int) ([]*models.Player, error) {
	query := `
		SELECT id, name, first_seen, last_seen, total_games, total_wins,
			   total_playtime_minutes, favorite_map, preferred_color, created_at, updated_at
		FROM players 
		ORDER BY last_seen DESC
		LIMIT ? OFFSET ?
	`
	
	rows, err := r.db.Query(query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query players: %w", err)
	}
	defer rows.Close()
	
	var players []*models.Player
	for rows.Next() {
		var player models.Player
		err := rows.Scan(
			&player.ID,
			&player.Name,
			&player.FirstSeen,
			&player.LastSeen,
			&player.TotalGames,
			&player.TotalWins,
			&player.TotalPlaytimeMinutes,
			&player.FavoriteMap,
			&player.PreferredColor,
			&player.CreatedAt,
			&player.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player: %w", err)
		}
		players = append(players, &player)
	}
	
	return players, nil
}

// GetPlayerStats retrieves aggregated statistics for a player
func (r *PlayerRepository) GetPlayerStats(playerID int) (*models.PlayerStats, error) {
	query := `
		SELECT 
			player_id,
			games_played,
			games_won,
			games_lost,
			win_rate,
			avg_final_cities,
			avg_final_plants,
			avg_final_money,
			avg_game_duration_minutes,
			max_cities_single_game,
			max_plants_single_game,
			max_money_single_game,
			fastest_win_minutes,
			longest_win_minutes,
			total_resources_bought,
			total_money_spent,
			total_plants_owned,
			avg_plant_efficiency,
			total_cities_built,
			avg_expansion_rate,
			total_achievement_points,
			total_achievements_earned,
			total_playtime_minutes,
			avg_session_length_minutes,
			last_updated
		FROM player_statistics 
		WHERE player_id = ?
	`
	
	var stats models.PlayerStats
	err := r.db.QueryRow(query, playerID).Scan(
		&stats.PlayerID,
		&stats.GamesPlayed,
		&stats.GamesWon,
		&stats.GamesLost,
		&stats.WinRate,
		&stats.AvgFinalCities,
		&stats.AvgFinalPlants,
		&stats.AvgFinalMoney,
		&stats.AvgGameDurationMinutes,
		&stats.MaxCitiesSingleGame,
		&stats.MaxPlantsSingleGame,
		&stats.MaxMoneySingleGame,
		&stats.FastestWinMinutes,
		&stats.LongestWinMinutes,
		&stats.TotalResourcesBought,
		&stats.TotalMoneySpent,
		&stats.TotalPlantsOwned,
		&stats.AvgPlantEfficiency,
		&stats.TotalCitiesBuilt,
		&stats.AvgExpansionRate,
		&stats.TotalAchievementPoints,
		&stats.TotalAchievementsEarned,
		&stats.TotalPlaytimeMinutes,
		&stats.AvgSessionLengthMinutes,
		&stats.LastUpdated,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// Return empty stats for player without recorded games
			return &models.PlayerStats{
				PlayerID:    playerID,
				LastUpdated: time.Now(),
			}, nil
		}
		return nil, fmt.Errorf("failed to get player stats for %d: %w", playerID, err)
	}
	
	return &stats, nil
}

// UpdatePlayerMetrics updates specific player metrics
func (r *PlayerRepository) UpdatePlayerMetrics(playerID int, totalGames, totalWins int, totalPlaytime time.Duration, favoriteMap, preferredColor string) error {
	query := `
		UPDATE players 
		SET total_games = ?, 
			total_wins = ?, 
			total_playtime_minutes = ?,
			favorite_map = COALESCE(?, favorite_map),
			preferred_color = COALESCE(?, preferred_color),
			updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`
	
	_, err := r.db.Exec(query, totalGames, totalWins, int(totalPlaytime.Minutes()), 
		favoriteMap, preferredColor, playerID)
	if err != nil {
		return fmt.Errorf("failed to update player metrics for %d: %w", playerID, err)
	}
	
	r.logger.Debug("Updated metrics for player ID %d", playerID)
	return nil
}

// GetLeaderboard retrieves the player leaderboard
func (r *PlayerRepository) GetLeaderboard(limit int) ([]*models.LeaderboardEntry, error) {
	query := `
		SELECT 
			p.id,
			p.name,
			COALESCE(ps.games_played, 0) as games_played,
			COALESCE(ps.games_won, 0) as games_won,
			COALESCE(ps.win_rate, 0.0) as win_rate,
			COALESCE(ps.avg_final_cities, 0.0) as avg_final_cities,
			COALESCE(ps.total_achievement_points, 0) as total_achievement_points,
			COALESCE(ps.total_cities_built, 0) as total_cities_built,
			(COALESCE(ps.games_won, 0) * 100 + COALESCE(ps.win_rate, 0) * 10 + 
			 COALESCE(ps.total_cities_built, 0) + COALESCE(ps.total_achievement_points, 0)) as composite_score,
			p.last_seen
		FROM players p
		LEFT JOIN player_statistics ps ON p.id = ps.player_id
		WHERE COALESCE(ps.games_played, 0) >= 1
		ORDER BY composite_score DESC
		LIMIT ?
	`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query leaderboard: %w", err)
	}
	defer rows.Close()
	
	var entries []*models.LeaderboardEntry
	for rows.Next() {
		var entry models.LeaderboardEntry
		err := rows.Scan(
			&entry.PlayerID,
			&entry.PlayerName,
			&entry.GamesPlayed,
			&entry.GamesWon,
			&entry.WinRate,
			&entry.AvgFinalCities,
			&entry.TotalAchievementPoints,
			&entry.TotalCitiesBuilt,
			&entry.CompositeScore,
			&entry.LastSeen,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	
	return entries, nil
}

// GetPlayerCount returns the total number of players
func (r *PlayerRepository) GetPlayerCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM players").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get player count: %w", err)
	}
	return count, nil
}