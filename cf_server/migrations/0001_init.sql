-- Power Grid D1 schema. Adapted from go_server/internal/database/migrations/001-004.
-- Players and game records are dynamic; maps, decks and achievements are game assets
-- that admins can change without a redeploy.

CREATE TABLE IF NOT EXISTS players (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT UNIQUE NOT NULL,
    last_player_id TEXT,                      -- most recent lobby/game player uuid
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
CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);
CREATE INDEX IF NOT EXISTS idx_players_last_seen ON players(last_seen);

CREATE TABLE IF NOT EXISTS games (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id TEXT UNIQUE NOT NULL,             -- Durable Object name / wire game_id
    name TEXT NOT NULL,
    map_name TEXT NOT NULL,
    max_players INTEGER DEFAULT 6,
    actual_players INTEGER NOT NULL,
    status TEXT DEFAULT 'lobby',              -- lobby, playing, completed, abandoned
    winner_player_id INTEGER,
    started_at TIMESTAMP,
    ended_at TIMESTAMP,
    duration_minutes INTEGER,
    total_rounds INTEGER DEFAULT 0,
    final_step INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (winner_player_id) REFERENCES players(id)
);
CREATE INDEX IF NOT EXISTS idx_games_game_id ON games(game_id);
CREATE INDEX IF NOT EXISTS idx_games_status ON games(status);
CREATE INDEX IF NOT EXISTS idx_games_started_at ON games(started_at);
CREATE INDEX IF NOT EXISTS idx_games_map_name ON games(map_name);

CREATE TABLE IF NOT EXISTS game_participants (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER NOT NULL,
    player_name TEXT NOT NULL,
    player_uuid TEXT,                         -- wire player_id inside this game
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
CREATE INDEX IF NOT EXISTS idx_game_participants_game_id ON game_participants(game_id);
CREATE INDEX IF NOT EXISTS idx_game_participants_player_id ON game_participants(player_id);

CREATE TABLE IF NOT EXISTS player_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER,
    round_number INTEGER NOT NULL,
    phase TEXT NOT NULL,
    action_type TEXT NOT NULL,                -- BID_PLANT, BUY_RESOURCES, BUILD_CITY, POWER_CITIES, END_TURN
    action_data TEXT,                         -- JSON
    action_result TEXT,                       -- success | failed
    error_message TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);
CREATE INDEX IF NOT EXISTS idx_player_actions_game_player ON player_actions(game_id, player_id);
CREATE INDEX IF NOT EXISTS idx_player_actions_type ON player_actions(action_type);

CREATE TABLE IF NOT EXISTS game_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    game_id INTEGER NOT NULL,
    player_id INTEGER,
    event_type TEXT NOT NULL,                 -- game_started, phase_change, step_change, game_completed, ...
    event_data TEXT,                          -- JSON
    round_number INTEGER,
    phase TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (game_id) REFERENCES games(id) ON DELETE CASCADE,
    FOREIGN KEY (player_id) REFERENCES players(id)
);
CREATE INDEX IF NOT EXISTS idx_game_events_game_id ON game_events(game_id);
CREATE INDEX IF NOT EXISTS idx_game_events_type ON game_events(event_type);

CREATE TABLE IF NOT EXISTS achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    achievement_id TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    category TEXT NOT NULL,
    icon TEXT,
    points INTEGER DEFAULT 0,
    criteria TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_achievements_category ON achievements(category);

CREATE TABLE IF NOT EXISTS player_achievements (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER NOT NULL,
    achievement_id INTEGER NOT NULL,
    game_id INTEGER,
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
CREATE INDEX IF NOT EXISTS idx_player_achievements_player ON player_achievements(player_id);

CREATE TABLE IF NOT EXISTS player_statistics (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    player_id INTEGER UNIQUE NOT NULL,
    games_played INTEGER DEFAULT 0,
    games_won INTEGER DEFAULT 0,
    games_lost INTEGER DEFAULT 0,
    win_rate REAL DEFAULT 0.0,                -- fraction 0..1 (Go stored a percentage; fixed here)
    avg_final_cities REAL DEFAULT 0.0,
    avg_final_plants REAL DEFAULT 0.0,
    avg_final_money REAL DEFAULT 0.0,
    avg_game_duration_minutes REAL DEFAULT 0.0,
    max_cities_single_game INTEGER DEFAULT 0,
    max_plants_single_game INTEGER DEFAULT 0,
    max_money_single_game INTEGER DEFAULT 0,
    fastest_win_minutes INTEGER,
    longest_win_minutes INTEGER,
    total_cities_built INTEGER DEFAULT 0,
    total_achievement_points INTEGER DEFAULT 0,
    total_achievements_earned INTEGER DEFAULT 0,
    total_playtime_minutes INTEGER DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);

-- Game assets --------------------------------------------------------------

CREATE TABLE IF NOT EXISTS maps (
    id TEXT PRIMARY KEY,                      -- "usa", "germany"
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    player_min INTEGER NOT NULL DEFAULT 2,
    player_max INTEGER NOT NULL DEFAULT 6,
    data TEXT NOT NULL,                       -- full MapData JSON (regions, cities, connections, gameRules)
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    version INTEGER NOT NULL DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS power_plant_decks (
    id TEXT PRIMARY KEY,                      -- "standard"
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    is_default BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS power_plants (
    deck_id TEXT NOT NULL,
    number INTEGER NOT NULL,                  -- plant number == minimum bid == wire id
    capacity INTEGER NOT NULL,                -- cities powered
    resource_type TEXT NOT NULL,              -- Coal | Oil | Garbage | Uranium | Hybrid | Wind
    resource_cost INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (deck_id, number),
    FOREIGN KEY (deck_id) REFERENCES power_plant_decks(id) ON DELETE CASCADE
);

-- Views ---------------------------------------------------------------------

CREATE VIEW IF NOT EXISTS leaderboard AS
SELECT p.id, p.name,
    COALESCE(ps.games_played, 0) AS games_played,
    COALESCE(ps.games_won, 0) AS games_won,
    COALESCE(ps.win_rate, 0.0) AS win_rate,
    COALESCE(ps.avg_final_cities, 0.0) AS avg_final_cities,
    COALESCE(ps.total_achievement_points, 0) AS total_achievement_points,
    COALESCE(ps.total_cities_built, 0) AS total_cities_built,
    (COALESCE(ps.games_won, 0) * 100 + COALESCE(ps.win_rate, 0) * 1000 +
     COALESCE(ps.total_cities_built, 0) + COALESCE(ps.total_achievement_points, 0)) AS composite_score,
    p.last_seen
FROM players p
LEFT JOIN player_statistics ps ON p.id = ps.player_id;

CREATE VIEW IF NOT EXISTS game_summary AS
SELECT g.id, g.game_id, g.name, g.map_name, g.actual_players, g.status,
    p.name AS winner_name, g.duration_minutes, g.total_rounds, g.started_at, g.ended_at
FROM games g
LEFT JOIN players p ON g.winner_player_id = p.id;

-- Triggers ------------------------------------------------------------------

CREATE TRIGGER IF NOT EXISTS trg_participant_inserted
AFTER INSERT ON game_participants
BEGIN
    UPDATE players SET last_seen = CURRENT_TIMESTAMP, total_games = total_games + 1,
        updated_at = CURRENT_TIMESTAMP WHERE id = NEW.player_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_participant_finalized
AFTER UPDATE OF final_position ON game_participants
WHEN NEW.final_position IS NOT NULL AND OLD.final_position IS NULL
BEGIN
    INSERT INTO player_statistics (player_id) VALUES (NEW.player_id)
        ON CONFLICT(player_id) DO NOTHING;
    UPDATE player_statistics SET
        games_played = (SELECT COUNT(*) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        games_won = (SELECT COUNT(*) FROM game_participants WHERE player_id = NEW.player_id AND is_winner = TRUE),
        games_lost = (SELECT COUNT(*) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL AND is_winner = FALSE),
        win_rate = (SELECT ROUND(COUNT(CASE WHEN is_winner = TRUE THEN 1 END) * 1.0 / MAX(COUNT(*), 1), 4)
                    FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        avg_final_cities = (SELECT ROUND(AVG(final_cities), 2) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        avg_final_plants = (SELECT ROUND(AVG(final_plants), 2) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        avg_final_money = (SELECT ROUND(AVG(final_money), 2) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        max_cities_single_game = (SELECT MAX(final_cities) FROM game_participants WHERE player_id = NEW.player_id),
        max_plants_single_game = (SELECT MAX(final_plants) FROM game_participants WHERE player_id = NEW.player_id),
        max_money_single_game = (SELECT MAX(final_money) FROM game_participants WHERE player_id = NEW.player_id),
        total_cities_built = (SELECT SUM(final_cities) FROM game_participants WHERE player_id = NEW.player_id AND final_position IS NOT NULL),
        last_updated = CURRENT_TIMESTAMP
    WHERE player_id = NEW.player_id;
    UPDATE players SET total_wins = total_wins + (CASE WHEN NEW.is_winner = TRUE THEN 1 ELSE 0 END),
        updated_at = CURRENT_TIMESTAMP WHERE id = NEW.player_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_achievement_completed
AFTER INSERT ON player_achievements
WHEN NEW.is_completed = TRUE
BEGIN
    INSERT INTO player_statistics (player_id) VALUES (NEW.player_id)
        ON CONFLICT(player_id) DO NOTHING;
    UPDATE player_statistics
    SET total_achievements_earned = total_achievements_earned + 1,
        total_achievement_points = total_achievement_points + (SELECT points FROM achievements WHERE id = NEW.achievement_id),
        last_updated = CURRENT_TIMESTAMP
    WHERE player_id = NEW.player_id;
END;

CREATE TRIGGER IF NOT EXISTS trg_game_ended
AFTER UPDATE OF ended_at ON games
WHEN NEW.ended_at IS NOT NULL AND OLD.ended_at IS NULL
BEGIN
    UPDATE games
    SET duration_minutes = CAST((julianday(NEW.ended_at) - julianday(COALESCE(started_at, created_at))) * 24 * 60 AS INTEGER),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;
