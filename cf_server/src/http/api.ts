import { Schema as S } from "effect"
import { HttpApi, HttpApiEndpoint, HttpApiGroup, HttpApiSchema, HttpApiMiddleware, HttpApiSecurity, OpenApi } from "effect/http-api"
import { MapData } from "../domain/map.ts"
import { MapInfo, LobbySummary } from "../protocol/lobby.ts"
import { GameStatePayload } from "../protocol/game.ts"
const Row=S.Record(S.String, S.Unknown)
const Rows=S.Array(Row)
const Status=S.Struct({ status: S.String })
const Mutation=S.Struct({ status: S.String, id: S.String })
const Lobbies=S.Struct({ lobbies: S.Array(LobbySummary) })
const PlayerStats=S.Struct({  
id: S.Number,
  player_id: S.Number,
  player_name: S.String,
  games_played: S.Number,
  games_won: S.Number,
  games_lost: S.Number,
  win_rate: S.Number,
  avg_final_cities: S.Number,
  avg_final_plants: S.Number,
  avg_final_money: S.Number,
  max_cities_single_game: S.Number,
  max_plants_single_game: S.Number,
  max_money_single_game: S.Number,
  total_cities_built: S.Number,
  total_achievement_points: S.Number,
  total_achievements_earned: S.Number,
  total_playtime_minutes: S.Number,
  first_seen: S.NullOr(S.String),
  last_seen: S.NullOr(S.String),
  last_updated: S.NullOr(S.String),
})
export const DeckBody=S.Struct({
  name: S.String,
  description: S.optional(S.String),
  is_default: S.optional(S.Boolean),
  plants: S.Array(
    S.Struct({
      number: S.Number,
      capacity: S.Number,
      resourceType: S.Literals(["Coal", "Oil", "Garbage", "Uranium", "Hybrid", "Wind"]),
      resourceCost: S.Number
    })
  )
})
const Plant=DeckBody.fields.plants.value
const { id: _mapId, ...mapFields }=MapData.fields
export const MapBody=S.Struct(mapFields)
export class ApiNotFound extends S.TaggedError<ApiNotFound>()("ApiNotFound", { error: S.String, status: S.Number, timestamp: S.String }) { }
export class ApiFailure extends S.TaggedError<ApiFailure>()("ApiFailure", { error: S.String, status: S.Number, timestamp: S.String }) { }
export class Unauthorized extends S.TaggedError<Unauthorized>()("Unauthorized", { error: S.String, status: S.Number }, { httpApiStatus: 401 }) { }
export class AdminAuthorization extends HttpApiMiddleware.Service<AdminAuthorization>()("AdminAuthorization", {
  security: { bearer: HttpApiSecurity.bearer }, error: Unauthorized
}) { }
const errors=[ApiNotFound.pipe(HttpApiSchema.status(404)), ApiFailure.pipe(HttpApiSchema.status(500))]
export class PowerGridApi extends HttpApi.make("powerGrid").add(
  HttpApiGroup.make("server").add(
    HttpApiEndpoint.get("info", "/", { success: S.Struct({ name: S.String, version: S.String, status: S.String }), error: errors }),
    HttpApiEndpoint.get("health", "/health", { success: Status, error: errors }),
    HttpApiEndpoint.get("ready", "/ready", { success: Status, error: errors }),
    HttpApiEndpoint.get("analyticsHealth", "/api/health", { success: S.Struct({ status: S.String, service: S.String, timestamp: S.String, version: S.String }), error: errors }),
    HttpApiEndpoint.get("maps", "/maps", { success: S.Struct({ maps: S.Array(MapInfo) }), error: errors }),
    HttpApiEndpoint.get("map", "/maps/:id", { success: MapData, error: errors, params: { id: S.String } }),
    HttpApiEndpoint.get("players", "/api/players", { success: S.Struct({ players: Rows, count: S.Number }), error: errors, query: { limit: S.optional(S.String), offset: S.optional(S.String) } }),
    HttpApiEndpoint.get("player", "/api/players/:name", { success: PlayerStats, error: errors, params: { name: S.String } }),
    HttpApiEndpoint.get("playerStats", "/api/players/:name/stats", { success: PlayerStats, error: errors, params: { name: S.String } }),
    HttpApiEndpoint.get("playerAchievements", "/api/players/:name/achievements", { success: S.Struct({ player_name: S.String, achievements: Rows, total_count: S.Number }), error: errors, params: { name: S.String } }),
    HttpApiEndpoint.get("playerHistory", "/api/players/:name/history", { success: S.Struct({ player_name: S.String, game_count: S.Number, games: Rows }), error: errors, params: { name: S.String }, query: { limit: S.optional(S.String) } }),
    HttpApiEndpoint.get("achievements", "/api/achievements", { success: Rows, error: errors }),
    HttpApiEndpoint.get("leaderboard", "/api/leaderboard", { success: S.Struct({ leaderboard: Rows, limit: S.Number, count: S.Number, updated_at: S.String }), error: errors, query: { limit: S.optional(S.String) } }),
    HttpApiEndpoint.get("gameAnalytics", "/api/analytics/games", { success: S.Struct({ analytics: Row, period_days: S.Number, updated_at: S.String }), error: errors, query: { days: S.optional(S.String) } }),
    HttpApiEndpoint.get("achievementStats", "/api/analytics/achievements", { success: S.Struct({ stats: Row, updated_at: S.String }), error: errors }),
    HttpApiEndpoint.get("activity", "/api/analytics/activity", { success: Row, error: errors, query: { days: S.optional(S.String) } }),
    HttpApiEndpoint.get("games", "/api/games", { success: S.Struct({ games: Rows, count: S.Number }), error: errors, query: { limit: S.optional(S.String), status: S.optional(S.String) } }),
    HttpApiEndpoint.get("game", "/api/games/:id", { success: Row, error: errors, params: { id: S.String } }),
    HttpApiEndpoint.get("liveGame", "/api/games/:id/live", { success: GameStatePayload, error: errors, params: { id: S.String } }),
    HttpApiEndpoint.get("lobbies", "/api/lobbies", { success: Lobbies, error: errors }),
    HttpApiEndpoint.get("adminStatus", "/admin/control/status", { success: S.Struct({ status: S.String, timestamp: S.String, platform: S.String }), error: errors }).middleware(AdminAuthorization),
    HttpApiEndpoint.get("sessions", "/admin/sessions", { success: S.Struct({ sessions: Rows, count: S.Number, timestamp: S.String }), error: errors }).middleware(AdminAuthorization),
    HttpApiEndpoint.post("kickSession", "/admin/sessions/kick", { success: S.Struct({ status: S.String, session_id: S.String, timestamp: S.String }), error: errors, payload: S.Struct({ session_id: S.String, reason: S.optional(S.String) }) }).middleware(AdminAuthorization),
    HttpApiEndpoint.get("adminLobbies", "/admin/lobbies", { success: Lobbies, error: errors }).middleware(AdminAuthorization),
    HttpApiEndpoint.put("putMap", "/admin/maps/:id", { success: Mutation, error: errors, params: { id: S.String }, payload: MapBody }).middleware(AdminAuthorization),
    HttpApiEndpoint.delete("deleteMap", "/admin/maps/:id", { success: Mutation, error: errors, params: { id: S.String } }).middleware(AdminAuthorization),
    HttpApiEndpoint.get("decks", "/admin/decks", { success: S.Struct({ decks: S.Array(S.Struct({ id: S.String, name: S.String, description: S.String, is_default: S.Boolean, plant_count: S.Number })), default_plants: S.Array(Plant) }), error: errors }).middleware(AdminAuthorization),
    HttpApiEndpoint.put("putDeck", "/admin/decks/:id", { success: S.Struct({ status: S.String, id: S.String, plants: S.Number }), error: errors, params: { id: S.String }, payload: DeckBody }).middleware(AdminAuthorization),
  ).annotateMerge(OpenApi.annotations({ title: "Power Grid API" }))
) { }
