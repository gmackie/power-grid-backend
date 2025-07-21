# Analytics & Achievement System

The Power Grid server includes a comprehensive analytics and achievement system that tracks player performance, game statistics, and provides REST APIs for data access.

## ✨ Features

### 🏆 Achievement System
- **25+ Predefined Achievements** across 5 categories
- **Real-time Achievement Tracking** during gameplay
- **Point-based Scoring** system
- **Category Breakdown**: Victory, Economic, Expansion, Plants, Special, Participation

### 📊 Player Analytics
- **Complete Game History** with detailed match results
- **Performance Metrics**: Win rate, average cities, resource usage
- **Progress Tracking** over time
- **Favorite Maps** and play patterns
- **Total Play Time** tracking

### 🏅 Leaderboards
- **Composite Scoring** based on wins, achievements, and performance
- **Minimum Game Requirements** for fair ranking
- **Configurable Limits** for display
- **Real-time Updates** as games complete

### 📈 Game Analytics
- **Aggregate Statistics** across all games
- **Peak Hour Analysis** for server optimization
- **Map Popularity** tracking
- **Average Game Duration** metrics
- **Time-range Filtering** for custom reports

## 🚀 Quick Start

### Generate Demo Data
```bash
# Create sample analytics data for testing
make demo-analytics

# Or with custom parameters
./cmd/analytics_demo/analytics_demo -games 25 -data-dir ./data/analytics
```

### Test the API
```bash
# Test all endpoints
make test-analytics

# Or manually
./scripts/test_analytics_api.sh
```

### Start Server with Analytics
```bash
# Build and run server
make build
./powergrid_server

# Server automatically enables analytics APIs at /api/*
```

## 📋 API Endpoints

### Player Data
- `GET /api/players/{name}` - Player statistics
- `GET /api/players/{name}/achievements` - Player achievements
- `GET /api/players/{name}/history` - Game history
- `GET /api/players/{name}/progress` - Performance over time

### Global Data
- `GET /api/leaderboard` - Top players ranking
- `GET /api/achievements` - All available achievements
- `GET /api/analytics/games` - Game analytics overview
- `GET /api/health` - API health status

### Example Usage
```bash
# Get player stats
curl http://localhost:4080/api/players/Alice

# View leaderboard
curl http://localhost:4080/api/leaderboard?limit=10

# List all achievements
curl http://localhost:4080/api/achievements

# Game analytics for specific time period
curl "http://localhost:4080/api/analytics/games?start=2025-07-01T00:00:00Z&end=2025-07-31T23:59:59Z"
```

## 🏆 Achievement Categories

### Victory Achievements (🏆)
- **First Victory** (10 pts) - Win your first game
- **Hat Trick** (25 pts) - Win 3 games in a row
- **Unstoppable** (50 pts) - Win 5 games in a row
- **Master Strategist** (100 pts) - Win 50 games

### Economic Achievements (💰)
- **Money Bags** (20 pts) - End with 200+ elektro
- **Resource Hoarder** (15 pts) - Own 20+ resources at once
- **Eco Warrior** (30 pts) - Win using only renewable plants

### Expansion Achievements (🏙️)
- **City Builder** (25 pts) - Build in 15+ cities in one game
- **Rapid Expansion** (30 pts) - Build 10 cities in 5 rounds
- **Monopolist** (35 pts) - Control all cities in a region

### Power Plant Achievements (⚡)
- **Plant Collector** (20 pts) - Own 5 power plants at once
- **High Capacity** (15 pts) - Own a plant powering 7+ cities
- **Diversified Portfolio** (25 pts) - Own 4 different resource types

### Special Achievements (✨)
- **Underdog Victory** (40 pts) - Win from last place
- **Perfectionist** (30 pts) - Win by powering all cities
- **Speed Demon** (35 pts) - Win in under 30 minutes

### Participation Achievements (🎮)
- **Regular Player** (15 pts) - Play 25 games
- **Dedicated Player** (50 pts) - Play 100 games
- **Map Explorer** (25 pts) - Play on all maps

## 📊 Leaderboard Scoring

Player rankings use a composite score:
- **Games Won** × 100 points
- **Win Rate** × 10 points
- **Total Cities** built
- **Total Plants** × 5 points
- **Achievement Points** earned

Minimum 5 games required for leaderboard inclusion.

## 🔧 Integration

### Automatic Tracking
The analytics system automatically captures:
- Game start/end events
- Player actions and moves
- Final game results
- Achievement progress
- Performance metrics

### Player Continuity
Players are tracked by name across games, providing:
- Persistent statistics
- Achievement progress
- Historical performance
- Long-term trends

### Real-time Updates
- Achievement notifications during gameplay
- Live leaderboard updates
- Immediate stat recording
- Progress tracking

## 📁 Data Storage

Analytics data is stored in JSON files:
```
data/analytics/
├── player_stats.json      # Individual player statistics
├── game_records.json      # Complete game history
└── player_achievements.json # Achievement progress
```

Data is automatically saved:
- Every 5 minutes during operation
- On server shutdown
- After each game completion

## 🛠️ Development

### Adding New Achievements
1. Define achievement in `models/analytics.go`
2. Add criteria logic in `internal/analytics/service.go`
3. Test with demo data generation

### Custom Analytics
```go
// Track custom events
service.TrackGameEvent(gameID, "custom_event", eventData)

// Add custom player metrics
playerStats.CustomField = value
```

### API Extensions
New endpoints can be added to `handlers/analytics.go`:
```go
mux.HandleFunc("/api/custom-endpoint", h.handleCustomEndpoint)
```

## 🎯 Use Cases

### Tournament Management
- Track tournament leaderboards
- Monitor player performance
- Generate tournament reports
- Award achievements for milestones

### Game Balance Analysis
- Identify dominant strategies
- Monitor resource usage patterns
- Track map popularity
- Analyze game duration trends

### Community Engagement
- Celebrate player achievements
- Showcase leaderboard champions
- Track community growth
- Identify active players

### Performance Monitoring
- Monitor server game load
- Identify peak playing times
- Track player retention
- Measure engagement metrics

## 🔮 Future Enhancements

### Planned Features
- **Season-based Leaderboards** with reset periods
- **Player Comparison Tool** for head-to-head stats
- **Advanced Game Analytics** with strategy analysis
- **Custom Achievement Editor** for server admins
- **Real-time WebSocket Updates** for live dashboards

### Database Migration
- PostgreSQL/MySQL support for production
- Better query performance
- Concurrent access handling
- Data backup and recovery

### Enhanced APIs
- GraphQL endpoint for complex queries
- Bulk data export capabilities
- Advanced filtering and sorting
- Pagination for large datasets

## 🧪 Testing

### Run Analytics Tests
```bash
# Start server
make run &

# Generate test data
make demo-analytics

# Test all endpoints
make test-analytics

# View generated data
ls -la data/analytics/
```

### Integration Testing
```bash
# Test with AI clients
make ai-demo

# Run simulation to generate real game data
make simulation

# Check API responses
curl http://localhost:4080/api/leaderboard
```

This analytics system provides comprehensive insights into Power Grid gameplay while maintaining simple integration and powerful REST APIs for external applications.