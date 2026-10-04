/** Dynamic player/game records in D1. */
import { Context, Effect, Layer, Schema } from "effect"
import { SqlClient } from "effect/sql"
import type { SqlError } from "effect/sql/SqlError"
import type { GameState } from "../domain/state.ts"

export interface ParticipantInput {
  readonly playerUuid: string
  readonly name: string
  readonly color: string
  readonly turnOrder: number
}

export interface RecordsShape {
  /** Upsert players by name and create the game + participants. */
  readonly gameStarted: (s: GameState) => Effect.Effect<void, SqlError>
  readonly logAction: (
    s: GameState,
    playerUuid: string,
    actionType: string,
    data: unknown,
    result: "success" | "failed",
    error?: string
  ) => Effect.Effect<void, SqlError>
  readonly logEvent: (gameId: string, eventType: string, data: unknown, round: number, phase: string) => Effect.Effect<void, SqlError>
  readonly gameCompleted: (s: GameState) => Effect.Effect<void, SqlError>
  readonly gameAbandoned: (gameId: string) => Effect.Effect<void, SqlError>
}

export class Records extends Context.Service<Records, RecordsShape>()("Records") {}

const IdRow = Schema.Struct({ id: Schema.Number })
const decodeIds = Schema.decodeUnknownSync(Schema.Array(IdRow))

export const RecordsLive = Layer.effect(
  Records,
  Effect.gen(function* () {
    const sql = yield* SqlClient.SqlClient

    const playerIdByName = (name: string, playerUuid: string) =>
      Effect.map(
        sql`INSERT INTO players (name, last_player_id, first_seen, last_seen, updated_at)
            VALUES (${name}, ${playerUuid}, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
            ON CONFLICT(name) DO UPDATE SET last_seen = CURRENT_TIMESTAMP, last_player_id = ${playerUuid}, updated_at = CURRENT_TIMESTAMP
            RETURNING id`,
        (rows) => decodeIds(rows)[0]?.id
      )

    const gameRowId = (gameId: string) =>
      Effect.map(sql`SELECT id FROM games WHERE game_id = ${gameId}`, (rows) => decodeIds(rows)[0]?.id)

    const gameStarted: RecordsShape["gameStarted"] = (s) =>
      Effect.gen(function* () {
        const n = Object.keys(s.players).length
        yield* sql`INSERT OR IGNORE INTO games (game_id, name, map_name, max_players, actual_players, status, started_at)
                   VALUES (${s.id}, ${s.name}, ${s.mapId}, 6, ${n}, 'playing', CURRENT_TIMESTAMP)`
        const gid = yield* gameRowId(s.id)
        if (gid === undefined) return
        for (const [i, pid] of s.turnOrder.entries()) {
          const p = s.players[pid]!
          const playerId = yield* playerIdByName(p.name, p.id)
          if (playerId === undefined) continue
          yield* sql`INSERT INTO game_participants (game_id, player_id, player_name, player_uuid, color, turn_order)
                     VALUES (${gid}, ${playerId}, ${p.name}, ${p.id}, ${p.color}, ${i + 1})
                     ON CONFLICT(game_id, player_id) DO UPDATE SET color = EXCLUDED.color, turn_order = EXCLUDED.turn_order, player_uuid = EXCLUDED.player_uuid`
        }
        yield* sql`INSERT INTO game_events (game_id, event_type, event_data, round_number, phase)
                   VALUES (${gid}, 'game_started', ${JSON.stringify({ turn_order: s.turnOrder, map: s.mapId })}, ${s.round}, ${s.phase})`
      })

    const logAction: RecordsShape["logAction"] = (s, playerUuid, actionType, data, result, error) =>
      Effect.gen(function* () {
        const gid = yield* gameRowId(s.id)
        if (gid === undefined) return
        const name = s.players[playerUuid]?.name
        const pidRows = name ? yield* sql`SELECT id FROM players WHERE name = ${name}` : []
        const playerId = decodeIds(pidRows)[0]?.id ?? null
        yield* sql`INSERT INTO player_actions (game_id, player_id, round_number, phase, action_type, action_data, action_result, error_message)
                   VALUES (${gid}, ${playerId}, ${s.round}, ${s.phase}, ${actionType}, ${JSON.stringify(data ?? null)}, ${result}, ${error ?? null})`
      })

    const logEvent: RecordsShape["logEvent"] = (gameId, eventType, data, round, phase) =>
      Effect.gen(function* () {
        const gid = yield* gameRowId(gameId)
        if (gid === undefined) return
        yield* sql`INSERT INTO game_events (game_id, event_type, event_data, round_number, phase)
                   VALUES (${gid}, ${eventType}, ${JSON.stringify(data ?? null)}, ${round}, ${phase})`
      })

    const gameCompleted: RecordsShape["gameCompleted"] = (s) =>
      Effect.gen(function* () {
        const gid = yield* gameRowId(s.id)
        if (gid === undefined) return
        const ranked = Object.values(s.players).sort(
          (a, b) => b.poweredCities - a.poweredCities || b.money - a.money || b.cities.length - a.cities.length
        )
        for (const [i, p] of ranked.entries()) {
          const totalRes = Object.values(p.resources).reduce((a, b) => a + b, 0)
          const isWinner = p.id === s.winnerId
          yield* sql`UPDATE game_participants SET final_position = ${i + 1}, final_cities = ${p.cities.length},
                       final_plants = ${p.plants.length}, final_money = ${p.money}, final_resources = ${totalRes},
                       powered_cities = ${p.poweredCities}, is_winner = ${isWinner}
                     WHERE game_id = ${gid} AND player_name = ${p.name}`
        }
        const winnerName = s.winnerId ? s.players[s.winnerId]?.name : undefined
        const winnerRows = winnerName ? yield* sql`SELECT id FROM players WHERE name = ${winnerName}` : []
        const winnerId = decodeIds(winnerRows)[0]?.id ?? null
        yield* sql`UPDATE games SET status = 'completed', ended_at = CURRENT_TIMESTAMP, winner_player_id = ${winnerId},
                     total_rounds = ${s.round}, final_step = ${s.step}, updated_at = CURRENT_TIMESTAMP
                   WHERE id = ${gid}`
        yield* sql`INSERT INTO game_events (game_id, event_type, event_data, round_number, phase)
                   VALUES (${gid}, 'game_completed', ${JSON.stringify({ winner: s.winnerId, total_rounds: s.round })}, ${s.round}, 'completed')`
        // Simple per-game achievements (same four the Go service awarded).
        for (const p of ranked) {
          const earned: Array<string> = []
          if (p.id === s.winnerId) earned.push("first_win")
          if (p.cities.length >= 15) earned.push("city_builder")
          if (p.money >= 200) earned.push("money_bags")
          if (p.plants.some((pl) => pl.capacity >= 7)) earned.push("high_capacity")
          if (new Set(p.plants.map((pl) => pl.resourceType)).size >= 4) earned.push("diversified")
          if (p.id === s.winnerId && p.poweredCities === p.cities.length && p.cities.length > 0) earned.push("perfectionist")
          for (const key of earned) {
            yield* sql`INSERT OR IGNORE INTO player_achievements (player_id, achievement_id, game_id, progress, max_progress, is_completed, completed_at)
                       SELECT pl.id, a.id, ${gid}, 1, 1, TRUE, CURRENT_TIMESTAMP FROM players pl, achievements a
                       WHERE pl.name = ${p.name} AND a.achievement_id = ${key}`
          }
        }
      })

    const gameAbandoned: RecordsShape["gameAbandoned"] = (gameId) =>
      Effect.asVoid(
        sql`UPDATE games SET status = 'abandoned', ended_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
            WHERE game_id = ${gameId} AND status = 'playing'`
      )

    return Records.of({ gameStarted, logAction, logEvent, gameCompleted, gameAbandoned })
  })
)
