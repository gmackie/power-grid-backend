# Power Grid Analytics API

The Power Grid server provides comprehensive REST APIs for game analytics, player statistics, achievements, and leaderboards.

## Base URL

```
http://localhost:4080/api
```

## Authentication

Currently, the API is open and uses player names for identification. In production, you should implement proper authentication.

## Endpoints

### Player Statistics

#### Get Player Stats
```http
GET /api/players/{playerName}
```

Returns comprehensive statistics for a specific player.

**Response:**
```json
{
  "player_name": "JohnDoe",
  "games_played": 42,
  "games_won": 15,
  "win_rate": 35.71,
  "total_cities": 324,
  "total_plants": 189,
  "total_money": 12450,
  "total_resources": 876,
  "favorite_map": "usa",
  "play_time_minutes": 1260,
  "last_played": "2025-07-18T15:30:00Z",
  "created_at": "2025-01-01T00:00:00Z",
  "updated_at": "2025-07-18T15:30:00Z"
}
```

#### Get Player Achievements
```http
GET /api/players/{playerName}/achievements
```

Returns all achievements earned by a player.

**Response:**
```json
{
  "player_name": "JohnDoe",
  "total_achievements": 12,
  "total_points": 285,
  "achievements": [
    {
      "achievement_id": "first_win",
      "achievement": {
        "id": "first_win",
        "name": "First Victory",
        "description": "Win your first game",
        "icon": "🏆",
        "category": "victory",
        "points": 10
      },
      "earned_at": "2025-01-15T20:00:00Z",
      "game_id": "abc123"
    }
  ],
  "recent_achievements": [...],
  "category_breakdown": {
    "victory": 4,
    "economic": 3,
    "expansion": 3,
    "special": 2
  }
}
```

#### Get Player Game History
```http
GET /api/players/{playerName}/history?limit=20
```

Returns recent games played by a player.

**Query Parameters:**
- `limit` (optional): Number of games to return (default: 20)

**Response:**
```json
{
  "player_name": "JohnDoe",
  "game_count": 20,
  "games": [
    {
      "id": "game_123_1234567890",
      "game_id": "game_123",
      "game_name": "Competitive Match #5",
      "map_name": "usa",
      "start_time": "2025-07-18T14:00:00Z",
      "end_time": "2025-07-18T15:30:00Z",
      "duration_minutes": 90,
      "winner": "JohnDoe",
      "total_rounds": 12,
      "players": [
        {
          "player_name": "JohnDoe",
          "position": 1,
          "final_cities": 17,
          "final_plants": 4,
          "final_money": 45,
          "powered_cities": 17,
          "resources_used": 24,
          "is_winner": true
        }
      ]
    }
  ]
}
```

#### Get Player Progress
```http
GET /api/players/{playerName}/progress
```

Returns player improvement metrics over time.

**Response:**
```json
{
  "player_name": "JohnDoe",
  "progress": [
    {
      "date": "2025-07-01T00:00:00Z",
      "win_rate": 25.0,
      "avg_cities": 12.5,
      "avg_score": 180.0,
      "game_count": 4
    }
  ],
  "milestones": [
    {
      "name": "First Victory",
      "description": "Won your first game",
      "achieved_at": "2025-01-15T20:00:00Z",
      "game_id": "abc123"
    }
  ]
}
```

### Achievements

#### List All Achievements
```http
GET /api/achievements
```

Returns all available achievements in the game.

**Response:**
```json
[
  {
    "id": "first_win",
    "name": "First Victory",
    "description": "Win your first game",
    "icon": "🏆",
    "category": "victory",
    "points": 10,
    "criteria": "games_won >= 1"
  },
  {
    "id": "money_bags",
    "name": "Money Bags",
    "description": "End a game with 200+ elektro",
    "icon": "💰",
    "category": "economic",
    "points": 20,
    "criteria": "final_money >= 200"
  }
]
```

### Leaderboard

#### Get Leaderboard
```http
GET /api/leaderboard?limit=10
```

Returns the top players ranked by composite score.

**Query Parameters:**
- `limit` (optional): Number of entries to return (default: 10)

**Response:**
```json
{
  "timestamp": "2025-07-18T16:00:00Z",
  "player_count": 10,
  "leaderboard": [
    {
      "rank": 1,
      "player_name": "ProPlayer",
      "score": 2450,
      "games_won": 23,
      "win_rate": 46.0,
      "last_played": "2025-07-18T15:00:00Z"
    },
    {
      "rank": 2,
      "player_name": "JohnDoe",
      "score": 2180,
      "games_won": 15,
      "win_rate": 35.71,
      "last_played": "2025-07-18T15:30:00Z"
    }
  ]
}
```

### Game Analytics

#### Get Game Analytics
```http
GET /api/analytics/games?start=2025-07-01T00:00:00Z&end=2025-07-31T23:59:59Z
```

Returns aggregate analytics for all games.

**Query Parameters:**
- `start` (optional): Start date in RFC3339 format
- `end` (optional): End date in RFC3339 format

**Response:**
```json
{
  "total_games": 156,
  "average_game_time_minutes": 75,
  "popular_maps": {
    "usa": 89,
    "germany": 67
  },
  "peak_hours": {
    "20": 25,
    "21": 30,
    "22": 18
  },
  "recent_games": [
    {
      "id": "game_456_1234567890",
      "game_id": "game_456",
      "game_name": "Evening Match",
      "map_name": "usa",
      "start_time": "2025-07-18T20:00:00Z",
      "end_time": "2025-07-18T21:15:00Z",
      "duration_minutes": 75,
      "winner": "Alice",
      "total_rounds": 10,
      "players": [...]
    }
  ]
}
```

### Health Check

#### API Health
```http
GET /api/health
```

Returns the health status of the analytics API.

**Response:**
```json
{
  "status": "healthy",
  "service": "analytics-api",
  "timestamp": "2025-07-18T16:00:00Z"
}
```

## Achievement Categories

### Victory Achievements
- **First Victory** (10 pts): Win your first game
- **Hat Trick** (25 pts): Win 3 games in a row
- **Unstoppable** (50 pts): Win 5 games in a row
- **Master Strategist** (100 pts): Win 50 games

### Economic Achievements
- **Money Bags** (20 pts): End a game with 200+ elektro
- **Resource Hoarder** (15 pts): Own 20+ resources at once
- **Eco Warrior** (30 pts): Win using only renewable power plants

### Expansion Achievements
- **City Builder** (25 pts): Build in 15+ cities in a single game
- **Rapid Expansion** (30 pts): Build in 10 cities within 5 rounds
- **Monopolist** (35 pts): Control all cities in a region

### Power Plant Achievements
- **Plant Collector** (20 pts): Own 5 power plants at once
- **High Capacity** (15 pts): Own a plant that powers 7+ cities
- **Diversified Portfolio** (25 pts): Own plants of 4 different resource types

### Special Achievements
- **Underdog Victory** (40 pts): Win from last place in turn order
- **Perfectionist** (30 pts): Win by powering all your cities
- **Speed Demon** (35 pts): Win a game in under 30 minutes

### Participation Achievements
- **Regular Player** (15 pts): Play 25 games
- **Dedicated Player** (50 pts): Play 100 games
- **Map Explorer** (25 pts): Play on all available maps

## Score Calculation

Player scores on the leaderboard are calculated using:
- Games won × 100 points
- Win rate × 10 points
- Total cities built
- Total power plants × 5 points
- Achievement points

## Data Persistence

Analytics data is stored locally in JSON files:
- `data/analytics/player_stats.json`
- `data/analytics/game_records.json`
- `data/analytics/player_achievements.json`

Data is automatically saved every 5 minutes and on server shutdown.

## Integration with Game Server

The analytics system automatically tracks:
- Game start/end events
- Player joins/leaves
- Game state changes
- Round progression
- Final results

No additional client integration is required - analytics are collected automatically based on player names.

## Usage Examples

### Get a player's complete profile
```bash
# Get stats
curl http://localhost:4080/api/players/JohnDoe

# Get achievements
curl http://localhost:4080/api/players/JohnDoe/achievements

# Get recent games
curl http://localhost:4080/api/players/JohnDoe/history?limit=10
```

### Check the leaderboard
```bash
curl http://localhost:4080/api/leaderboard?limit=20
```

### View game analytics for this month
```bash
curl "http://localhost:4080/api/analytics/games?start=2025-07-01T00:00:00Z&end=2025-07-31T23:59:59Z"
```

### List all available achievements
```bash
curl http://localhost:4080/api/achievements
```

## Future Enhancements

1. **Authentication**: Add proper player authentication
2. **Database**: Migrate from JSON files to a proper database
3. **Real-time Updates**: WebSocket endpoints for live updates
4. **Advanced Analytics**: More detailed game analysis and insights
5. **Player Comparisons**: Compare stats between players
6. **Seasonal Leaderboards**: Time-based leaderboards and tournaments
7. **Custom Achievements**: Allow server admins to define custom achievements