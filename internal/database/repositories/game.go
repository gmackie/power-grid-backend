package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"powergrid/internal/analytics/models"
	"powergrid/pkg/logger"
)

// GameRepository handles database operations for games
type GameRepository struct {
	db     *sql.DB
	logger *logger.ColoredLogger
}

// NewGameRepository creates a new game repository
func NewGameRepository(db *sql.DB) *GameRepository {
	return &GameRepository{
		db:     db,
		logger: logger.CreateAILogger("GameRepo", logger.ColorCyan),
	}
}

// CreateGame creates a new game record
func (r *GameRepository) CreateGame(gameID, name, mapName string, maxPlayers, actualPlayers int) (*models.GameRecord, error) {
	query := `
		INSERT INTO games (game_id, name, map_name, max_players, actual_players, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, 'lobby', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		RETURNING id, game_id, name, map_name, max_players, actual_players, status, 
				  winner_player_id, started_at, ended_at, duration_minutes, total_rounds, created_at, updated_at
	`
	
	var game models.GameRecord
	err := r.db.QueryRow(query, gameID, name, mapName, maxPlayers, actualPlayers).Scan(
		&game.ID,
		&game.GameID,
		&game.Name,
		&game.MapName,
		&game.MaxPlayers,
		&game.ActualPlayers,
		&game.Status,
		&game.WinnerPlayerID,
		&game.StartedAt,
		&game.EndedAt,
		&game.DurationMinutes,
		&game.TotalRounds,
		&game.CreatedAt,
		&game.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create game %s: %w", gameID, err)
	}
	
	r.logger.Debug("Created game: %s (ID: %d)", game.GameID, game.ID)
	return &game, nil
}

// GetGame retrieves a game by game ID
func (r *GameRepository) GetGame(gameID string) (*models.GameRecord, error) {
	query := `
		SELECT id, game_id, name, map_name, max_players, actual_players, status,
			   winner_player_id, started_at, ended_at, duration_minutes, total_rounds, created_at, updated_at
		FROM games WHERE game_id = ?
	`
	
	var game models.GameRecord
	err := r.db.QueryRow(query, gameID).Scan(
		&game.ID,
		&game.GameID,
		&game.Name,
		&game.MapName,
		&game.MaxPlayers,
		&game.ActualPlayers,
		&game.Status,
		&game.WinnerPlayerID,
		&game.StartedAt,
		&game.EndedAt,
		&game.DurationMinutes,
		&game.TotalRounds,
		&game.CreatedAt,
		&game.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("game not found: %s", gameID)
		}
		return nil, fmt.Errorf("failed to get game %s: %w", gameID, err)
	}
	
	return &game, nil
}

// UpdateGameStatus updates the game status and timestamps
func (r *GameRepository) UpdateGameStatus(gameID string, status string, winnerPlayerID *int) error {
	var query string
	var args []interface{}
	
	if status == "playing" {
		query = `UPDATE games SET status = ?, started_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE game_id = ?`
		args = []interface{}{status, gameID}
	} else if status == "completed" {
		query = `UPDATE games SET status = ?, ended_at = CURRENT_TIMESTAMP, winner_player_id = ?, updated_at = CURRENT_TIMESTAMP WHERE game_id = ?`
		args = []interface{}{status, winnerPlayerID, gameID}
	} else {
		query = `UPDATE games SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE game_id = ?`
		args = []interface{}{status, gameID}
	}
	
	_, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update game status for %s: %w", gameID, err)
	}
	
	r.logger.Debug("Updated game %s status to %s", gameID, status)
	return nil
}

// UpdateGameRounds updates the total rounds for a game
func (r *GameRepository) UpdateGameRounds(gameID string, totalRounds int) error {
	query := `UPDATE games SET total_rounds = ?, updated_at = CURRENT_TIMESTAMP WHERE game_id = ?`
	
	_, err := r.db.Exec(query, totalRounds, gameID)
	if err != nil {
		return fmt.Errorf("failed to update game rounds for %s: %w", gameID, err)
	}
	
	return nil
}

// AddGameParticipant adds a player to a game
func (r *GameRepository) AddGameParticipant(gameID string, playerID int, playerName, color string, turnOrder int) error {
	// First get the internal game ID
	var internalGameID int
	err := r.db.QueryRow("SELECT id FROM games WHERE game_id = ?", gameID).Scan(&internalGameID)
	if err != nil {
		return fmt.Errorf("failed to find game %s: %w", gameID, err)
	}
	
	query := `
		INSERT INTO game_participants (game_id, player_id, player_name, color, turn_order, joined_at)
		VALUES (?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(game_id, player_id) DO UPDATE SET
			color = EXCLUDED.color,
			turn_order = EXCLUDED.turn_order
	`
	
	_, err = r.db.Exec(query, internalGameID, playerID, playerName, color, turnOrder)
	if err != nil {
		return fmt.Errorf("failed to add participant to game %s: %w", gameID, err)
	}
	
	r.logger.Debug("Added participant %s to game %s", playerName, gameID)
	return nil
}

// UpdateGameParticipantResults updates final results for a game participant
func (r *GameRepository) UpdateGameParticipantResults(gameID string, playerID int, finalPosition, finalCities, finalPlants, finalMoney, finalResources, poweredCities int, isWinner bool) error {
	// Get internal game ID
	var internalGameID int
	err := r.db.QueryRow("SELECT id FROM games WHERE game_id = ?", gameID).Scan(&internalGameID)
	if err != nil {
		return fmt.Errorf("failed to find game %s: %w", gameID, err)
	}
	
	query := `
		UPDATE game_participants 
		SET final_position = ?, final_cities = ?, final_plants = ?, final_money = ?,
			final_resources = ?, powered_cities = ?, is_winner = ?
		WHERE game_id = ? AND player_id = ?
	`
	
	_, err = r.db.Exec(query, finalPosition, finalCities, finalPlants, finalMoney,
		finalResources, poweredCities, isWinner, internalGameID, playerID)
	if err != nil {
		return fmt.Errorf("failed to update participant results for game %s, player %d: %w", gameID, playerID, err)
	}
	
	r.logger.Debug("Updated results for player %d in game %s", playerID, gameID)
	return nil
}

// GetGameParticipants retrieves all participants for a game
func (r *GameRepository) GetGameParticipants(gameID string) ([]*models.GameParticipant, error) {
	query := `
		SELECT gp.player_id, gp.player_name, gp.color, gp.turn_order,
			   gp.final_position, gp.final_cities, gp.final_plants, gp.final_money,
			   gp.final_resources, gp.powered_cities, gp.is_winner, gp.joined_at
		FROM game_participants gp
		JOIN games g ON gp.game_id = g.id
		WHERE g.game_id = ?
		ORDER BY gp.turn_order
	`
	
	rows, err := r.db.Query(query, gameID)
	if err != nil {
		return nil, fmt.Errorf("failed to query game participants for %s: %w", gameID, err)
	}
	defer rows.Close()
	
	var participants []*models.GameParticipant
	for rows.Next() {
		var p models.GameParticipant
		err := rows.Scan(
			&p.PlayerID,
			&p.PlayerName,
			&p.Color,
			&p.TurnOrder,
			&p.FinalPosition,
			&p.FinalCities,
			&p.FinalPlants,
			&p.FinalMoney,
			&p.FinalResources,
			&p.PoweredCities,
			&p.IsWinner,
			&p.JoinedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game participant: %w", err)
		}
		participants = append(participants, &p)
	}
	
	return participants, nil
}

// GetRecentGames retrieves recent games with optional filters
func (r *GameRepository) GetRecentGames(limit int, status string) ([]*models.GameRecord, error) {
	var query string
	var args []interface{}
	
	if status != "" {
		query = `
			SELECT id, game_id, name, map_name, max_players, actual_players, status,
				   winner_player_id, started_at, ended_at, duration_minutes, total_rounds, created_at, updated_at
			FROM games 
			WHERE status = ?
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{status, limit}
	} else {
		query = `
			SELECT id, game_id, name, map_name, max_players, actual_players, status,
				   winner_player_id, started_at, ended_at, duration_minutes, total_rounds, created_at, updated_at
			FROM games 
			ORDER BY created_at DESC
			LIMIT ?
		`
		args = []interface{}{limit}
	}
	
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query recent games: %w", err)
	}
	defer rows.Close()
	
	var games []*models.GameRecord
	for rows.Next() {
		var game models.GameRecord
		err := rows.Scan(
			&game.ID,
			&game.GameID,
			&game.Name,
			&game.MapName,
			&game.MaxPlayers,
			&game.ActualPlayers,
			&game.Status,
			&game.WinnerPlayerID,
			&game.StartedAt,
			&game.EndedAt,
			&game.DurationMinutes,
			&game.TotalRounds,
			&game.CreatedAt,
			&game.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan game: %w", err)
		}
		games = append(games, &game)
	}
	
	return games, nil
}

// LogGameEvent logs a game event
func (r *GameRepository) LogGameEvent(gameID string, playerID *int, eventType string, eventData interface{}, roundNumber *int, phase string) error {
	// Get internal game ID
	var internalGameID int
	err := r.db.QueryRow("SELECT id FROM games WHERE game_id = ?", gameID).Scan(&internalGameID)
	if err != nil {
		return fmt.Errorf("failed to find game %s: %w", gameID, err)
	}
	
	// Serialize event data to JSON
	var eventDataJSON *string
	if eventData != nil {
		data, err := json.Marshal(eventData)
		if err != nil {
			return fmt.Errorf("failed to marshal event data: %w", err)
		}
		jsonStr := string(data)
		eventDataJSON = &jsonStr
	}
	
	query := `
		INSERT INTO game_events (game_id, player_id, event_type, event_data, round_number, phase, created_at)
		VALUES (?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
	`
	
	_, err = r.db.Exec(query, internalGameID, playerID, eventType, eventDataJSON, roundNumber, phase)
	if err != nil {
		return fmt.Errorf("failed to log game event: %w", err)
	}
	
	r.logger.Debug("Logged event %s for game %s", eventType, gameID)
	return nil
}

// GetGameAnalytics retrieves analytics for games
func (r *GameRepository) GetGameAnalytics(days int) (*models.GameAnalytics, error) {
	analytics := &models.GameAnalytics{
		MapPopularity: make(map[string]int),
		PlayerCounts:  make(map[int]int),
	}
	
	// Calculate date threshold
	threshold := time.Now().AddDate(0, 0, -days)
	
	// Get total games
	err := r.db.QueryRow("SELECT COUNT(*) FROM games WHERE created_at >= ?", threshold).Scan(&analytics.TotalGames)
	if err != nil {
		return nil, fmt.Errorf("failed to get total games: %w", err)
	}
	
	// Get completed games
	err = r.db.QueryRow("SELECT COUNT(*) FROM games WHERE status = 'completed' AND created_at >= ?", threshold).Scan(&analytics.CompletedGames)
	if err != nil {
		return nil, fmt.Errorf("failed to get completed games: %w", err)
	}
	
	// Get average duration
	err = r.db.QueryRow("SELECT COALESCE(AVG(duration_minutes), 0) FROM games WHERE status = 'completed' AND created_at >= ?", threshold).Scan(&analytics.AvgGameDuration)
	if err != nil {
		return nil, fmt.Errorf("failed to get average duration: %w", err)
	}
	
	// Get map popularity
	rows, err := r.db.Query("SELECT map_name, COUNT(*) FROM games WHERE created_at >= ? GROUP BY map_name", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get map popularity: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var mapName string
		var count int
		if err := rows.Scan(&mapName, &count); err != nil {
			return nil, fmt.Errorf("failed to scan map popularity: %w", err)
		}
		analytics.MapPopularity[mapName] = count
	}
	
	// Get player count distribution
	rows, err = r.db.Query("SELECT actual_players, COUNT(*) FROM games WHERE created_at >= ? GROUP BY actual_players", threshold)
	if err != nil {
		return nil, fmt.Errorf("failed to get player counts: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var playerCount, gameCount int
		if err := rows.Scan(&playerCount, &gameCount); err != nil {
			return nil, fmt.Errorf("failed to scan player counts: %w", err)
		}
		analytics.PlayerCounts[playerCount] = gameCount
	}
	
	return analytics, nil
}

// GetGameCount returns the total number of games
func (r *GameRepository) GetGameCount() (int, error) {
	var count int
	err := r.db.QueryRow("SELECT COUNT(*) FROM games").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get game count: %w", err)
	}
	return count, nil
}