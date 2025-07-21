package models

import (
	"time"
)

// Achievement represents an achievement definition
type Achievement struct {
	ID            int       `json:"id" db:"id"`
	AchievementID string    `json:"achievement_id" db:"achievement_id"`
	Name          string    `json:"name" db:"name"`
	Description   string    `json:"description" db:"description"`
	Category      string    `json:"category" db:"category"`
	Icon          *string   `json:"icon" db:"icon"`
	Points        int       `json:"points" db:"points"`
	Criteria      *string   `json:"criteria" db:"criteria"`
	IsActive      bool      `json:"is_active" db:"is_active"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
}

// PlayerAchievement represents a player's progress on an achievement
type PlayerAchievement struct {
	ID            int          `json:"id" db:"id"`
	PlayerID      int          `json:"player_id" db:"player_id"`
	AchievementID int          `json:"achievement_id" db:"achievement_id"`
	GameID        *int         `json:"game_id" db:"game_id"`
	Progress      int          `json:"progress" db:"progress"`
	MaxProgress   int          `json:"max_progress" db:"max_progress"`
	IsCompleted   bool         `json:"is_completed" db:"is_completed"`
	CompletedAt   *time.Time   `json:"completed_at" db:"completed_at"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
	
	// Associated achievement details (populated via JOIN)
	Achievement   *Achievement `json:"achievement,omitempty"`
}

// AchievementStats represents aggregate achievement statistics
type AchievementStats struct {
	TotalAchievements       int                       `json:"total_achievements"`
	TotalCompletions        int                       `json:"total_completions"`
	PlayersWithAchievements int                       `json:"players_with_achievements"`
	RarestAchievement       string                    `json:"rarest_achievement"`
	RarestCompletionCount   int                       `json:"rarest_completion_count"`
	CategoryStats           map[string]CategoryStats  `json:"category_stats"`
}

// CategoryStats represents statistics for an achievement category
type CategoryStats struct {
	TotalAchievements     int     `json:"total_achievements"`
	TotalCompletions      int     `json:"total_completions"`
	CompletionRate        float64 `json:"completion_rate"`
	TotalPointsAvailable  int     `json:"total_points_available"`
	PointsEarned          int     `json:"points_earned"`
}

// AchievementLeaderboardEntry represents a player's position on the achievement leaderboard
type AchievementLeaderboardEntry struct {
	PlayerID           int        `json:"player_id" db:"id"`
	PlayerName         string     `json:"player_name" db:"name"`
	AchievementsEarned int        `json:"achievements_earned" db:"achievements_earned"`
	TotalPoints        int        `json:"total_points" db:"total_points"`
	LastAchievementAt  *time.Time `json:"last_achievement_at" db:"last_achievement_at"`
}