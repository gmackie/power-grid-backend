/**
 * One Durable Object per game. Holds the engine state in DO storage, fans out to
 * hibernating WebSockets (direct `/game?game_id=` connections, admin spectators) and to
 * relay connections proxied through LobbyHubDO (bare `/game` clients), and records
 * players/actions/results to D1 through the Effect runtime.
 */
import { DurableObject } from "cloudflare:workers"
import { Effect, Schema } from "effect"
import { Assets } from "../db/assets.ts"
import { Records } from "../db/records.ts"
import { RESOURCE_TYPES } from "../protocol/game.ts"
import type { Env } from "../env.ts"
import * as Engine from "../domain/engine.ts"
import type { GameState } from "../domain/state.ts"
import { safeJsonParse } from "../protocol/common.ts"
import {
  BidPlantPayload,
  BuildCityPayload,
  BuyResourcesPayload,
  ConnectPayload,
  CreateGamePayload,
  ErrorCode,
  GameEnvelope,
  GameErrors,
  JoinGamePayload,
  MsgType,
  PowerCitiesPayload
} from "../protocol/game.ts"
import { makeRuntime, type AppRuntime } from "./runtime.ts"
import type { LobbyHubDO } from "./LobbyHubDO.ts"
import { gameMsg, isUpgrade, readAttachment, sendJson, upgradeResponse, writeAttachment } from "./ws.ts"

export interface InitInput {
  readonly game_id: string
  readonly name: string
  readonly map_id: string
  readonly players: ReadonlyArray<{ readonly id: string; readonly name: string; readonly color?: string }>
}

interface Attachment {
  kind: "player" | "spectator"
  player_id?: string
  session_id?: string
}

/** Something that can receive messages: a real socket or a relay through the hub. */
interface Sender {
  readonly key: string
  readonly playerId: string | undefined
  readonly sessionId: string | undefined
  send(msg: unknown): void
  bind(playerId: string): void
}

const STORAGE_STATE = "state"
const STORAGE_RELAYS = "relays"
const MAX_MESSAGE_BYTES = 4096 // Go read limit
const IDLE_MS = 24 * 60 * 60 * 1000

const decodeEnvelope = Schema.decodeUnknownOption(GameEnvelope)

export class GameDO extends DurableObject<Env> {
  private runtime: AppRuntime
  private state: GameState | null = null
  /** relayKey -> bound player id ("" until bound). */
  private relays: Record<string, string> = {}

  constructor(ctx: DurableObjectState, env: Env) {
    super(ctx, env)
    this.runtime = makeRuntime(env)
    ctx.blockConcurrencyWhile(async () => {
      this.state = (await ctx.storage.get<GameState>(STORAGE_STATE)) ?? null
      this.relays = (await ctx.storage.get<Record<string, string>>(STORAGE_RELAYS)) ?? {}
    })
  }

  // --- RPC (called by LobbyHubDO / Worker) --------------------------------------

  async init(input: InitInput): Promise<{ ok: true } | { ok: false; error: string }> {
    if (this.state) return { ok: true }
    try {
      const state = await this.runtime.runPromise(
        Effect.gen(function* () {
          const assets = yield* Assets
          const map = yield* assets.getMap(input.map_id)
          const deck = yield* assets.getDeck()
          const seed = crypto.getRandomValues(new Uint32Array(1))[0]!
          let s = Engine.createGame({ id: input.game_id, name: input.name, map, deck, seed, now: Date.now() })
          for (const p of input.players) s = yield* Engine.join(s, p)
          return s
        })
      )
      await this.persist(state)
      return { ok: true }
    } catch (e) {
      return { ok: false, error: e instanceof Error ? e.message : String(e) }
    }
  }

  async snapshot(): Promise<unknown> {
    return this.state ? Engine.toPayload(this.state) : null
  }

  async relayMessage(relayKey: string, raw: string): Promise<void> {
    if (!(relayKey in this.relays)) {
      this.relays[relayKey] = ""
      await this.ctx.storage.put(STORAGE_RELAYS, this.relays)
    }
    await this.handleRaw(this.relaySender(relayKey), raw)
  }

  async relayClosed(relayKey: string): Promise<void> {
    const playerId = this.relays[relayKey]
    delete this.relays[relayKey]
    await this.ctx.storage.put(STORAGE_RELAYS, this.relays)
    if (playerId) await this.onDisconnected(playerId)
  }

  // --- HTTP / WebSocket ---------------------------------------------------------------

  override async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url)
    if (isUpgrade(request)) {
      const pair = new WebSocketPair()
      const [client, server] = [pair[0], pair[1]]
      const spectator = url.pathname.startsWith("/ws/admin/game/")
      this.ctx.acceptWebSocket(server)
      writeAttachment<Attachment>(server, { kind: spectator ? "spectator" : "player" })
      await this.touch()
      if (spectator && this.state) sendJson(server, gameMsg(MsgType.GAME_STATE, Engine.toPayload(this.state)))
      return upgradeResponse(client)
    }
    if (request.method === "GET") {
      return Response.json(this.state ? Engine.toPayload(this.state) : { error: "game not found" }, {
        status: this.state ? 200 : 404
      })
    }
    return new Response("method not allowed", { status: 405 })
  }

  override async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer): Promise<void> {
    const att = readAttachment<Attachment>(ws, { kind: "player" })
    if (att.kind === "spectator") return
    if (typeof message !== "string" || message.length > MAX_MESSAGE_BYTES) {
      ws.close(1009, "Message too large or binary")
      return
    }
    await this.handleRaw(this.socketSender(ws), message)
  }

  override async webSocketClose(ws: WebSocket): Promise<void> {
    const att = readAttachment<Attachment>(ws, { kind: "player" })
    if (att.player_id) await this.onDisconnected(att.player_id)
  }

  override async webSocketError(ws: WebSocket): Promise<void> {
    await this.webSocketClose(ws)
  }

  override async alarm(): Promise<void> {
    if (!this.state) return
    if (this.state.status === "PLAYING") this.record(Records.pipe(Effect.flatMap((r) => r.gameAbandoned(this.state!.id))))
    await this.ctx.storage.deleteAll()
    this.state = null
    this.relays = {}
  }

  // --- senders ------------------------------------------------------------------------

  private socketSender(ws: WebSocket): Sender {
    const att = readAttachment<Attachment>(ws, { kind: "player" })
    return {
      key: "ws",
      playerId: att.player_id,
      sessionId: att.session_id,
      send: (msg) => sendJson(ws, msg),
      bind: (playerId) => writeAttachment<Attachment>(ws, { ...att, player_id: playerId })
    }
  }

  private relaySender(relayKey: string): Sender {
    const hub = this.env.LOBBY_HUB.get(this.env.LOBBY_HUB.idFromName("global")) as DurableObjectStub<LobbyHubDO>
    return {
      key: relayKey,
      playerId: this.relays[relayKey] || undefined,
      sessionId: undefined,
      send: (msg) => {
        this.ctx.waitUntil(hub.relayDeliver(relayKey, JSON.stringify(msg)).catch(() => undefined))
      },
      bind: (playerId) => {
        this.relays[relayKey] = playerId
        this.ctx.waitUntil(this.ctx.storage.put(STORAGE_RELAYS, this.relays))
      }
    }
  }

  private broadcast(msg: unknown): void {
    for (const ws of this.ctx.getWebSockets()) sendJson(ws, msg)
    if (Object.keys(this.relays).length > 0) {
      const hub = this.env.LOBBY_HUB.get(this.env.LOBBY_HUB.idFromName("global")) as DurableObjectStub<LobbyHubDO>
      const text = JSON.stringify(msg)
      for (const key of Object.keys(this.relays)) this.ctx.waitUntil(hub.relayDeliver(key, text).catch(() => undefined))
    }
  }

  private broadcastState(): void {
    if (this.state) this.broadcast(gameMsg(MsgType.GAME_STATE, Engine.toPayload(this.state)))
  }

  // --- message handling -------------------------------------------------------------------

  private async handleRaw(sender: Sender, raw: string): Promise<void> {
    const parsed = safeJsonParse(raw)
    const env = parsed === undefined ? undefined : decodeEnvelope(parsed)
    if (!env || env._tag === "None") {
      sender.send(gameMsg(MsgType.ERROR, { code: ErrorCode.INVALID_MESSAGE, message: "Could not parse message" }))
      return
    }
    await this.touch()
    const e = env.value
    const payload = e.payload ?? e.data
    try {
      await this.dispatch(sender, e.type, payload, e.player_id ?? e.session_id)
    } catch (err) {
      const message = err instanceof Engine.GameError ? err.message : err instanceof Error ? err.message : String(err)
      sender.send(gameMsg(MsgType.ERROR, { code: ErrorCode.MESSAGE_ERROR, message }, sender.sessionId))
    }
  }

  private async dispatch(sender: Sender, type: string, payload: unknown, envelopePlayerId: string | undefined): Promise<void> {
    // React client dialect (lowercase) is translated onto the Go protocol.
    const reactMapped = this.translateReact(type, payload)
    if (reactMapped) return this.dispatch(sender, reactMapped.type, reactMapped.payload, envelopePlayerId)

    switch (type) {
      case MsgType.PING:
        return sender.send(gameMsg(MsgType.PONG, undefined, sender.sessionId))
      case "ping":
        return sender.send({ type: "pong", timestamp: Math.floor(Date.now() / 1000) })
      case MsgType.DISCONNECT:
        return
      case MsgType.CREATE_GAME:
        return this.createGame(sender, payload)
      case MsgType.CONNECT:
        return this.connectPlayer(sender, payload)
      case MsgType.JOIN_GAME:
        return this.joinGame(sender, payload, envelopePlayerId)
      case MsgType.START_GAME:
        return this.startGame(sender)
      case MsgType.BID_PLANT: {
        const p = Schema.decodeUnknownOption(BidPlantPayload)(payload)
        if (p._tag === "None") throw new Engine.GameError({ message: GameErrors.InvalidBidPayload })
        return this.act(sender, type, payload, { _tag: "BidPlant", plantId: p.value.plant_id, bid: p.value.bid })
      }
      case MsgType.BUY_RESOURCES: {
        const p = Schema.decodeUnknownOption(BuyResourcesPayload)(payload)
        if (p._tag === "None") throw new Engine.GameError({ message: GameErrors.InvalidBuyPayload })
        return this.act(sender, type, payload, { _tag: "BuyResources", resources: normalizeResourceKeys(p.value.resources) })
      }
      case MsgType.BUILD_CITY: {
        const p = Schema.decodeUnknownOption(BuildCityPayload)(payload)
        if (p._tag === "None") throw new Engine.GameError({ message: GameErrors.InvalidBuildPayload })
        return this.act(sender, type, payload, { _tag: "BuildCity", cityId: p.value.city_id })
      }
      case MsgType.POWER_CITIES: {
        const p = Schema.decodeUnknownOption(PowerCitiesPayload)(payload)
        if (p._tag === "None") throw new Engine.GameError({ message: GameErrors.InvalidPowerPayload })
        return this.act(sender, type, payload, { _tag: "PowerCities", plantIds: p.value.power_plants })
      }
      case MsgType.END_TURN:
        return this.act(sender, type, payload, { _tag: "EndTurn" })
      default:
        throw new Engine.GameError({ message: GameErrors.UnknownMessage(type) })
    }
  }

  private translateReact(type: string, payload: unknown): { type: string; payload: unknown } | undefined {
    const data = (payload ?? {}) as Record<string, unknown>
    switch (type) {
      case "join_game":
        return {
          type: MsgType.JOIN_GAME,
          payload: { game_id: data["gameId"], player_name: data["playerName"], color: data["playerColor"] }
        }
      case "create_game":
        return { type: MsgType.CREATE_GAME, payload: { name: data["name"] ?? "Game", map: data["mapId"] ?? data["map"] ?? "usa" } }
      case "start_game":
        return { type: MsgType.START_GAME, payload: undefined }
      case "player_action": {
        const action = String(data["action"] ?? "")
        const params = (data["params"] ?? {}) as Record<string, unknown>
        switch (action) {
          case "bid":
            return { type: MsgType.BID_PLANT, payload: { plant_id: params["plant_id"] ?? this.state?.auction?.plantNumber ?? 0, bid: params["amount"] ?? 0 } }
          case "pass":
            return this.state?.auction
              ? { type: MsgType.BID_PLANT, payload: { plant_id: this.state.auction.plantNumber, bid: 0 } }
              : { type: MsgType.END_TURN, payload: undefined }
          case "buy_resources":
            return { type: MsgType.BUY_RESOURCES, payload: { resources: params["resources"] ?? {} } }
          case "build_city":
            return { type: MsgType.BUILD_CITY, payload: { city_id: params["cityId"] ?? params["city_id"] } }
          case "power_cities":
            return { type: MsgType.POWER_CITIES, payload: { power_plants: params["plant_ids"] ?? params["power_plants"] ?? [] } }
          case "end_turn":
            return { type: MsgType.END_TURN, payload: undefined }
        }
        return undefined
      }
      default:
        return undefined
    }
  }

  private async createGame(sender: Sender, payload: unknown): Promise<void> {
    if (this.state) throw new Engine.GameError({ message: GameErrors.AlreadyStarted })
    const p = Schema.decodeUnknownOption(CreateGamePayload)(payload)
    if (p._tag === "None") throw new Engine.GameError({ message: "invalid payload for create game" })
    const gameId = (await this.ctx.storage.get<string>("game_id")) ?? crypto.randomUUID()
    const result = await this.init({ game_id: gameId, name: p.value.name, map_id: p.value.map ?? "usa", players: [] })
    if (!result.ok) throw new Engine.GameError({ message: result.error })
    sender.send(gameMsg(MsgType.GAME_STATE, Engine.toPayload(this.state!), sender.sessionId))
  }

  private async connectPlayer(sender: Sender, payload: unknown): Promise<void> {
    const p = Schema.decodeUnknownOption(ConnectPayload)(payload)
    const data = p._tag === "Some" ? p.value : {}
    if (!this.state) throw new Engine.GameError({ message: GameErrors.GameNotFound(data.game_id ?? "") })
    let state = this.state
    const playerId = data.player_id ?? sender.playerId
    if (!playerId) throw new Engine.GameError({ message: GameErrors.NotInGame })
    if (!state.players[playerId]) {
      if (!data.player_name) throw new Engine.GameError({ message: GameErrors.PlayerNotFound })
      state = Effect.runSync(Engine.join(state, { id: playerId, name: data.player_name }))
    }
    state = Engine.setConnected(state, playerId, true)
    sender.bind(playerId)
    await this.persist(state)
    sender.send(gameMsg(MsgType.GAME_STATE, Engine.toPayload(state), sender.sessionId))
    this.broadcast(gameMsg(MsgType.PLAYER_JOINED, { player_id: playerId, player_name: state.players[playerId]!.name }))
  }

  private async joinGame(sender: Sender, payload: unknown, envelopePlayerId: string | undefined): Promise<void> {
    const p = Schema.decodeUnknownOption(JoinGamePayload)(payload)
    if (p._tag === "None") throw new Engine.GameError({ message: "invalid payload for join game" })
    if (!this.state) throw new Engine.GameError({ message: GameErrors.GameNotFound(p.value.game_id ?? "") })
    const playerId = envelopePlayerId ?? sender.playerId ?? crypto.randomUUID()
    let state = this.state
    if (!state.players[playerId]) {
      state = Effect.runSync(Engine.join(state, { id: playerId, name: p.value.player_name, color: p.value.color || undefined }))
    }
    state = Engine.setConnected(state, playerId, true)
    sender.bind(playerId)
    await this.persist(state)
    this.broadcastState()
  }

  private async startGame(sender: Sender): Promise<void> {
    if (!this.state) throw new Engine.GameError({ message: GameErrors.GameNotFound("") })
    if (!sender.playerId) throw new Engine.GameError({ message: GameErrors.NotInGame })
    const { state, events } = Effect.runSync(Engine.start(this.state, Date.now()))
    await this.persist(state)
    this.record(Records.pipe(Effect.flatMap((r) => r.gameStarted(state))))
    this.emit(events)
    this.broadcastState()
  }

  private async act(sender: Sender, type: string, payload: unknown, action: Engine.Action): Promise<void> {
    if (!this.state) throw new Engine.GameError({ message: GameErrors.GameNotFound("") })
    const playerId = sender.playerId
    if (!playerId) throw new Engine.GameError({ message: GameErrors.NotInGame })
    const before = this.state
    try {
      const { state, events } = Effect.runSync(Engine.applyAction(before, playerId, action, Date.now()))
      await this.persist(state)
      this.record(Records.pipe(Effect.flatMap((r) => r.logAction(before, playerId, type, payload, "success"))))
      this.emit(events)
      this.broadcastState()
      if (state.status === "FINISHED") {
        this.record(Records.pipe(Effect.flatMap((r) => r.gameCompleted(state))))
      }
    } catch (err) {
      const message = err instanceof Engine.GameError ? err.message : String(err)
      this.record(Records.pipe(Effect.flatMap((r) => r.logAction(before, playerId, type, payload, "failed", message))))
      throw err
    }
  }

  private emit(events: ReadonlyArray<Engine.Event>): void {
    for (const ev of events) {
      switch (ev._tag) {
        case "PhaseChange":
          this.broadcast(gameMsg(MsgType.PHASE_CHANGE, { phase: ev.phase, round: ev.round }))
          break
        case "TurnChange":
          this.broadcast(gameMsg(MsgType.TURN_CHANGE, { current_player_id: ev.playerId, turn: ev.turn }))
          break
        case "StepChange":
          this.broadcast(gameMsg("STEP_CHANGE", { step: ev.step }))
          break
        case "PlantBought":
          this.broadcast(gameMsg("PLANT_BOUGHT", { player_id: ev.playerId, plant_id: ev.plant, price: ev.price }))
          break
        case "GameEnd":
          this.broadcast(gameMsg("GAME_END", { winner_id: ev.winnerId }))
          break
      }
    }
  }

  private async onDisconnected(playerId: string): Promise<void> {
    if (!this.state?.players[playerId]) return
    const stillConnected = this.ctx
      .getWebSockets()
      .some((ws) => readAttachment<Attachment>(ws, { kind: "player" }).player_id === playerId)
    if (stillConnected || Object.values(this.relays).includes(playerId)) return
    await this.persist(Engine.setConnected(this.state, playerId, false))
    this.broadcastState()
  }

  // --- persistence --------------------------------------------------------------------

  private async persist(state: GameState): Promise<void> {
    this.state = state
    await this.ctx.storage.put(STORAGE_STATE, state)
  }

  private async touch(): Promise<void> {
    await this.ctx.storage.setAlarm(Date.now() + IDLE_MS)
  }

  private record<E>(effect: Effect.Effect<void, E, Records>): void {
    this.ctx.waitUntil(
      this.runtime.runPromise(effect).catch((e: unknown) => {
        console.error("[GameDO] record failed", e instanceof Error ? e.message : e)
      })
    )
  }
}

/** React sends lowercase resource keys; the engine uses Title case. */
const normalizeResourceKeys = (r: Record<string, number>): Record<string, number> => {
  const out: Record<string, number> = {}
  for (const [k, v] of Object.entries(r)) {
    const match = RESOURCE_TYPES.find((t) => t.toLowerCase() === k.toLowerCase())
    out[match ?? k] = v
  }
  return out
}
