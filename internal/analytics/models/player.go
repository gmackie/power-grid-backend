package models

import (
	"time"
)

// Player represents a player in the database
type Player struct {
	ID                   int       `json:"id" db:"id"`
	Name                 string    `json:"name" db:"name"`
	FirstSeen            time.Time `json:"first_seen" db:"first_seen"`
	LastSeen             time.Time `json:"last_seen" db:"last_seen"`
	TotalGames           int       `json:"total_games" db:"total_games"`
	TotalWins            int       `json:"total_wins" db:"total_wins"`
	TotalPlaytimeMinutes int       `json:"total_playtime_minutes" db:"total_playtime_minutes"`
	FavoriteMap          *string   `json:"favorite_map" db:"favorite_map"`
	PreferredColor       *string   `json:"preferred_color" db:"preferred_color"`
	CreatedAt            time.Time `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time `json:"updated_at" db:"updated_at"`
}

// PlayerStats represents aggregated player statistics
type PlayerStats struct {
	ID                        int       `json:"id" db:"id"`
	PlayerID                  int       `json:"player_id" db:"player_id"`
	GamesPlayed               int       `json:"games_played" db:"games_played"`
	GamesWon                  int       `json:"games_won" db:"games_won"`
	GamesLost                 int       `json:"games_lost" db:"games_lost"`
	WinRate                   float64   `json:"win_rate" db:"win_rate"`
	AvgFinalCities            float64   `json:"avg_final_cities" db:"avg_final_cities"`
	AvgFinalPlants            float64   `json:"avg_final_plants" db:"avg_final_plants"`
	AvgFinalMoney             float64   `json:"avg_final_money" db:"avg_final_money"`
	AvgGameDurationMinutes    float64   `json:"avg_game_duration_minutes" db:"avg_game_duration_minutes"`
	MaxCitiesSingleGame       int       `json:"max_cities_single_game" db:"max_cities_single_game"`
	MaxPlantsSingleGame       int       `json:"max_plants_single_game" db:"max_plants_single_game"`
	MaxMoneySingleGame        int       `json:"max_money_single_game" db:"max_money_single_game"`
	FastestWinMinutes         *int      `json:"fastest_win_minutes" db:"fastest_win_minutes"`
	LongestWinMinutes         *int      `json:"longest_win_minutes" db:"longest_win_minutes"`
	TotalResourcesBought      int       `json:"total_resources_bought" db:"total_resources_bought"`
	TotalMoneySpent           int       `json:"total_money_spent" db:"total_money_spent"`
	TotalPlantsOwned          int       `json:"total_plants_owned" db:"total_plants_owned"`
	AvgPlantEfficiency        float64   `json:"avg_plant_efficiency" db:"avg_plant_efficiency"`
	TotalCitiesBuilt          int       `json:"total_cities_built" db:"total_cities_built"`
	AvgExpansionRate          float64   `json:"avg_expansion_rate" db:"avg_expansion_rate"`
	TotalAchievementPoints    int       `json:"total_achievement_points" db:"total_achievement_points"`
	TotalAchievementsEarned   int       `json:"total_achievements_earned" db:"total_achievements_earned"`
	TotalPlaytimeMinutes      int       `json:"total_playtime_minutes" db:"total_playtime_minutes"`
	AvgSessionLengthMinutes   float64   `json:"avg_session_length_minutes" db:"avg_session_length_minutes"`
	LastUpdated               time.Time `json:"last_updated" db:"last_updated"`
}

// LeaderboardEntry represents a player's position on the leaderboard
type LeaderboardEntry struct {
	PlayerID                int       `json:"player_id" db:"id"`
	PlayerName              string    `json:"player_name" db:"name"`
	GamesPlayed             int       `json:"games_played" db:"games_played"`
	GamesWon                int       `json:"games_won" db:"games_won"`
	WinRate                 float64   `json:"win_rate" db:"win_rate"`
	AvgFinalCities          float64   `json:"avg_final_cities" db:"avg_final_cities"`
	TotalAchievementPoints  int       `json:"total_achievement_points" db:"total_achievement_points"`
	TotalCitiesBuilt        int       `json:"total_cities_built" db:"total_cities_built"`
	CompositeScore          float64   `json:"composite_score" db:"composite_score"`
	LastSeen                time.Time `json:"last_seen" db:"last_seen"`
}