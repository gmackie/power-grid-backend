-- Add statistics tables and views
-- This migration adds aggregated statistics and useful views

-- Player statistics - Aggregated statistics (maintained via triggers/updates)
CREATE TABLE player_statistics (
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

-- Game events - General purpose event logging
CREATE TABLE game_events (
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

-- Views for common queries

-- Player leaderboard view
CREATE VIEW leaderboard AS
SELECT 
    p.id,
    p.name,
    COALESCE(ps.games_played, 0) as games_played,
    COALESCE(ps.games_won, 0) as games_won,
    COALESCE(ps.win_rate, 0.0) as win_rate,
    COALESCE(ps.avg_final_cities, 0.0) as avg_final_cities,
    COALESCE(ps.total_achievement_points, 0) as total_achievement_points,
    COALESCE(ps.total_cities_built, 0) as total_cities_built,
    (COALESCE(ps.games_won, 0) * 100 + COALESCE(ps.win_rate, 0) * 10 + 
     COALESCE(ps.total_cities_built, 0) + COALESCE(ps.total_achievement_points, 0)) as composite_score,
    p.last_seen
FROM players p
LEFT JOIN player_statistics ps ON p.id = ps.player_id
WHERE COALESCE(ps.games_played, 0) >= 5  -- Minimum games for leaderboard
ORDER BY composite_score DESC;

-- Game summary view
CREATE VIEW game_summary AS
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
CREATE VIEW player_game_history AS
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

-- Add indexes for statistics and events
CREATE INDEX idx_player_statistics_player_id ON player_statistics(player_id);
CREATE INDEX idx_game_events_game_id ON game_events(game_id);
CREATE INDEX idx_game_events_type ON game_events(event_type);
CREATE INDEX idx_game_events_timestamp ON game_events(created_at);