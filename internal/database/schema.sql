-- Power Grid Analytics Database Schema
-- SQLite Database for comprehensive game analytics

-- Players table - Master player registry
CREATE TABLE IF NOT EXISTS players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_games INTEGER DEFAULT 0,
    total_wins INTEGER DEFAULT 0,
    total_playtime_minutes INTEGER DEFAULT 0,
    favorite_map TEXT,
    preferred_color TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Games table - Master game registry
CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    map_name TEXT NOT NULL,
    max_players INTEGER DEFAULT 6,
    actual_players INTEGER NOT NULL,
    status TEXT DEFAULT 'lobby', -- lobby, playing, completed, abandoned
    winner_player_id INTEGER,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration_minutes INTEGER,
    total_rounds INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (winner_player_id) REFERENCES players(id)
);

-- Game participants - Player participation in games
CREATE TABLE IF NOT EXISTS game_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    player_name TEXT NOT NULL, -- Denormalized for easier queries
    color TEXT,
    turn_order INTEGER,
    final_position INTEGER,
    final_cities INTEGER DEFAULT 0,
    final_plants INTEGER DEFAULT 0,
    final_money INTEGER DEFAULT 0,
    final_resources INTEGER DEFAULT 0,
    powered_cities INTEGER DEFAULT 0,
    is_winner BOOLEAN DEFAULT FALSE,
    joined_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id),
    UNIQUE(game_id, player_id)
);

-- Game states - Detailed game progression tracking
CREATE TABLE IF NOT EXISTS game_states (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    round_number INTEGER NOT NULL,
    phase TEXT NOT NULL, -- player_order, auction, resources, building, bureaucracy
    current_turn_player_id INTEGER,
    state_data TEXT, -- JSON blob of complete game state
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (current_turn_player_id) REFERENCES players(id)
);

-- Player actions - Individual player moves and decisions
CREATE TABLE IF NOT EXISTS player_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    round_number INTEGER NOT NULL,
    phase TEXT NOT NULL,
    action_type TEXT NOT NULL, -- bid_plant, buy_resources, build_city, power_cities, end_turn
    action_data TEXT, -- JSON blob of action details
    action_result TEXT, -- success, failed, invalid
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- Power plant ownership tracking
CREATE TABLE IF NOT EXISTS player_power_plants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    plant_id INTEGER NOT NULL,
    plant_cost INTEGER NOT NULL,
    plant_capacity INTEGER NOT NULL,
    resource_type TEXT,
    resource_cost INTEGER DEFAULT 0,
    acquired_round INTEGER NOT NULL,
    sold_round INTEGER, -- NULL if still owned at game end
    auction_price INTEGER, -- What player paid for it
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- City ownership tracking
CREATE TABLE IF NOT EXISTS player_cities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    city_id TEXT NOT NULL,
    city_name TEXT,
    region TEXT,
    connection_cost INTEGER DEFAULT 0,
    built_round INTEGER NOT NULL,
    order_in_round INTEGER, -- Order of building within the round
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- Resource transactions - Track resource market activity
CREATE TABLE IF NOT EXISTS resource_transactions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    round_number INTEGER NOT NULL,
    resource_type TEXT NOT NULL,
    quantity INTEGER NOT NULL,
    price_per_unit INTEGER NOT NULL,
    total_cost INTEGER NOT NULL,
    market_state_before TEXT, -- JSON of market state before transaction
    market_state_after TEXT,  -- JSON of market state after transaction
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- Achievements - Master achievement definitions
CREATE TABLE IF NOT EXISTS achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    achievement_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    icon TEXT,
    points INTEGER DEFAULT 0,
    criteria TEXT, -- Description of criteria for earning
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Player achievements - Achievement progress and completion
CREATE TABLE IF NOT EXISTS player_achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    achievement_id INTEGER NOT NULL,
    game_id INTEGER, -- Game where achievement was earned (NULL for cumulative achievements)
    progress INTEGER DEFAULT 0,
    max_progress INTEGER DEFAULT 1,
    is_completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id),
    FOREIGN KEY (achievement_id) REFERENCES achievements(id),
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE SET NULL,
    UNIQUE(player_id, achievement_id)
);

-- Game events - General purpose event logging
CREATE TABLE IF NOT EXISTS game_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER, -- NULL for game-wide events
    event_type TEXT NOT NULL,
    event_data TEXT, -- JSON blob
    round_number INTEGER,
    phase TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);

-- Player statistics - Aggregated statistics (maintained via triggers/updates)
CREATE TABLE IF NOT EXISTS player_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER UNIQUE NOT NULL,
    
    -- Game statistics
    games_played INTEGER DEFAULT 0,
    games_won INTEGER DEFAULT 0,
    games_lost INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0.0,
    
    -- Performance metrics
    avg_final_cities REAL DEFAULT 0.0,
    avg_final_plants REAL DEFAULT 0.0,
    avg_final_money REAL DEFAULT 0.0,
    avg_game_duration_minutes REAL DEFAULT 0.0,
    
    -- Best performances
    max_cities_single_game INTEGER DEFAULT 0,
    max_plants_single_game INTEGER DEFAULT 0,
    max_money_single_game INTEGER DEFAULT 0,
    fastest_win_minutes INTEGER,
    longest_win_minutes INTEGER,
    
    -- Resource usage
    total_resources_bought INTEGER DEFAULT 0,
    total_money_spent INTEGER DEFAULT 0,
    
    -- Power plant statistics
    total_plants_owned INTEGER DEFAULT 0,
    avg_plant_efficiency REAL DEFAULT 0.0,
    
    -- City building
    total_cities_built INTEGER DEFAULT 0,
    avg_expansion_rate REAL DEFAULT 0.0,
    
    -- Achievements
    total_achievement_points INTEGER DEFAULT 0,
    total_achievements_earned INTEGER DEFAULT 0,
    
    -- Temporal statistics
    total_playtime_minutes INTEGER DEFAULT 0,
    avg_session_length_minutes REAL DEFAULT 0.0,
    
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
CREATE INDEX IF NOT EXISTS idx_players_last_seen ON players(last_seen);

CREATE INDEX IF NOT EXISTS idx_games_game_id ON games(game_id);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_started_at ON games(started_at);
CREATE INDEX IF NOT EXISTS idx_games_map_name ON games(map_name);

CREATE INDEX IF NOT EXISTS idx_game_participants_game_id ON game_participants(game_id);
CREATE INDEX IF NOT EXISTS idx_game_participants_player_id ON game_participants(player_id);
CREATE INDEX IF NOT EXISTS idx_game_participants_is_winner ON game_participants(is_winner);

CREATE INDEX IF NOT EXISTS idx_game_states_game_id ON game_states(game_id);
CREATE INDEX IF NOT EXISTS idx_game_states_round_phase ON game_states(round_number, phase);

CREATE INDEX IF NOT EXISTS idx_player_actions_game_player ON player_actions(game_id, player_id);
CREATE INDEX IF NOT EXISTS idx_player_actions_type ON player_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_player_actions_timestamp ON player_actions(timestamp);

CREATE INDEX IF NOT EXISTS idx_player_power_plants_game_player ON player_power_plants(game_id, player_id);
CREATE INDEX IF NOT EXISTS idx_player_power_plants_plant_id ON player_power_plants(plant_id);

CREATE INDEX IF NOT EXISTS idx_player_cities_game_player ON player_cities(game_id, player_id);
CREATE INDEX IF NOT EXISTS idx_player_cities_region ON player_cities(region);

CREATE INDEX IF NOT EXISTS idx_resource_transactions_game_player ON resource_transactions(game_id, player_id);
CREATE INDEX IF NOT EXISTS idx_resource_transactions_type ON resource_transactions(resource_type);

CREATE INDEX IF NOT EXISTS idx_achievements_category ON achievements(category);
CREATE INDEX IF NOT EXISTS idx_achievements_active ON achievements(is_active);

CREATE INDEX IF NOT EXISTS idx_player_achievements_player ON player_achievements(player_id);
CREATE INDEX IF NOT EXISTS idx_player_achievements_completed ON player_achievements(is_completed);

CREATE INDEX IF NOT EXISTS idx_game_events_game_id ON game_events(game_id);
CREATE INDEX IF NOT EXISTS idx_game_events_type ON game_events(event_type);
CREATE INDEX IF NOT EXISTS idx_game_events_timestamp ON game_events(created_at);

-- Views for common queries

-- Player leaderboard view
CREATE VIEW IF NOT EXISTS leaderboard AS
SELECT 
    p.id,
    p.name,
    ps.games_played,
    ps.games_won,
    ps.win_rate,
    ps.avg_final_cities,
    ps.total_achievement_points,
    (ps.games_won * 100 + ps.win_rate * 10 + ps.total_cities_built + ps.total_achievement_points) as composite_score,
    p.last_seen
FROM players p
JOIN player_statistics ps ON p.id = ps.player_id
WHERE ps.games_played >= 5  -- Minimum games for leaderboard
ORDER BY composite_score DESC;

-- Game summary view
CREATE VIEW IF NOT EXISTS game_summary AS
SELECT 
    g.id,
    g.game_id,
    g.name,
    g.map_name,
    g.actual_players,
    g.status,
    p.name as winner_name,
    g.duration_minutes,
    g.total_rounds,
    g.started_at,
    g.ended_at
FROM games g
LEFT JOIN players p ON g.winner_player_id = p.id;

-- Player game history view
CREATE VIEW IF NOT EXISTS player_game_history AS
SELECT 
    gp.player_name,
    gs.game_id,
    gs.name as game_name,
    gs.map_name,
    gs.duration_minutes,
    gs.started_at,
    gs.ended_at,
    gp.final_position,
    gp.final_cities,
    gp.final_plants,
    gp.final_money,
    gp.is_winner,
    gs.actual_players
FROM game_participants gp
JOIN game_summary gs ON gp.game_id = gs.id
WHERE gs.status = 'completed'
ORDER BY gs.ended_at DESC;

-- Recent activity view
CREATE VIEW IF NOT EXISTS recent_activity AS
SELECT 
    'game_completed' as activity_type,
    g.game_id as reference_id,
    'Game ' || g.name || ' completed' as description,
    p.name as player_name,
    g.ended_at as timestamp
FROM games g
JOIN players p ON g.winner_player_id = p.id
WHERE g.status = 'completed' AND g.ended_at > datetime('now', '-7 days')

UNION ALL

SELECT 
    'achievement_earned' as activity_type,
    a.achievement_id as reference_id,
    p.name || ' earned ' || a.name as description,
    p.name as player_name,
    pa.completed_at as timestamp
FROM player_achievements pa
JOIN players p ON pa.player_id = p.id
JOIN achievements a ON pa.achievement_id = a.id
WHERE pa.is_completed = TRUE AND pa.completed_at > datetime('now', '-7 days')

ORDER BY timestamp DESC;

-- Triggers to maintain player statistics

-- Update player statistics when game participation changes
CREATE TRIGGER IF NOT EXISTS update_player_stats_on_game_complete
AFTER UPDATE OF final_position ON game_participants
WHEN NEW.final_position IS NOT NULL AND OLD.final_position IS NULL
BEGIN
    INSERT OR REPLACE INTO player_statistics (
        player_id,
        games_played,
        games_won,
        games_lost,
        win_rate,
        avg_final_cities,
        avg_final_plants,
        avg_final_money,
        max_cities_single_game,
        max_plants_single_game,
        max_money_single_game,
        total_cities_built,
        last_updated
    )
    SELECT 
        NEW.player_id,
        COUNT(*) as games_played,
        COUNT(CASE WHEN is_winner = TRUE THEN 1 END) as games_won,
        COUNT(CASE WHEN is_winner = FALSE THEN 1 END) as games_lost,
        ROUND(COUNT(CASE WHEN is_winner = TRUE THEN 1 END) * 100.0 / COUNT(*), 2) as win_rate,
        ROUND(AVG(final_cities), 2) as avg_final_cities,
        ROUND(AVG(final_plants), 2) as avg_final_plants,
        ROUND(AVG(final_money), 2) as avg_final_money,
        MAX(final_cities) as max_cities_single_game,
        MAX(final_plants) as max_plants_single_game,
        MAX(final_money) as max_money_single_game,
        SUM(final_cities) as total_cities_built,
        CURRENT_TIMESTAMP
    FROM game_participants
    WHERE player_id = NEW.player_id AND final_position IS NOT NULL;
END;

-- Update player last_seen when they join a game
CREATE TRIGGER IF NOT EXISTS update_player_last_seen
AFTER INSERT ON game_participants
BEGIN
    UPDATE players 
    SET last_seen = CURRENT_TIMESTAMP,
        total_games = total_games + 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.player_id;
END;