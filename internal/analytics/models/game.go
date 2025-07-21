package models

import (
	"time"
)

// GameRecord represents a game in the database
type GameRecord struct {
	ID              int       `json:"id" db:"id"`
	GameID          string    `json:"game_id" db:"game_id"`
	Name            string    `json:"name" db:"name"`
	MapName         string    `json:"map_name" db:"map_name"`
	MaxPlayers      int       `json:"max_players" db:"max_players"`
	ActualPlayers   int       `json:"actual_players" db:"actual_players"`
	Status          string    `json:"status" db:"status"`
	WinnerPlayerID  *int      `json:"winner_player_id" db:"winner_player_id"`
	StartedAt       *time.Time `json:"started_at" db:"started_at"`
	EndedAt         *time.Time `json:"ended_at" db:"ended_at"`
	DurationMinutes *int      `json:"duration_minutes" db:"duration_minutes"`
	TotalRounds     int       `json:"total_rounds" db:"total_rounds"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// GameParticipant represents a player's participation in a game
type GameParticipant struct {
	ID             int       `json:"id" db:"id"`
	GameID         int       `json:"game_id" db:"game_id"`
	PlayerID       int       `json:"player_id" db:"player_id"`
	PlayerName     string    `json:"player_name" db:"player_name"`
	Color          *string   `json:"color" db:"color"`
	TurnOrder      *int      `json:"turn_order" db:"turn_order"`
	FinalPosition  *int      `json:"final_position" db:"final_position"`
	FinalCities    int       `json:"final_cities" db:"final_cities"`
	FinalPlants    int       `json:"final_plants" db:"final_plants"`
	FinalMoney     int       `json:"final_money" db:"final_money"`
	FinalResources int       `json:"final_resources" db:"final_resources"`
	PoweredCities  int       `json:"powered_cities" db:"powered_cities"`
	IsWinner       bool      `json:"is_winner" db:"is_winner"`
	JoinedAt       time.Time `json:"joined_at" db:"joined_at"`
}

// GameEvent represents an event that occurred during a game
type GameEvent struct {
	ID           int       `json:"id" db:"id"`
	GameID       int       `json:"game_id" db:"game_id"`
	PlayerID     *int      `json:"player_id" db:"player_id"`
	EventType    string    `json:"event_type" db:"event_type"`
	EventData    *string   `json:"event_data" db:"event_data"`
	RoundNumber  *int      `json:"round_number" db:"round_number"`
	Phase        *string   `json:"phase" db:"phase"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
}

// GameAnalytics represents aggregated game analytics
type GameAnalytics struct {
	TotalGames      int                    `json:"total_games"`
	CompletedGames  int                    `json:"completed_games"`
	AvgGameDuration float64                `json:"avg_game_duration_minutes"`
	MapPopularity   map[string]int         `json:"map_popularity"`
	PlayerCounts    map[int]int            `json:"player_counts"`
	RecentGames     []*GameRecord          `json:"recent_games,omitempty"`
}