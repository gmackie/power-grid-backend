package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"powergrid/internal/analytics/models"
	"powergrid/pkg/logger"
)

// AchievementRepository handles database operations for achievements
type AchievementRepository struct {
	db     *sql.DB
	logger *logger.ColoredLogger
}

// NewAchievementRepository creates a new achievement repository
func NewAchievementRepository(db *sql.DB) *AchievementRepository {
	return &AchievementRepository{
		db:     db,
		logger: logger.CreateAILogger("AchievementRepo", logger.ColorYellow),
	}
}

// CreateAchievement creates a new achievement definition
func (r *AchievementRepository) CreateAchievement(achievementID, name, description, category, icon, criteria string, points int) (*models.Achievement, error) {
	query := `
		INSERT INTO achievements (achievement_id, name, description, category, icon, points, criteria, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		RETURNING id, achievement_id, name, description, category, icon, points, criteria, is_active, created_at
	`
	
	var achievement models.Achievement
	err := r.db.QueryRow(query, achievementID, name, description, category, icon, points, criteria).Scan(
		&achievement.ID,
		&achievement.AchievementID,
		&achievement.Name,
		&achievement.Description,
		&achievement.Category,
		&achievement.Icon,
		&achievement.Points,
		&achievement.Criteria,
		&achievement.IsActive,
		&achievement.CreatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create achievement %s: %w", achievementID, err)
	}
	
	r.logger.Debug("Created achievement: %s (ID: %d)", achievement.AchievementID, achievement.ID)
	return &achievement, nil
}

// GetAchievement retrieves an achievement by achievement ID
func (r *AchievementRepository) GetAchievement(achievementID string) (*models.Achievement, error) {
	query := `
		SELECT id, achievement_id, name, description, category, icon, points, criteria, is_active, created_at
		FROM achievements WHERE achievement_id = ?
	`
	
	var achievement models.Achievement
	err := r.db.QueryRow(query, achievementID).Scan(
		&achievement.ID,
		&achievement.AchievementID,
		&achievement.Name,
		&achievement.Description,
		&achievement.Category,
		&achievement.Icon,
		&achievement.Points,
		&achievement.Criteria,
		&achievement.IsActive,
		&achievement.CreatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("achievement not found: %s", achievementID)
		}
		return nil, fmt.Errorf("failed to get achievement %s: %w", achievementID, err)
	}
	
	return &achievement, nil
}

// GetAllAchievements retrieves all active achievements
func (r *AchievementRepository) GetAllAchievements() ([]*models.Achievement, error) {
	query := `
		SELECT id, achievement_id, name, description, category, icon, points, criteria, is_active, created_at
		FROM achievements 
		WHERE is_active = TRUE
		ORDER BY category, points DESC
	`
	
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query achievements: %w", err)
	}
	defer rows.Close()
	
	var achievements []*models.Achievement
	for rows.Next() {
		var achievement models.Achievement
		err := rows.Scan(
			&achievement.ID,
			&achievement.AchievementID,
			&achievement.Name,
			&achievement.Description,
			&achievement.Category,
			&achievement.Icon,
			&achievement.Points,
			&achievement.Criteria,
			&achievement.IsActive,
			&achievement.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan achievement: %w", err)
		}
		achievements = append(achievements, &achievement)
	}
	
	return achievements, nil
}

// GetAchievementsByCategory retrieves achievements by category
func (r *AchievementRepository) GetAchievementsByCategory(category string) ([]*models.Achievement, error) {
	query := `
		SELECT id, achievement_id, name, description, category, icon, points, criteria, is_active, created_at
		FROM achievements 
		WHERE category = ? AND is_active = TRUE
		ORDER BY points DESC
	`
	
	rows, err := r.db.Query(query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to query achievements by category: %w", err)
	}
	defer rows.Close()
	
	var achievements []*models.Achievement
	for rows.Next() {
		var achievement models.Achievement
		err := rows.Scan(
			&achievement.ID,
			&achievement.AchievementID,
			&achievement.Name,
			&achievement.Description,
			&achievement.Category,
			&achievement.Icon,
			&achievement.Points,
			&achievement.Criteria,
			&achievement.IsActive,
			&achievement.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan achievement: %w", err)
		}
		achievements = append(achievements, &achievement)
	}
	
	return achievements, nil
}

// CreatePlayerAchievement creates or updates player achievement progress
func (r *AchievementRepository) CreatePlayerAchievement(playerID, achievementInternalID int, gameID *int, progress, maxProgress int) (*models.PlayerAchievement, error) {
	isCompleted := progress >= maxProgress
	var completedAt *time.Time
	if isCompleted {
		now := time.Now()
		completedAt = &now
	}
	
	query := `
		INSERT INTO player_achievements (player_id, achievement_id, game_id, progress, max_progress, is_completed, completed_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(player_id, achievement_id) DO UPDATE SET
			progress = EXCLUDED.progress,
			max_progress = EXCLUDED.max_progress,
			is_completed = EXCLUDED.is_completed,
			completed_at = EXCLUDED.completed_at,
			updated_at = CURRENT_TIMESTAMP
		RETURNING id, player_id, achievement_id, game_id, progress, max_progress, is_completed, completed_at, created_at, updated_at
	`
	
	var achievement models.PlayerAchievement
	err := r.db.QueryRow(query, playerID, achievementInternalID, gameID, progress, maxProgress, isCompleted, completedAt).Scan(
		&achievement.ID,
		&achievement.PlayerID,
		&achievement.AchievementID,
		&achievement.GameID,
		&achievement.Progress,
		&achievement.MaxProgress,
		&achievement.IsCompleted,
		&achievement.CompletedAt,
		&achievement.CreatedAt,
		&achievement.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("failed to create/update player achievement: %w", err)
	}
	
	r.logger.Debug("Updated achievement progress for player %d, achievement %d (%d/%d)", 
		playerID, achievementInternalID, progress, maxProgress)
	return &achievement, nil
}

// GetPlayerAchievements retrieves all achievements for a player
func (r *AchievementRepository) GetPlayerAchievements(playerID int) ([]*models.PlayerAchievement, error) {
	query := `
		SELECT pa.id, pa.player_id, pa.achievement_id, pa.game_id, pa.progress, pa.max_progress,
			   pa.is_completed, pa.completed_at, pa.created_at, pa.updated_at,
			   a.achievement_id, a.name, a.description, a.category, a.icon, a.points, a.criteria
		FROM player_achievements pa
		JOIN achievements a ON pa.achievement_id = a.id
		WHERE pa.player_id = ?
		ORDER BY pa.is_completed DESC, a.category, a.points DESC
	`
	
	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query player achievements: %w", err)
	}
	defer rows.Close()
	
	var achievements []*models.PlayerAchievement
	for rows.Next() {
		var pa models.PlayerAchievement
		var achievement models.Achievement
		
		err := rows.Scan(
			&pa.ID,
			&pa.PlayerID,
			&pa.AchievementID,
			&pa.GameID,
			&pa.Progress,
			&pa.MaxProgress,
			&pa.IsCompleted,
			&pa.CompletedAt,
			&pa.CreatedAt,
			&pa.UpdatedAt,
			&achievement.AchievementID,
			&achievement.Name,
			&achievement.Description,
			&achievement.Category,
			&achievement.Icon,
			&achievement.Points,
			&achievement.Criteria,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan player achievement: %w", err)
		}
		
		// Attach achievement details
		pa.Achievement = &achievement
		achievements = append(achievements, &pa)
	}
	
	return achievements, nil
}

// GetPlayerCompletedAchievements retrieves only completed achievements for a player
func (r *AchievementRepository) GetPlayerCompletedAchievements(playerID int) ([]*models.PlayerAchievement, error) {
	query := `
		SELECT pa.id, pa.player_id, pa.achievement_id, pa.game_id, pa.progress, pa.max_progress,
			   pa.is_completed, pa.completed_at, pa.created_at, pa.updated_at,
			   a.achievement_id, a.name, a.description, a.category, a.icon, a.points, a.criteria
		FROM player_achievements pa
		JOIN achievements a ON pa.achievement_id = a.id
		WHERE pa.player_id = ? AND pa.is_completed = TRUE
		ORDER BY pa.completed_at DESC
	`
	
	rows, err := r.db.Query(query, playerID)
	if err != nil {
		return nil, fmt.Errorf("failed to query completed achievements: %w", err)
	}
	defer rows.Close()
	
	var achievements []*models.PlayerAchievement
	for rows.Next() {
		var pa models.PlayerAchievement
		var achievement models.Achievement
		
		err := rows.Scan(
			&pa.ID,
			&pa.PlayerID,
			&pa.AchievementID,
			&pa.GameID,
			&pa.Progress,
			&pa.MaxProgress,
			&pa.IsCompleted,
			&pa.CompletedAt,
			&pa.CreatedAt,
			&pa.UpdatedAt,
			&achievement.AchievementID,
			&achievement.Name,
			&achievement.Description,
			&achievement.Category,
			&achievement.Icon,
			&achievement.Points,
			&achievement.Criteria,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan completed achievement: %w", err)
		}
		
		pa.Achievement = &achievement
		achievements = append(achievements, &pa)
	}
	
	return achievements, nil
}

// UpdateAchievementProgress updates progress for a specific player achievement
func (r *AchievementRepository) UpdateAchievementProgress(playerID, achievementInternalID, progress int) error {
	// Check if this would complete the achievement
	var maxProgress int
	err := r.db.QueryRow("SELECT max_progress FROM player_achievements WHERE player_id = ? AND achievement_id = ?", 
		playerID, achievementInternalID).Scan(&maxProgress)
	if err != nil {
		return fmt.Errorf("failed to get max progress: %w", err)
	}
	
	isCompleted := progress >= maxProgress
	var query string
	var args []interface{}
	
	if isCompleted {
		query = `
			UPDATE player_achievements 
			SET progress = ?, is_completed = TRUE, completed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
			WHERE player_id = ? AND achievement_id = ? AND is_completed = FALSE
		`
		args = []interface{}{progress, playerID, achievementInternalID}
	} else {
		query = `
			UPDATE player_achievements 
			SET progress = ?, updated_at = CURRENT_TIMESTAMP
			WHERE player_id = ? AND achievement_id = ?
		`
		args = []interface{}{progress, playerID, achievementInternalID}
	}
	
	result, err := r.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update achievement progress: %w", err)
	}
	
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 && isCompleted {
		r.logger.Info("Player %d completed achievement %d!", playerID, achievementInternalID)
	}
	
	return nil
}

// GetAchievementStats retrieves achievement statistics
func (r *AchievementRepository) GetAchievementStats() (*models.AchievementStats, error) {
	stats := &models.AchievementStats{
		CategoryStats: make(map[string]models.CategoryStats),
	}
	
	// Get total achievements
	err := r.db.QueryRow("SELECT COUNT(*) FROM achievements WHERE is_active = TRUE").Scan(&stats.TotalAchievements)
	if err != nil {
		return nil, fmt.Errorf("failed to get total achievements: %w", err)
	}
	
	// Get total completions
	err = r.db.QueryRow("SELECT COUNT(*) FROM player_achievements WHERE is_completed = TRUE").Scan(&stats.TotalCompletions)
	if err != nil {
		return nil, fmt.Errorf("failed to get total completions: %w", err)
	}
	
	// Get unique players with achievements
	err = r.db.QueryRow("SELECT COUNT(DISTINCT player_id) FROM player_achievements WHERE is_completed = TRUE").Scan(&stats.PlayersWithAchievements)
	if err != nil {
		return nil, fmt.Errorf("failed to get players with achievements: %w", err)
	}
	
	// Get rarest achievement
	query := `
		SELECT a.name, COUNT(pa.id) as completion_count
		FROM achievements a
		LEFT JOIN player_achievements pa ON a.id = pa.achievement_id AND pa.is_completed = TRUE
		WHERE a.is_active = TRUE
		GROUP BY a.id, a.name
		ORDER BY completion_count ASC, a.points DESC
		LIMIT 1
	`
	err = r.db.QueryRow(query).Scan(&stats.RarestAchievement, &stats.RarestCompletionCount)
	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get rarest achievement: %w", err)
	}
	
	// Get category statistics
	categoryQuery := `
		SELECT 
			a.category,
			COUNT(a.id) as total_in_category,
			COUNT(CASE WHEN pa.is_completed = TRUE THEN 1 END) as completed_in_category,
			SUM(a.points) as total_points_available,
			SUM(CASE WHEN pa.is_completed = TRUE THEN a.points ELSE 0 END) as points_earned
		FROM achievements a
		LEFT JOIN player_achievements pa ON a.id = pa.achievement_id
		WHERE a.is_active = TRUE
		GROUP BY a.category
	`
	
	rows, err := r.db.Query(categoryQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to query category stats: %w", err)
	}
	defer rows.Close()
	
	for rows.Next() {
		var category string
		var catStats models.CategoryStats
		err := rows.Scan(
			&category,
			&catStats.TotalAchievements,
			&catStats.TotalCompletions,
			&catStats.TotalPointsAvailable,
			&catStats.PointsEarned,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan category stats: %w", err)
		}
		
		if catStats.TotalAchievements > 0 {
			catStats.CompletionRate = float64(catStats.TotalCompletions) / float64(catStats.TotalAchievements) * 100
		}
		
		stats.CategoryStats[category] = catStats
	}
	
	return stats, nil
}

// GetAchievementLeaderboard retrieves top players by achievement points
func (r *AchievementRepository) GetAchievementLeaderboard(limit int) ([]*models.AchievementLeaderboardEntry, error) {
	query := `
		SELECT 
			p.id,
			p.name,
			COUNT(CASE WHEN pa.is_completed = TRUE THEN 1 END) as achievements_earned,
			SUM(CASE WHEN pa.is_completed = TRUE THEN a.points ELSE 0 END) as total_points,
			MAX(pa.completed_at) as last_achievement_at
		FROM players p
		LEFT JOIN player_achievements pa ON p.id = pa.player_id
		LEFT JOIN achievements a ON pa.achievement_id = a.id
		GROUP BY p.id, p.name
		HAVING total_points > 0
		ORDER BY total_points DESC, achievements_earned DESC, last_achievement_at DESC
		LIMIT ?
	`
	
	rows, err := r.db.Query(query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query achievement leaderboard: %w", err)
	}
	defer rows.Close()
	
	var entries []*models.AchievementLeaderboardEntry
	for rows.Next() {
		var entry models.AchievementLeaderboardEntry
		err := rows.Scan(
			&entry.PlayerID,
			&entry.PlayerName,
			&entry.AchievementsEarned,
			&entry.TotalPoints,
			&entry.LastAchievementAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan leaderboard entry: %w", err)
		}
		entries = append(entries, &entry)
	}
	
	return entries, nil
}