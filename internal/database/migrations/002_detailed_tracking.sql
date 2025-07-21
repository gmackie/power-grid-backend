-- Add detailed game tracking tables
-- This migration adds tables for in-depth game analysis

-- Game states - Detailed game progression tracking
CREATE TABLE game_states (
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
CREATE TABLE player_actions (
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
CREATE TABLE player_power_plants (
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
CREATE TABLE player_cities (
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
CREATE TABLE resource_transactions (
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

-- Add indexes for the new tables
CREATE INDEX idx_game_states_game_id ON game_states(game_id);
CREATE INDEX idx_game_states_round_phase ON game_states(round_number, phase);
CREATE INDEX idx_player_actions_game_player ON player_actions(game_id, player_id);
CREATE INDEX idx_player_actions_type ON player_actions(action_type);
CREATE INDEX idx_player_actions_timestamp ON player_actions(timestamp);
CREATE INDEX idx_player_power_plants_game_player ON player_power_plants(game_id, player_id);
CREATE INDEX idx_player_cities_game_player ON player_cities(game_id, player_id);
CREATE INDEX idx_resource_transactions_game_player ON resource_transactions(game_id, player_id);