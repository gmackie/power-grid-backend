/**
 * HTTP API implemented from the shared HttpApi contract and served through HttpRouter.
 * WebSocket upgrades never reach this router; src/index.ts routes them to Durable Objects.
 */
import { Context, Effect, Layer, Redacted, Schema } from "effect"
import { HttpRouter, HttpServer, HttpServerResponse } from "effect/http"
import { HttpApiBuilder, HttpApiSwagger } from "effect/http-api"
import { PowerGridApi, ApiNotFound, ApiFailure, AdminAuthorization, Unauthorized } from "./api.ts"
import { Analytics, AnalyticsLive, NotFound } from "../db/analytics.ts"
import { AssetError, Assets, AssetsLive } from "../db/assets.ts"
import { D1Live } from "../db/layer.ts"
import { GameStatePayload } from "../protocol/game.ts"
import { MapData, decodeMapData } from "../domain/map.ts"
import { DEFAULT_DECK } from "../domain/deck-default.ts"
import type { Env } from "../env.ts"
import type { GameDO } from "../do/GameDO.ts"
import type { LobbyHubDO } from "../do/LobbyHubDO.ts"

export class WorkerEnv extends Context.Service<WorkerEnv, Env>()("WorkerEnv") { }

const now=() => new Date().toISOString()

const errorBody=(message: string, status: number) => ({ error: message, status, timestamp: now() })

/** Translate expected domain failures into the errors declared by the contract. */
const respond=<A, E, R>(self: Effect.Effect<A, E, R>): Effect.Effect<A, ApiNotFound|ApiFailure, R> => self.pipe(
  Effect.catch((e): Effect.Effect<never, ApiNotFound|ApiFailure> => e instanceof NotFound||e instanceof AssetError
    ? Effect.fail(new ApiNotFound(errorBody(e.message, 404)))
    :Effect.fail(new ApiFailure(errorBody("Internal server error", 500))))
)
const intParam=(params: URLSearchParams, key: string, fallback: number, min: number, max: number) => {
  const raw=params.get(key); const n=raw===null? NaN:Number(raw)
  return Number.isInteger(n)&&n>=min&&n<=max? n:fallback
}
const hub=Effect.map(WorkerEnv, (env) => env.LOBBY_HUB.get(env.LOBBY_HUB.idFromName("global")) as DurableObjectStub<LobbyHubDO>)
const gameStub=(id: string) => Effect.map(WorkerEnv, (env) => env.GAME.get(env.GAME.idFromName(id)) as DurableObjectStub<GameDO>)

const Handlers=HttpApiBuilder.group(PowerGridApi, "server", handlers => handlers
  .handle("info", ({ }) => Effect.flatMap(WorkerEnv, (env) => Effect.succeed({ name: env.SERVER_NAME, version: env.SERVER_VERSION, status: "running" })))
  .handle("health", ({ }) => Effect.succeed({ status: "healthy" }))
  .handle("ready", ({ }) => Effect.succeed({ status: "ready" }))
  .handle("analyticsHealth", ({ }) => Effect.succeed({ status: "healthy", service: "analytics-database", timestamp: now(), version: "1.0.0" }))
  .handle("maps", ({ }) => respond(Effect.flatMap(Assets, (a) => Effect.map(a.listMaps, (maps) => ({ maps })))))
  .handle("map", ({ params }) => respond(Effect.flatMap(Effect.succeed(params.id), (id) => Effect.flatMap(Assets, (a) => a.getMap(id)))))
  .handle("players", ({ query }) => respond(
    Effect.gen(function*() {
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const a=yield* Analytics
      const players=yield* a.listPlayers(intParam(q, "limit", 50, 1, 200), intParam(q, "offset", 0, 0, 1_000_000))
      return { players, count: players.length }
    })
  ))
  .handle("player", ({ params }) => respond(Effect.flatMap(Effect.succeed(params.name), (n) => Effect.flatMap(Analytics, (a) => a.playerStats(n)))))
  .handle("playerStats", ({ params }) => respond(Effect.flatMap(Effect.succeed(params.name), (n) => Effect.flatMap(Analytics, (a) => a.playerStats(n)))))
  .handle("playerAchievements", ({ params }) => respond(
    Effect.gen(function*() {
      const name=yield* Effect.succeed(params.name)
      const achievements=yield* Effect.flatMap(Analytics, (a) => a.playerAchievements(name))
      return { player_name: name, achievements, total_count: achievements.length }
    })
  ))
  .handle("playerHistory", ({ params, query }) => respond(
    Effect.gen(function*() {
      const name=yield* Effect.succeed(params.name)
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const games=yield* Effect.flatMap(Analytics, (a) => a.playerHistory(name, intParam(q, "limit", 20, 1, 100)))
      return { player_name: name, game_count: games.length, games }
    })
  ))
  .handle("achievements", ({ }) => respond(Effect.flatMap(Analytics, (a) => a.achievements)))
  .handle("leaderboard", ({ query }) => respond(
    Effect.gen(function*() {
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const limit=intParam(q, "limit", 50, 1, 100)
      const leaderboard=yield* Effect.flatMap(Analytics, (a) => a.leaderboard(limit))
      return { leaderboard, limit, count: leaderboard.length, updated_at: now() }
    })
  ))
  .handle("gameAnalytics", ({ query }) => respond(
    Effect.gen(function*() {
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const days=intParam(q, "days", 30, 1, 365)
      const analytics=yield* Effect.flatMap(Analytics, (a) => a.gameAnalytics(days))
      return { analytics, period_days: days, updated_at: now() }
    })
  ))
  .handle("achievementStats", ({ }) => respond(Effect.map(Effect.flatMap(Analytics, (a) => a.achievementStats), (stats) => ({ stats, updated_at: now() }))))
  .handle("activity", ({ query }) => respond(
    Effect.gen(function*() {
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const days=intParam(q, "days", 30, 1, 365)
      const activity=yield* Effect.flatMap(Analytics, (a) => a.activity(days))
      return { ...activity, generated_at: now() }
    })
  ))
  .handle("games", ({ query }) => respond(
    Effect.gen(function*() {
      const q=new URLSearchParams(Object.entries(query).filter((entry): entry is [string, string] => entry[1]!==undefined))
      const games=yield* Effect.flatMap(Analytics, (a) => a.recentGames(intParam(q, "limit", 20, 1, 100), q.get("status")??undefined))
      return { games, count: games.length }
    })
  ))
  .handle("game", ({ params }) => respond(Effect.flatMap(Effect.succeed(params.id), (id) => Effect.flatMap(Analytics, (a) => a.game(id)))))
  .handle("liveGame", ({ params }) => respond(
    Effect.gen(function*() {
      const id=yield* Effect.succeed(params.id)
      const stub=yield* gameStub(id)
      const snapshot=yield* Effect.promise(() => stub.snapshot())
      if(snapshot===null) return yield* Effect.fail(new NotFound({ message: "Game not found" }))
      return yield* Schema.decodeUnknownEffect(GameStatePayload)(snapshot)
    })
  ))
  .handle("lobbies", ({ }) => respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listLobbies()), (lobbies) => ({ lobbies })))))
  .handle("adminStatus", ({ }) => Effect.succeed({ status: "running", timestamp: now(), platform: "cloudflare-workers" }))
  .handle("sessions", ({ }) => respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listSessions()), (sessions) => ({ sessions, count: sessions.length, timestamp: now() })))))
  .handle("kickSession", ({ payload }) => respond(
    Effect.gen(function*() {
      const b=payload
      const h=yield* hub
      const kicked=yield* Effect.promise(() => h.kickSession(b.session_id))
      return { status: kicked? "session_kicked":"session_not_found", session_id: b.session_id, timestamp: now() }
    })
  ))
  .handle("adminLobbies", ({ }) => respond(Effect.flatMap(hub, (h) => Effect.map(Effect.promise(() => h.listLobbies()), (lobbies) => ({ lobbies })))))
  .handle("putMap", ({ params, payload }) => respond(
    Effect.gen(function*() {
      const id=yield* Effect.succeed(params.id)
      const raw=payload
      const map=yield* Schema.decodeUnknownEffect(MapData)({ ...(raw as object), id })
      yield* Effect.flatMap(Assets, (a) => a.upsertMap(decodeMapData(map)))
      return { status: "ok", id }
    })
  ))
  .handle("deleteMap", ({ params }) => respond(
    Effect.gen(function*() {
      const id=yield* Effect.succeed(params.id)
      const ok=yield* Effect.flatMap(Assets, (a) => a.setMapActive(id, false))
      if(!ok) return yield* Effect.fail(new NotFound({ message: "Map not found" }))
      return { status: "deactivated", id }
    })
  ))
  .handle("decks", ({ }) => respond(Effect.flatMap(Assets, (a) => Effect.map(a.listDecks, (decks) => ({ decks, default_plants: DEFAULT_DECK })))))
  .handle("putDeck", ({ params, payload }) => respond(
    Effect.gen(function*() {
      const id=yield* Effect.succeed(params.id)
      const b=payload
      yield* Effect.flatMap(Assets, (a) => a.upsertDeck({ id, ...b }))
      return { status: "ok", id, plants: b.plants.length }
    })
  ))
)
const Services=(env: Env) =>
  Layer.mergeAll(AssetsLive, AnalyticsLive).pipe(Layer.provide(D1Live(env.DB)), Layer.merge(Layer.succeed(WorkerEnv, env)))

const AuthorizationLive=(env: Env) => Layer.succeed(AdminAuthorization, AdminAuthorization.of({
  bearer: (effect, { credential }) => env.ADMIN_TOKEN&&Redacted.value(credential)===env.ADMIN_TOKEN
    ? effect:Effect.fail(new Unauthorized({ error: "unauthorized", status: 401 }))
}))

export const makeHttpHandler=(env: Env) => HttpRouter.toWebHandler(
  Layer.mergeAll(
    HttpApiBuilder.layer(PowerGridApi, { openapiPath: "/openapi.json" }).pipe(Layer.provide(Handlers), Layer.provide(AuthorizationLive(env))),
    HttpApiSwagger.layer(PowerGridApi, { path: "/docs" }),
    HttpRouter.add("*", "/api/admin/simulated/*", HttpServerResponse.json({ error: "Simulated games are not available on Cloudflare Workers", status: 501 }, { status: 501 }))
  ).pipe(Layer.provide(HttpServer.layerServices), Layer.provideMerge(Services(env))), { disableLogger: true })
