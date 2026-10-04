/** Read-side analytics queries for the /api endpoints (wire shapes follow handlers/analytics_db.go). */
import { Context, Effect, Layer, Schema } from "effect"
import { SqlClient } from "effect/sql"
import type { SqlError } from "effect/sql/SqlError"

export class NotFound extends Schema.TaggedError<NotFound>()("NotFound", { message: Schema.String }) {}

const Rows = Schema.Array(Schema.Record(Schema.String, Schema.Unknown))
const rows = Schema.decodeUnknownSync(Rows)
const num = (v: unknown): number => (typeof v === "number" ? v : typeof v === "string" ? Number(v) : 0)
const str = (v: unknown): string | null => (v === null || v === undefined ? null : String(v))
const bool = (v: unknown): boolean => v === 1 || v === true || v === "1"

export interface PlayerStats {
  readonly id: number
  readonly player_id: number
  readonly player_name: string
  readonly games_played: number
  readonly games_won: number
  readonly games_lost: number
  readonly win_rate: number
  readonly avg_final_cities: number
  readonly avg_final_plants: number
  readonly avg_final_money: number
  readonly max_cities_single_game: number
  readonly max_plants_single_game: number
  readonly max_money_single_game: number
  readonly total_cities_built: number
  readonly total_achievement_points: number
  readonly total_achievements_earned: number
  readonly total_playtime_minutes: number
  readonly first_seen: string | null
  readonly last_seen: string | null
  readonly last_updated: string | null
}

export interface AnalyticsShape {
  readonly playerStats: (name: string) => Effect.Effect<PlayerStats, SqlError | NotFound>
  readonly playerAchievements: (name: string) => Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError | NotFound>
  readonly playerHistory: (name: string, limit: number) => Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError | NotFound>
  readonly listPlayers: (limit: number, offset: number) => Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError>
  readonly leaderboard: (limit: number) => Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError>
  readonly achievements: Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError>
  readonly achievementStats: Effect.Effect<Record<string, unknown>, SqlError>
  readonly gameAnalytics: (days: number) => Effect.Effect<Record<string, unknown>, SqlError>
  readonly activity: (days: number) => Effect.Effect<Record<string, unknown>, SqlError>
  readonly recentGames: (limit: number, status?: string) => Effect.Effect<ReadonlyArray<Record<string, unknown>>, SqlError>
  readonly game: (gameId: string) => Effect.Effect<Record<string, unknown>, SqlError | NotFound>
}

export class Analytics extends Context.Service<Analytics, AnalyticsShape>()("Analytics") {}

export const AnalyticsLive = Layer.effect(
  Analytics,
  Effect.gen(function* () {
    const sql = yield* SqlClient.SqlClient
    const since = (days: number) => new Date(Date.now() - days * 86_400_000).toISOString().replace("T", " ").slice(0, 19)

    const playerRow = (name: string) =>
      Effect.flatMap(sql`SELECT id, name, first_seen, last_seen FROM players WHERE name = ${name}`, (r) => {
        const row = rows(r)[0]
        return row ? Effect.succeed(row) : Effect.fail(new NotFound({ message: "Player not found" }))
      })

    const playerStats: AnalyticsShape["playerStats"] = (name) =>
      Effect.gen(function* () {
        const p = yield* playerRow(name)
        const id = num(p["id"])
        const s = rows(yield* sql`SELECT * FROM player_statistics WHERE player_id = ${id}`)[0] ?? {}
        return {
          id: num(s["id"]),
          player_id: id,
          player_name: name,
          games_played: num(s["games_played"]),
          games_won: num(s["games_won"]),
          games_lost: num(s["games_lost"]),
          win_rate: num(s["win_rate"]),
          avg_final_cities: num(s["avg_final_cities"]),
          avg_final_plants: num(s["avg_final_plants"]),
          avg_final_money: num(s["avg_final_money"]),
          max_cities_single_game: num(s["max_cities_single_game"]),
          max_plants_single_game: num(s["max_plants_single_game"]),
          max_money_single_game: num(s["max_money_single_game"]),
          total_cities_built: num(s["total_cities_built"]),
          total_achievement_points: num(s["total_achievement_points"]),
          total_achievements_earned: num(s["total_achievements_earned"]),
          total_playtime_minutes: num(s["total_playtime_minutes"]),
          first_seen: str(p["first_seen"]),
          last_seen: str(p["last_seen"]),
          last_updated: str(s["last_updated"])
        }
      })

    const playerAchievements: AnalyticsShape["playerAchievements"] = (name) =>
      Effect.gen(function* () {
        const p = yield* playerRow(name)
        const r = yield* sql`SELECT pa.id, pa.player_id, pa.achievement_id, pa.game_id, pa.progress, pa.max_progress, pa.is_completed,
                                    pa.completed_at, pa.created_at, pa.updated_at,
                                    a.achievement_id AS key, a.name, a.description, a.category, a.icon, a.points, a.criteria
                             FROM player_achievements pa JOIN achievements a ON a.id = pa.achievement_id
                             WHERE pa.player_id = ${num(p["id"])}
                             ORDER BY pa.is_completed DESC, a.category, a.points DESC`
        return rows(r).map((x) => ({
          id: num(x["id"]),
          player_id: num(x["player_id"]),
          achievement_id: num(x["achievement_id"]),
          game_id: x["game_id"] ?? null,
          progress: num(x["progress"]),
          max_progress: num(x["max_progress"]),
          is_completed: bool(x["is_completed"]),
          completed_at: str(x["completed_at"]),
          created_at: str(x["created_at"]),
          updated_at: str(x["updated_at"]),
          achievement: {
            achievement_id: str(x["key"]),
            name: str(x["name"]),
            description: str(x["description"]),
            category: str(x["category"]),
            icon: str(x["icon"]),
            points: num(x["points"]),
            criteria: str(x["criteria"])
          }
        }))
      })

    const playerHistory: AnalyticsShape["playerHistory"] = (name, limit) =>
      Effect.gen(function* () {
        yield* playerRow(name)
        const r = yield* sql`SELECT g.game_id, g.name AS game_name, g.map_name, g.duration_minutes, g.started_at, g.ended_at, g.actual_players,
                                    gp.final_position, gp.final_cities, gp.final_plants, gp.final_money, gp.is_winner, gp.color
                             FROM game_participants gp JOIN games g ON g.id = gp.game_id
                             WHERE gp.player_name = ${name} AND g.status = 'completed'
                             ORDER BY g.ended_at DESC LIMIT ${limit}`
        return rows(r).map((x) => ({ ...x, is_winner: bool(x["is_winner"]) }))
      })

    const listPlayers: AnalyticsShape["listPlayers"] = (limit, offset) =>
      Effect.map(
        sql`SELECT p.id, p.name, p.first_seen, p.last_seen, p.total_games, p.total_wins,
                   COALESCE(ps.win_rate, 0) AS win_rate, COALESCE(ps.total_achievement_points, 0) AS total_achievement_points
            FROM players p LEFT JOIN player_statistics ps ON ps.player_id = p.id
            ORDER BY p.last_seen DESC LIMIT ${limit} OFFSET ${offset}`,
        rows
      )

    const leaderboard: AnalyticsShape["leaderboard"] = (limit) =>
      Effect.map(
        sql`SELECT id AS player_id, name AS player_name, games_played, games_won, win_rate, avg_final_cities,
                   total_achievement_points, total_cities_built, composite_score, last_seen
            FROM leaderboard WHERE games_played >= 1 ORDER BY composite_score DESC LIMIT ${limit}`,
        rows
      )

    const achievements: AnalyticsShape["achievements"] = Effect.map(
      sql`SELECT id, achievement_id, name, description, category, icon, points, criteria, is_active, created_at
          FROM achievements WHERE is_active = TRUE ORDER BY category, points DESC`,
      (r) => rows(r).map((x) => ({ ...x, is_active: bool(x["is_active"]) }))
    )

    const achievementStats: AnalyticsShape["achievementStats"] = Effect.gen(function* () {
      const totals = rows(
        yield* sql`SELECT (SELECT COUNT(*) FROM achievements WHERE is_active = TRUE) AS total_achievements,
                          (SELECT COUNT(*) FROM player_achievements WHERE is_completed = TRUE) AS total_completions,
                          (SELECT COUNT(DISTINCT player_id) FROM player_achievements WHERE is_completed = TRUE) AS players_with_achievements`
      )[0] ?? {}
      const rarest = rows(
        yield* sql`SELECT a.name, COUNT(pa.id) AS completion_count FROM achievements a
                   LEFT JOIN player_achievements pa ON pa.achievement_id = a.id AND pa.is_completed = TRUE
                   WHERE a.is_active = TRUE GROUP BY a.id, a.name ORDER BY completion_count ASC, a.points DESC LIMIT 1`
      )[0]
      const cats = rows(
        yield* sql`SELECT a.category, COUNT(DISTINCT a.id) AS total_achievements,
                          COUNT(CASE WHEN pa.is_completed = TRUE THEN 1 END) AS total_completions,
                          SUM(a.points) AS total_points_available,
                          SUM(CASE WHEN pa.is_completed = TRUE THEN a.points ELSE 0 END) AS points_earned
                   FROM achievements a LEFT JOIN player_achievements pa ON pa.achievement_id = a.id
                   WHERE a.is_active = TRUE GROUP BY a.category`
      )
      const category_stats: Record<string, unknown> = {}
      for (const c of cats) {
        const total = num(c["total_achievements"])
        const done = num(c["total_completions"])
        category_stats[String(c["category"])] = {
          total_achievements: total,
          total_completions: done,
          completion_rate: total > 0 ? done / total : 0,
          total_points_available: num(c["total_points_available"]),
          points_earned: num(c["points_earned"])
        }
      }
      return {
        total_achievements: num(totals["total_achievements"]),
        total_completions: num(totals["total_completions"]),
        players_with_achievements: num(totals["players_with_achievements"]),
        rarest_achievement: rarest ? str(rarest["name"]) : null,
        rarest_completion_count: rarest ? num(rarest["completion_count"]) : 0,
        category_stats
      }
    })

    const gameAnalytics: AnalyticsShape["gameAnalytics"] = (days) =>
      Effect.gen(function* () {
        const t = since(days)
        const base = rows(
          yield* sql`SELECT COUNT(*) AS total_games,
                            COUNT(CASE WHEN status = 'completed' THEN 1 END) AS completed_games,
                            COALESCE(AVG(CASE WHEN status = 'completed' THEN duration_minutes END), 0) AS avg_game_duration_minutes
                     FROM games WHERE created_at >= ${t}`
        )[0] ?? {}
        const maps = rows(yield* sql`SELECT map_name, COUNT(*) AS n FROM games WHERE created_at >= ${t} GROUP BY map_name`)
        const counts = rows(yield* sql`SELECT actual_players, COUNT(*) AS n FROM games WHERE created_at >= ${t} GROUP BY actual_players`)
        const recent = rows(
          yield* sql`SELECT g.game_id, g.name, g.map_name, g.actual_players, g.status, g.duration_minutes, g.total_rounds, g.started_at, g.ended_at,
                            p.name AS winner_name
                     FROM games g LEFT JOIN players p ON p.id = g.winner_player_id
                     WHERE g.status = 'completed' ORDER BY g.ended_at DESC LIMIT 10`
        )
        return {
          total_games: num(base["total_games"]),
          completed_games: num(base["completed_games"]),
          avg_game_duration_minutes: num(base["avg_game_duration_minutes"]),
          map_popularity: Object.fromEntries(maps.map((m) => [String(m["map_name"]), num(m["n"])])),
          player_counts: Object.fromEntries(counts.map((c) => [String(c["actual_players"]), num(c["n"])])),
          recent_games: recent
        }
      })

    const activity: AnalyticsShape["activity"] = (days) =>
      Effect.gen(function* () {
        const t = since(days)
        const base = rows(
          yield* sql`SELECT COUNT(DISTINCT g.id) AS total_games, COUNT(DISTINCT gp.player_id) AS unique_players,
                            COALESCE(AVG(g.duration_minutes), 0) AS avg_game_duration, COUNT(DISTINCT DATE(g.created_at)) AS active_days
                     FROM games g LEFT JOIN game_participants gp ON gp.game_id = g.id WHERE g.created_at >= ${t}`
        )[0] ?? {}
        const daily = rows(
          yield* sql`SELECT DATE(created_at) AS date, COUNT(*) AS games FROM games WHERE created_at >= ${t}
                     GROUP BY DATE(created_at) ORDER BY date DESC LIMIT 30`
        )
        return {
          total_games: num(base["total_games"]),
          unique_players: num(base["unique_players"]),
          active_days: num(base["active_days"]),
          avg_game_duration: num(base["avg_game_duration"]),
          daily_activity: daily.map((d) => ({ date: str(d["date"]), games: num(d["games"]) })),
          time_frame_days: days
        }
      })

    const recentGames: AnalyticsShape["recentGames"] = (limit, status) =>
      Effect.map(
        status
          ? sql`SELECT * FROM game_summary WHERE status = ${status} ORDER BY started_at DESC LIMIT ${limit}`
          : sql`SELECT * FROM game_summary ORDER BY started_at DESC LIMIT ${limit}`,
        rows
      )

    const game: AnalyticsShape["game"] = (gameId) =>
      Effect.gen(function* () {
        const g = rows(yield* sql`SELECT * FROM game_summary WHERE game_id = ${gameId}`)[0]
        if (!g) return yield* Effect.fail(new NotFound({ message: "Game not found" }))
        const participants = rows(
          yield* sql`SELECT gp.player_name, gp.player_uuid, gp.color, gp.turn_order, gp.final_position, gp.final_cities, gp.final_plants,
                            gp.final_money, gp.final_resources, gp.powered_cities, gp.is_winner
                     FROM game_participants gp JOIN games g ON g.id = gp.game_id WHERE g.game_id = ${gameId} ORDER BY gp.turn_order`
        ).map((x) => ({ ...x, is_winner: bool(x["is_winner"]) }))
        const events = rows(
          yield* sql`SELECT e.event_type, e.event_data, e.round_number, e.phase, e.created_at FROM game_events e
                     JOIN games g ON g.id = e.game_id WHERE g.game_id = ${gameId} ORDER BY e.id`
        )
        return { ...g, participants, events }
      })

    return Analytics.of({
      playerStats,
      playerAchievements,
      playerHistory,
      listPlayers,
      leaderboard,
      achievements,
      achievementStats,
      gameAnalytics,
      activity,
      recentGames,
      game
    })
  })
)
