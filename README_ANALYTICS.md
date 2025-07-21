# Advanced Analytics System Documentation

## Overview

The Power Grid game server now includes a comprehensive SQLite-based analytics system that provides deep insights into player behavior, game dynamics, and strategic patterns. This system offers both basic statistics and advanced analytics capabilities.

## Architecture

### Database Schema
- **15+ Tables**: Players, games, achievements, statistics, detailed tracking
- **Automated Triggers**: Real-time statistics maintenance
- **Views**: Optimized queries for common analytics operations
- **Indexes**: Performance-optimized for analytics queries

### Service Layer
- **DatabaseService**: Main analytics service with SQLite backend
- **Repository Pattern**: Organized data access with specialized repositories
- **Advanced Analytics**: Complex queries and statistical analysis

## API Endpoints

### Basic Analytics

#### Player Statistics
```
GET /api/players/{name}/stats
```
Basic player statistics including games played, win rate, average performance.

#### Player Achievements
```
GET /api/players/{name}/achievements
```
All achievements earned by a player with progress tracking.

#### Leaderboard
```
GET /api/leaderboard?limit=50
```
Player rankings based on composite performance scores.

#### Game Analytics
```
GET /api/analytics/games?days=30
```
Game-level analytics including duration, completion rates, map popularity.

### Advanced Analytics

#### Player Performance Metrics
```
GET /api/players/{name}/performance?days=30
```
Comprehensive player analysis including:
- **Strategic Classification**: Player type based on behavior patterns
- **Performance Metrics**: Resource efficiency, expansion rate, economic performance
- **Consistency Analysis**: Strategic consistency and improvement trends
- **Map Preferences**: Win rates across different maps
- **Recent Form**: Performance trend analysis

**Response Structure:**
```json
{
  "player_name": "PlayerName",
  "metrics": {
    "player_id": 1,
    "player_name": "PlayerName",
    "total_games": 25,
    "win_rate": 0.48,
    "avg_position": 2.3,
    "avg_cities_built": 12.5,
    "avg_plants_owned": 3.2,
    "avg_final_money": 85.7,
    "resource_efficiency": 4.2,
    "expansion_rate": 1.8,
    "economic_performance": 3.5,
    "strategic_consistency": 72.8,
    "recent_form_trend": "improving",
    "player_type_classification": "tactical_player",
    "map_preferences": {
      "USA": 0.55,
      "Germany": 0.42
    }
  }
}
```

#### Competitor Analysis
```
GET /api/players/{name}/competitors?days=30
```
Head-to-head analysis against other players:
- **Matchup Records**: Wins/losses against specific opponents
- **Dominance Patterns**: Competitive relationships
- **Position Differences**: Average performance gaps

#### Player Skill Progression
```
GET /api/players/{name}/progression?days=90
```
Skill development tracking over time (framework for future implementation).

#### Advanced Game Analytics
```
GET /api/analytics/advanced?days=30
```
Deep game analysis including:
- **Competitiveness Metrics**: Close game percentages, comeback rates
- **Market Analysis**: Power plant and resource market dynamics
- **Strategy Effectiveness**: Success rates of different approaches
- **Seasonal Trends**: Activity patterns over time

#### Activity Report
```
GET /api/analytics/activity?days=30
```
Server activity and engagement metrics:
- **Daily Activity**: Games played per day
- **Player Engagement**: Unique players and session patterns
- **Player Type Distribution**: Classification of active players

**Response Structure:**
```json
{
  "total_games": 145,
  "unique_players": 28,
  "active_days": 23,
  "avg_game_duration": 45.2,
  "daily_activity": [
    {"date": "2024-01-15", "games": 8},
    {"date": "2024-01-14", "games": 12}
  ],
  "player_type_distribution": {
    "tactical_player": 8,
    "expansion_master": 3,
    "economic_powerhouse": 2,
    "developing_player": 12,
    "casual_player": 3
  }
}
```

#### Player Type Distribution
```
GET /api/analytics/player-types
```
Distribution of player classifications across the player base.

#### Map Analytics
```
GET /api/analytics/maps?days=30
```
Map-specific performance analysis (framework for future implementation).

## Player Classification System

The analytics system automatically classifies players based on their gameplay patterns:

### Classification Types
- **Novice**: Less than 5 games played
- **Expansion Master**: High win rate (60%+) with focus on city building (15+ cities)
- **Economic Powerhouse**: High win rate (60%+) with strong financial performance (100+ elektro)
- **Strategic Dominator**: High win rate (60%+) with balanced approach
- **Aggressive Expander**: Moderate win rate (40%+) with rapid expansion (12+ cities)
- **Tactical Player**: Moderate win rate (40%+) with strategic gameplay
- **Resource Optimizer**: Lower win rate (20%+) but efficient resource management (80+ elektro)
- **Developing Player**: Lower win rate (20%+) still learning the game
- **Casual Player**: Occasional play with variable performance

### Classification Criteria
Players are classified using multiple factors:
- **Win Rate**: Primary performance indicator
- **City Building**: Average cities built per game
- **Economic Management**: Average final money
- **Strategic Consistency**: Variance in performance metrics
- **Game Frequency**: Total games played

## Advanced Metrics

### Performance Metrics
- **Resource Efficiency**: Money-to-resource ratio optimization
- **Expansion Rate**: Cities built per game round
- **Economic Performance**: Money management relative to city count
- **Strategic Consistency**: Inverse of performance variance (higher = more consistent)

### Competitiveness Analysis
- **Margin of Victory**: How close games tend to be
- **Comeback Percentage**: Games won from behind
- **Position Variability**: How much final positions vary
- **Close Game Percentage**: Games decided by small margins

### Market Analysis (Framework)
- **Power Plant Popularity**: Most sought-after plants
- **Resource Market Dynamics**: Price and availability trends
- **Auction Patterns**: Bidding behavior analysis
- **Strategy Effectiveness**: Success rates of different approaches

## Usage Examples

### Getting Detailed Player Analysis
```bash
# Get comprehensive player performance metrics
curl "http://localhost:4080/api/players/Alice/performance?days=60"

# Get head-to-head matchup analysis
curl "http://localhost:4080/api/players/Alice/competitors?days=30"

# Get player achievement progress
curl "http://localhost:4080/api/players/Alice/achievements"
```

### Server Analytics
```bash
# Get overall server activity report
curl "http://localhost:4080/api/analytics/activity?days=7"

# Get player type distribution
curl "http://localhost:4080/api/analytics/player-types"

# Get advanced game analytics
curl "http://localhost:4080/api/analytics/advanced?days=30"
```

### Basic Analytics
```bash
# Get leaderboard
curl "http://localhost:4080/api/leaderboard?limit=10"

# Get game analytics
curl "http://localhost:4080/api/analytics/games?days=30"

# Get achievement statistics
curl "http://localhost:4080/api/analytics/achievements"
```

## Configuration

### Server Launch
```bash
# Launch with SQLite analytics (default)
./powergrid_server_db --use-db=true

# Launch with file-based analytics (legacy)
./powergrid_server_db --use-db=false

# Custom data directory
./powergrid_server_db --data-dir=/path/to/data
```

### Database Configuration
- **Database File**: `./data/analytics.db`
- **Migration System**: Automatic schema updates
- **Connection Pool**: 25 max connections, 10 idle
- **WAL Mode**: Enabled for better performance
- **Foreign Keys**: Enforced for data integrity

## Future Enhancements

### Planned Features
1. **Machine Learning Integration**: Predict player behavior and game outcomes
2. **Real-time Analytics**: Live game statistics and dashboards
3. **Tournament Support**: Bracket management and tournament analytics
4. **Advanced Visualizations**: Charts and graphs for analytics data
5. **Export Capabilities**: CSV/JSON export for external analysis
6. **Custom Metrics**: User-defined analytics queries
7. **Performance Monitoring**: Server performance and optimization metrics

### Extension Points
- **Custom Repositories**: Add specialized analytics repositories
- **Additional Metrics**: Extend PlayerPerformanceMetrics structure
- **New Classifications**: Add more sophisticated player typing
- **External Integrations**: Connect to external analytics platforms

## Performance Considerations

### Database Optimization
- **Indexes**: Strategically placed for query performance
- **Triggers**: Automated statistics maintenance
- **Views**: Pre-computed complex queries
- **Connection Pooling**: Efficient resource utilization

### Query Optimization
- **Parameterized Queries**: Prevent SQL injection and improve performance
- **Batch Operations**: Efficient bulk data operations
- **Caching Strategy**: Ready for Redis/Memcached integration
- **Pagination**: Large result set handling

## Security

### Data Protection
- **Parameterized Queries**: SQL injection prevention
- **Input Validation**: All user inputs validated
- **Rate Limiting**: API endpoint protection (ready for implementation)
- **CORS Configuration**: Cross-origin request handling

### Privacy Considerations
- **Player Data**: Aggregated statistics, no personal information
- **Game Data**: Complete game state tracking for analysis
- **Retention Policies**: Ready for data retention configuration

This advanced analytics system provides comprehensive insights into Power Grid gameplay, helping players improve their strategies and server administrators understand game dynamics and player engagement patterns.