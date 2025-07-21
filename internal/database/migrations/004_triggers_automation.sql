-- Add triggers for automatic statistics maintenance
-- This migration adds triggers to keep statistics up-to-date automatically

-- Trigger to update player statistics when game participation changes
CREATE TRIGGER update_player_stats_on_game_complete
AFTER UPDATE OF final_position ON game_participants
WHEN NEW.final_position IS NOT NULL AND OLD.final_position IS NULL
BEGIN
    -- Insert or update player statistics
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

-- Trigger to update player last_seen when they join a game
CREATE TRIGGER update_player_last_seen
AFTER INSERT ON game_participants
BEGIN
    UPDATE players 
    SET last_seen = CURRENT_TIMESTAMP,
        total_games = total_games + 1,
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.player_id;
END;

-- Trigger to update achievement statistics when achievements are earned
CREATE TRIGGER update_achievement_stats
AFTER UPDATE OF is_completed ON player_achievements
WHEN NEW.is_completed = TRUE AND OLD.is_completed = FALSE
BEGIN
    UPDATE player_statistics 
    SET total_achievements_earned = total_achievements_earned + 1,
        total_achievement_points = total_achievement_points + (
            SELECT points FROM achievements WHERE id = NEW.achievement_id
        ),
        last_updated = CURRENT_TIMESTAMP
    WHERE player_id = NEW.player_id;
END;

-- Trigger to update game duration when game ends
CREATE TRIGGER update_game_duration
AFTER UPDATE OF ended_at ON games
WHEN NEW.ended_at IS NOT NULL AND OLD.ended_at IS NULL
BEGIN
    UPDATE games 
    SET duration_minutes = CAST((julianday(NEW.ended_at) - julianday(started_at)) * 24 * 60 AS INTEGER),
        updated_at = CURRENT_TIMESTAMP
    WHERE id = NEW.id;
END;