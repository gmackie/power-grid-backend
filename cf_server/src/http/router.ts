/**
 * HTTP API built on Effect's HttpRouter and served through `HttpRouter.toWebHandler`.
 * WebSocket upgrades never reach this router; src/index.ts routes them to Durable Objects.
 */
import { Context, Effect, Layer, Schema } from "effect"
import { HttpRouter, HttpServerRequest, HttpServerResponse } from "effect/http"
import type { HttpServerResponse as Resp } from "effect/http/HttpServerResponse"
import { Analytics, AnalyticsLive, NotFound } from "../db/analytics.ts"
import { AssetError, Assets, AssetsLive } from "../db/assets.ts"
import { D1Live } from "../db/layer.ts"
import { MapData, decodeMapData } from "../domain/map.ts"
import { DEFAULT_DECK } from "../domain/deck-default.ts"
import type { Env } from "../env.ts"
import type { GameDO } from "../do/GameDO.ts"
import type { LobbyHubDO } from "../do/LobbyHubDO.ts"

export class WorkerEnv extends Context.Service<WorkerEnv, Env>()("WorkerEnv") {}

const json = (body: unknown, status = 200) => HttpServerResponse.json(body, { status })
const now = () => new Date().toISOString()

const errorBody = (message: string, status: number) => ({ error: message, status, timestamp: now() })

/** Map domain failures to Go-style error JSON; anything else is a 500. */
const respond = <A, E, R>(self: Effect.Effect<A, E, R>, status = 200): Effect.Effect<Resp, never, R> =>
  self.pipe(
    Effect.flatMap((a) => json(a, status)),
    Effect.catch((e) => {
      if (e instanceof NotFound) return json(errorBody(e.message, 404), 404)
      if (e instanceof AssetError) return json(errorBody(e.message, 404), 404)
      if (e instanceof Schema.SchemaError) return json(errorBody(`invalid request: ${e.message}`, 400), 400)
      const message = e instanceof Error ? e.message : String(e)
      return json(errorBody(message, 500), 500)
    }),
    Effect.orDie
  )

const query = Effect.map(HttpServerRequest.HttpServerRequest, (req) => new URL(req.url, "http://localhost").searchParams)

const intParam = (params: URLSearchParams, key: string, fallback: number, min: number, max: number): number => {
  const raw = params.get(key)
  const n = raw === null ? NaN : Number(raw)
  return Number.isInteger(n) && n >= min && n <= max ? n : fallback
}

const param = (name: string) => Effect.map(HttpRouter.params, (p) => decodeURIComponent(p[name] ?? ""))

const hub = Effect.map(WorkerEnv, (env) => env.LOBBY_HUB.get(env.LOBBY_HUB.idFromName("global")) as DurableObjectStub<LobbyHubDO>)
const gameStub = (id: string) => Effect.map(WorkerEnv, (env) => env.GAME.get(env.GAME.idFromName(id)) as DurableObjectStub<GameDO>)

const DeckBody = Schema.Struct({
  name: Schema.String,
  description: Schema.optional(Schema.String),
  is_default: Schema.optional(Schema.Boolean),
  plants: Schema.Array(
    Schema.Struct({
      number: Schema.Number,
      capacity: Schema.Number,
      resourceType: Schema.Literals(["Coal", "Oil", "Garbage", "Uranium", "Hybrid", "Wind"]),
      resourceCost: Schema.Number
    })
  )
})

const body = <S extends Schema.Top & Schema.ConstraintDecoder<unknown, never>>(schema: S) =>
  Effect.flatMap(HttpServerRequest.HttpServerRequest, (req) => Effect.flatMap(req.json, (v) => Schema.decodeUnknownEffect(schema)(v)))

const Routes = Layer.mergeAll(
  // --- liveness ---------------------------------------------------------------------
  HttpRouter.add(
    "GET",
    "/",
    Effect.flatMap(WorkerEnv, (env) => json({ name: env.SERVER_NAME, version: env.SERVER_VERSION, status: "running" }))
  ),
  HttpRouter.add("GET", "/health", json({ status: "healthy" })),
  HttpRouter.add("GET", "/ready", json({ status: "ready" })),
  HttpRouter.add("GET", "/api/health", json({ status: "healthy", service: "analytics-database", timestamp: now(), version: "1.0.0" })),

  // --- maps (public) ------------------------------------------------------------------
  HttpRouter.add("GET", "/maps", respond(Effect.flatMap(Assets, (a) => Effect.map(a.listMaps, (maps) => ({ maps }))))),
  HttpRouter.add("GET", "/maps/:id", respond(Effect.flatMap(param("id"), (id) => Effect.flatMap(Assets, (a) => a.getMap(id))))),

  // --- players ---------------------------------------------------------------------------
  HttpRouter.add(
    "GET",
    "/api/players",
    respond(
      Effect.gen(function* () {
        const q = yield* query
        const a = yield* Analytics
        const players = yield* a.listPlayers(intParam(q, "limit", 50, 1, 200), intParam(q, "offset", 0, 0, 1_000_000))
        return { players, count: players.length }
      })
    )
  ),
  HttpRouter.add("GET", "/api/players/:name", respond(Effect.flatMap(param("name"), (n) => Effect.flatMap(Analytics, (a) => a.playerStats(n))))),
  HttpRouter.add("GET", "/api/players/:name/stats", respond(Effect.flatMap(param("name"), (n) => Effect.flatMap(Analytics, (a) => a.playerStats(n))))),
  HttpRouter.add(
    "GET",
    "/api/players/:name/achievements",
    respond(
      Effect.gen(function* () {
        const name = yield* param("name")
        const achievements = yield* Effect.flatMap(Analytics, (a) => a.playerAchievements(name))
        return { player_name: name, achievements, total_count: achievements.length }
      })
    )
  ),
  HttpRouter.add(
    "GET",
    "/api/players/:name/history",
    respond(
      Effect.gen(function* () {
        const name = yield* param("name")
        const q = yield* query
        const games = yield* Effect.flatMap(Analytics, (a) => a.playerHistory(name, intParam(q, "limit", 20, 1, 100)))
        return { player_name: name, game_count: games.length, games }
      })
    )
  ),

  // --- achievements / leaderboard ------------------------------------------------------------
  HttpRouter.add("GET", "/api/achievements", respond(Effect.flatMap(Analytics, (a) => a.achievements))),
  HttpRouter.add(
    "GET",
    "/api/leaderboard",
    respond(
      Effect.gen(function* () {
        const q = yield* query
        const limit = intParam(q, "limit", 50, 1, 100)
        const leaderboard = yield* Effect.flatMap(Analytics, (a) => a.leaderboard(limit))
        return { leaderboard, limit, count: leaderboard.length, updated_at: now() }
      })
    )
  ),

  // --- analytics ---------------------------------------------------------------------------
  HttpRouter.add(
    "GET",
    "/api/analytics/games",
    respond(
      Effect.gen(function* () {
        const q = yield* query
        const days = intParam(q, "days", 30, 1, 365)
        const analytics = yield* Effect.flatMap(Analytics, (a) => a.gameAnalytics(days))
        return { analytics, period_days: days, updated_at: now() }
      })
    )
  ),
  HttpRouter.add(
    "GET",
    "/api/analytics/achievements",
    respond(Effect.map(Effect.flatMap(Analytics, (a) => a.achievementStats), (stats) => ({ stats, updated_at: now() })))
  ),
  HttpRouter.add(
    "GET",
    "/api/analytics/activity",
    respond(
      Effect.gen(function* () {
        const q = yield* query
        const days = intParam(q, "days", 30, 1, 365)
        const activity = yield* Effect.flatMap(Analytics, (a) => a.activity(days))
        return { ...activity, generated_at: now() }
      })
    )
  ),

  // --- games ------------------------------------------------------------------------------
  HttpRouter.add(
    "GET",
    "/api/games",
    respond(
      Effect.gen(function* () {
        const q = yield* query
        const games = yield* Effect.flatMap(Analytics, (a) => a.recentGames(intParam(q, "limit", 20, 1, 100), q.get("status") ?? undefined))
        return { games, count: games.length }
      })
    )
  ),
  HttpRouter.add("GET", "/api/games/:id", respond(Effect.flatMap(param("id"), (id) => Effect.flatMap(Analytics, (a) => a.game(id))))),
  HttpRouter.add(
    "GET",
    "/api/games/:id/live",
    respond(
      Effect.gen(function* () {
        const id = yield* param("id")
        const stub = yield* gameStub(id)
        const snapshot = yield* Effect.promise(() => stub.snapshot())
        if (snapshot === null) return yield* Effect.fail(new NotFound({ message: "Game not found" }))
        return snapshot
      })
    )
  ),
  HttpRouter.add("GET", "/api/lobbies", respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listLobbies()), (lobbies) => ({ lobbies }))))),

  // --- admin (bearer token enforced in src/index.ts) ------------------------------------------
  HttpRouter.add("GET", "/admin/control/status", json({ status: "running", timestamp: now(), platform: "cloudflare-workers" })),
  HttpRouter.add(
    "GET",
    "/admin/sessions",
    respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listSessions()), (sessions) => ({ sessions, count: sessions.length, timestamp: now() }))))
  ),
  HttpRouter.add(
    "POST",
    "/admin/sessions/kick",
    respond(
      Effect.gen(function* () {
        const b = yield* body(Schema.Struct({ session_id: Schema.String, reason: Schema.optional(Schema.String) }))
        const h = yield* hub
        const kicked = yield* Effect.promise(() => h.kickSession(b.session_id))
        return { status: kicked ? "session_kicked" : "session_not_found", session_id: b.session_id, timestamp: now() }
      })
    )
  ),
  HttpRouter.add("GET", "/admin/lobbies", respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listLobbies()), (lobbies) => ({ lobbies }))))),
  HttpRouter.add(
    "PUT",
    "/admin/maps/:id",
    respond(
      Effect.gen(function* () {
        const id = yield* param("id")
        const raw = yield* Effect.flatMap(HttpServerRequest.HttpServerRequest, (r) => r.json)
        const map = yield* Schema.decodeUnknownEffect(MapData)({ ...(raw as object), id })
        yield* Effect.flatMap(Assets, (a) => a.upsertMap(decodeMapData(map)))
        return { status: "ok", id }
      })
    )
  ),
  HttpRouter.add(
    "DELETE",
    "/admin/maps/:id",
    respond(
      Effect.gen(function* () {
        const id = yield* param("id")
        const ok = yield* Effect.flatMap(Assets, (a) => a.setMapActive(id, false))
        if (!ok) return yield* Effect.fail(new NotFound({ message: "Map not found" }))
        return { status: "deactivated", id }
      })
    )
  ),
  HttpRouter.add("GET", "/admin/decks", respond(Effect.flatMap(Assets, (a) => Effect.map(a.listDecks, (decks) => ({ decks, default_plants: DEFAULT_DECK }))))),
  HttpRouter.add(
    "PUT",
    "/admin/decks/:id",
    respond(
      Effect.gen(function* () {
        const id = yield* param("id")
        const b = yield* body(DeckBody)
        yield* Effect.flatMap(Assets, (a) => a.upsertDeck({ id, ...b }))
        return { status: "ok", id, plants: b.plants.length }
      })
    )
  ),
  HttpRouter.add(
    "*",
    "/api/admin/simulated/*",
    json({ error: "Simulated games (local AI processes) are not available on Cloudflare Workers", status: 501 }, 501)
  )
)

const Services = (env: Env) =>
  Layer.mergeAll(AssetsLive, AnalyticsLive).pipe(Layer.provide(D1Live(env.DB)), Layer.merge(Layer.succeed(WorkerEnv, env)))

/** Build the fetch handler once per Worker env. */
export const makeHttpHandler = (env: Env) => HttpRouter.toWebHandler(Layer.mergeAll(Routes, Services(env)), { disableLogger: true })
