package models

import (
	"time"
)

// PlayerStats represents aggregated statistics for a player
type PlayerStats struct {
	PlayerName     string    `json:"player_name"`
	GamesPlayed    int       `json:"games_played"`
	GamesWon       int       `json:"games_won"`
	WinRate        float64   `json:"win_rate"`
	TotalCities    int       `json:"total_cities"`
	TotalPlants    int       `json:"total_plants"`
	TotalMoney     int       `json:"total_money"`
	TotalResources int       `json:"total_resources"`
	FavoriteMap    string    `json:"favorite_map"`
	PlayTime       int       `json:"play_time_minutes"`
	LastPlayed     time.Time `json:"last_played"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// GameRecord represents a completed game record
type GameRecord struct {
	ID         string              `json:"id"`
	GameID     string              `json:"game_id"`
	GameName   string              `json:"game_name"`
	MapName    string              `json:"map_name"`
	StartTime  time.Time           `json:"start_time"`
	EndTime    time.Time           `json:"end_time"`
	Duration   int                 `json:"duration_minutes"`
	Winner     string              `json:"winner"`
	TotalRounds int                `json:"total_rounds"`
	Players    []PlayerGameResult  `json:"players"`
	CreatedAt  time.Time           `json:"created_at"`
}

// PlayerGameResult represents a player's performance in a single game
type PlayerGameResult struct {
	PlayerName    string `json:"player_name"`
	Position      int    `json:"position"`
	FinalCities   int    `json:"final_cities"`
	FinalPlants   int    `json:"final_plants"`
	FinalMoney    int    `json:"final_money"`
	PoweredCities int    `json:"powered_cities"`
	ResourcesUsed int    `json:"resources_used"`
	IsWinner      bool   `json:"is_winner"`
}

// Achievement represents a player achievement
type Achievement struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
	Points      int       `json:"points"`
	Criteria    string    `json:"criteria"`
	CreatedAt   time.Time `json:"created_at"`
}

// PlayerAchievement represents an earned achievement
type PlayerAchievement struct {
	PlayerName     string    `json:"player_name"`
	AchievementID  string    `json:"achievement_id"`
	Achievement    Achievement `json:"achievement"`
	EarnedAt       time.Time `json:"earned_at"`
	GameID         string    `json:"game_id"`
	Progress       int       `json:"progress"`
	MaxProgress    int       `json:"max_progress"`
}

// LeaderboardEntry represents a player's position on a leaderboard
type LeaderboardEntry struct {
	Rank       int     `json:"rank"`
	PlayerName string  `json:"player_name"`
	Score      int     `json:"score"`
	GamesWon   int     `json:"games_won"`
	WinRate    float64 `json:"win_rate"`
	LastPlayed time.Time `json:"last_played"`
}

// Analytics request/response types

// TimeRange for filtering analytics data
type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// PlayerAnalyticsRequest for player-specific analytics
type PlayerAnalyticsRequest struct {
	PlayerName string     `json:"player_name"`
	TimeRange  *TimeRange `json:"time_range,omitempty"`
	Limit      int        `json:"limit,omitempty"`
}

// GameAnalyticsResponse contains analytics for games
type GameAnalyticsResponse struct {
	TotalGames       int                    `json:"total_games"`
	AverageGameTime  int                    `json:"average_game_time_minutes"`
	PopularMaps      map[string]int         `json:"popular_maps"`
	PeakHours        map[int]int            `json:"peak_hours"`
	RecentGames      []GameRecord           `json:"recent_games"`
}

// PlayerProgressResponse tracks player improvement over time
type PlayerProgressResponse struct {
	PlayerName string                 `json:"player_name"`
	Progress   []ProgressDataPoint    `json:"progress"`
	Milestones []Milestone            `json:"milestones"`
}

// ProgressDataPoint represents a point in player progress
type ProgressDataPoint struct {
	Date      time.Time `json:"date"`
	WinRate   float64   `json:"win_rate"`
	AvgCities float64   `json:"avg_cities"`
	AvgScore  float64   `json:"avg_score"`
	GameCount int       `json:"game_count"`
}

// Milestone represents a significant achievement
type Milestone struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	AchievedAt  time.Time `json:"achieved_at"`
	GameID      string    `json:"game_id"`
}

// Achievement definitions
var PredefinedAchievements = []Achievement{
	// Victory achievements
	{
		ID:          "first_win",
		Name:        "First Victory",
		Description: "Win your first game",
		Icon:        "🏆",
		Category:    "victory",
		Points:      10,
		Criteria:    "games_won >= 1",
	},
	{
		ID:          "winning_streak_3",
		Name:        "Hat Trick",
		Description: "Win 3 games in a row",
		Icon:        "🎩",
		Category:    "victory",
		Points:      25,
		Criteria:    "winning_streak >= 3",
	},
	{
		ID:          "winning_streak_5",
		Name:        "Unstoppable",
		Description: "Win 5 games in a row",
		Icon:        "🔥",
		Category:    "victory",
		Points:      50,
		Criteria:    "winning_streak >= 5",
	},
	{
		ID:          "master_strategist",
		Name:        "Master Strategist",
		Description: "Win 50 games",
		Icon:        "🧠",
		Category:    "victory",
		Points:      100,
		Criteria:    "games_won >= 50",
	},
	
	// Economic achievements
	{
		ID:          "money_bags",
		Name:        "Money Bags",
		Description: "End a game with 200+ elektro",
		Icon:        "💰",
		Category:    "economic",
		Points:      20,
		Criteria:    "final_money >= 200",
	},
	{
		ID:          "resource_hoarder",
		Name:        "Resource Hoarder",
		Description: "Own 20+ resources at once",
		Icon:        "📦",
		Category:    "economic",
		Points:      15,
		Criteria:    "max_resources >= 20",
	},
	{
		ID:          "eco_warrior",
		Name:        "Eco Warrior",
		Description: "Win using only renewable power plants",
		Icon:        "🌱",
		Category:    "economic",
		Points:      30,
		Criteria:    "renewable_only_win",
	},
	
	// Expansion achievements
	{
		ID:          "city_builder",
		Name:        "City Builder",
		Description: "Build in 15+ cities in a single game",
		Icon:        "🏙️",
		Category:    "expansion",
		Points:      25,
		Criteria:    "max_cities >= 15",
	},
	{
		ID:          "rapid_expansion",
		Name:        "Rapid Expansion",
		Description: "Build in 10 cities within 5 rounds",
		Icon:        "⚡",
		Category:    "expansion",
		Points:      30,
		Criteria:    "rapid_expansion",
	},
	{
		ID:          "monopolist",
		Name:        "Monopolist",
		Description: "Control all cities in a region",
		Icon:        "👑",
		Category:    "expansion",
		Points:      35,
		Criteria:    "region_monopoly",
	},
	
	// Power plant achievements
	{
		ID:          "plant_collector",
		Name:        "Plant Collector",
		Description: "Own 5 power plants at once",
		Icon:        "🏭",
		Category:    "plants",
		Points:      20,
		Criteria:    "max_plants >= 5",
	},
	{
		ID:          "high_capacity",
		Name:        "High Capacity",
		Description: "Own a plant that powers 7+ cities",
		Icon:        "⚡",
		Category:    "plants",
		Points:      15,
		Criteria:    "max_plant_capacity >= 7",
	},
	{
		ID:          "diversified",
		Name:        "Diversified Portfolio",
		Description: "Own plants of 4 different resource types",
		Icon:        "🎨",
		Category:    "plants",
		Points:      25,
		Criteria:    "resource_diversity >= 4",
	},
	
	// Special achievements
	{
		ID:          "underdog",
		Name:        "Underdog Victory",
		Description: "Win from last place in turn order",
		Icon:        "🐕",
		Category:    "special",
		Points:      40,
		Criteria:    "underdog_win",
	},
	{
		ID:          "perfectionist",
		Name:        "Perfectionist",
		Description: "Win by powering all your cities",
		Icon:        "✨",
		Category:    "special",
		Points:      30,
		Criteria:    "perfect_power",
	},
	{
		ID:          "speed_demon",
		Name:        "Speed Demon",
		Description: "Win a game in under 30 minutes",
		Icon:        "🏎️",
		Category:    "special",
		Points:      35,
		Criteria:    "speed_win",
	},
	
	// Participation achievements
	{
		ID:          "regular_player",
		Name:        "Regular Player",
		Description: "Play 25 games",
		Icon:        "🎮",
		Category:    "participation",
		Points:      15,
		Criteria:    "games_played >= 25",
	},
	{
		ID:          "dedicated_player",
		Name:        "Dedicated Player",
		Description: "Play 100 games",
		Icon:        "🌟",
		Category:    "participation",
		Points:      50,
		Criteria:    "games_played >= 100",
	},
	{
		ID:          "map_explorer",
		Name:        "Map Explorer",
		Description: "Play on all available maps",
		Icon:        "🗺️",
		Category:    "participation",
		Points:      25,
		Criteria:    "all_maps_played",
	},
}