-- Initial database schema for Power Grid analytics
-- This creates all the core tables for comprehensive game tracking

-- Players table - Master player registry
CREATE TABLE players (
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
CREATE TABLE games (
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
CREATE TABLE game_participants (
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

-- Achievements - Master achievement definitions
CREATE TABLE achievements (
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
CREATE TABLE player_achievements (
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

-- Basic indexes for performance
CREATE INDEX idx_players_name ON players(name);
CREATE INDEX idx_players_last_seen ON players(last_seen);
CREATE INDEX idx_games_game_id ON games(game_id);
CREATE INDEX idx_games_status ON games(status);
CREATE INDEX idx_games_started_at ON games(started_at);
CREATE INDEX idx_game_participants_game_id ON game_participants(game_id);
CREATE INDEX idx_game_participants_player_id ON game_participants(player_id);
CREATE INDEX idx_player_achievements_player ON player_achievements(player_id);