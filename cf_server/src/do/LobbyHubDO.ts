/**
 * Single Durable Object ("global") implementing the `/ws` lobby protocol from
 * handlers/lobby_handler.go, plus a relay for bare `/game` sockets whose target game is
 * only known from their first message. Sessions and lobbies live in DO storage so they
 * survive hibernation.
 */
import { DurableObject } from "cloudflare:workers"
import { Effect, Schema } from "effect"
import { Assets } from "../db/assets.ts"
import type { Env } from "../env.ts"
import { safeJsonParse } from "../protocol/common.ts"
import {
  ChatData,
  ConnectData,
  ConnectedMessages,
  CreateLobbyData,
  JoinLobbyData,
  LobbyClientType as C,
  LobbyEnvelope,
  LobbyErrors as Err,
  LobbyServerType as S,
  SetReadyData,
  SystemMessages,
  type LobbyJSON,
  type LobbyMessage,
  type LobbyStatus,
  type LobbySummary
} from "../protocol/lobby.ts"
import type { GameDO } from "./GameDO.ts"
import { makeRuntime, type AppRuntime } from "./runtime.ts"
import { gameMsg, isUpgrade, lobbyMsg, readAttachment, sendJson, upgradeResponse, writeAttachment } from "./ws.ts"

interface Session {
  player_id: string
  player_name: string
  created_at: number
  last_activity: number
  lobby_id?: string
}

interface MutablePlayer {
  id: string
  name: string
  is_host: boolean
  is_ready: boolean
  joined_at: string
}

interface Lobby {
  id: string
  name: string
  status: LobbyStatus
  players: Record<string, MutablePlayer>
  messages: Array<LobbyMessage>
  max_players: number
  map_id: string
  password: string
  game_id?: string
  created_at: string
  updated_at: string
}

type Attachment =
  | { kind: "lobby"; temp: string; session_id?: string }
  | { kind: "relay"; key: string; game_id?: string }

const CLEANUP_INTERVAL_MS = 5 * 60 * 1000
const MAX_MESSAGES = 200
const MAX_MESSAGE_BYTES = 4096 // Go read limit

const decodeEnvelope = Schema.decodeUnknownOption(LobbyEnvelope)

export class LobbyHubDO extends DurableObject<Env> {
  private runtime: AppRuntime
  private sessions: Record<string, Session> = {}
  private lobbies: Record<string, Lobby> = {}

  constructor(ctx: DurableObjectState, env: Env) {
    super(ctx, env)
    this.runtime = makeRuntime(env)
    ctx.blockConcurrencyWhile(async () => {
      this.sessions = (await ctx.storage.get<Record<string, Session>>("sessions")) ?? {}
      this.lobbies = (await ctx.storage.get<Record<string, Lobby>>("lobbies")) ?? {}
      if ((await ctx.storage.getAlarm()) === null) await ctx.storage.setAlarm(Date.now() + CLEANUP_INTERVAL_MS)
    })
  }

  // --- RPC ------------------------------------------------------------------------

  async relayDeliver(relayKey: string, text: string): Promise<void> {
    for (const ws of this.ctx.getWebSockets("relay")) {
      const att = readAttachment<Attachment>(ws, { kind: "relay", key: "" })
      if (att.kind === "relay" && att.key === relayKey) {
        try {
          ws.send(text)
        } catch {
          /* closed */
        }
      }
    }
  }

  async listSessions(): Promise<Array<{ session_id: string; player_id: string; player_name: string; lobby_id?: string; last_activity: number; connected: boolean }>> {
    const connected = new Set<string>()
    for (const ws of this.ctx.getWebSockets("lobby")) {
      const att = readAttachment<Attachment>(ws, { kind: "lobby", temp: "" })
      if (att.kind === "lobby" && att.session_id) connected.add(att.session_id)
    }
    return Object.entries(this.sessions).map(([session_id, s]) => ({
      session_id,
      player_id: s.player_id,
      player_name: s.player_name,
      ...(s.lobby_id ? { lobby_id: s.lobby_id } : {}),
      last_activity: s.last_activity,
      connected: connected.has(session_id)
    }))
  }

  async kickSession(sessionId: string): Promise<boolean> {
    const s = this.sessions[sessionId]
    if (!s) return false
    for (const ws of this.ctx.getWebSockets("lobby")) {
      const att = readAttachment<Attachment>(ws, { kind: "lobby", temp: "" })
      if (att.kind === "lobby" && att.session_id === sessionId) ws.close(4000, "kicked")
    }
    await this.expireSession(sessionId)
    return true
  }

  async listLobbies(): Promise<Array<LobbySummary>> {
    return this.summaries()
  }

  // --- fetch / sockets ------------------------------------------------------------

  override async fetch(request: Request): Promise<Response> {
    const url = new URL(request.url)
    if (!isUpgrade(request)) return new Response("expected websocket", { status: 426 })
    const pair = new WebSocketPair()
    const [client, server] = [pair[0], pair[1]]
    if (url.pathname === "/game") {
      this.ctx.acceptWebSocket(server, ["relay"])
      writeAttachment<Attachment>(server, { kind: "relay", key: crypto.randomUUID() })
    } else {
      this.ctx.acceptWebSocket(server, ["lobby"])
      writeAttachment<Attachment>(server, { kind: "lobby", temp: crypto.randomUUID() })
    }
    return upgradeResponse(client)
  }

  override async webSocketMessage(ws: WebSocket, message: string | ArrayBuffer): Promise<void> {
    if (typeof message !== "string" || message.length > MAX_MESSAGE_BYTES) {
      ws.close(1009, "Message too large or binary")
      return
    }
    const raw = message
    const att = readAttachment<Attachment>(ws, { kind: "lobby", temp: "" })
    if (att.kind === "relay") return this.relayIncoming(ws, att, raw)
    await this.lobbyIncoming(ws, att, raw)
  }

  override async webSocketClose(ws: WebSocket): Promise<void> {
    const att = readAttachment<Attachment>(ws, { kind: "lobby", temp: "" })
    if (att.kind === "relay" && att.game_id) {
      const stub = this.gameStub(att.game_id)
      await stub.relayClosed(att.key).catch(() => undefined)
    }
    // Lobby sessions survive disconnects (Go behaviour); cleanup happens by idle timeout.
  }

  override async webSocketError(ws: WebSocket): Promise<void> {
    await this.webSocketClose(ws)
  }

  override async alarm(): Promise<void> {
    const idleMs = Number(this.env.SESSION_IDLE_MINUTES || "30") * 60 * 1000
    const cutoff = Date.now() - idleMs
    for (const [id, s] of Object.entries(this.sessions)) {
      if (s.last_activity < cutoff) await this.expireSession(id)
    }
    await this.ctx.storage.setAlarm(Date.now() + CLEANUP_INTERVAL_MS)
  }

  // --- relay (bare /game sockets) ------------------------------------------------

  private async relayIncoming(ws: WebSocket, att: Extract<Attachment, { kind: "relay" }>, raw: string): Promise<void> {
    let gameId = att.game_id
    if (!gameId) {
      const parsed = safeJsonParse(raw) as { type?: string; game_id?: string; payload?: Record<string, unknown>; data?: Record<string, unknown> } | undefined
      if (!parsed || typeof parsed.type !== "string") {
        return sendJson(ws, gameMsg("ERROR", { code: "INVALID_MESSAGE", message: "Could not parse message" }))
      }
      const body = parsed.payload ?? parsed.data ?? {}
      const type = parsed.type.toUpperCase()
      if (type === "PING") return sendJson(ws, gameMsg("PONG"))
      if (type === "CREATE_GAME") {
        gameId = crypto.randomUUID()
        const result = await this.gameStub(gameId).init({
          game_id: gameId,
          name: String(body["name"] ?? "Game"),
          map_id: String(body["map"] ?? body["mapId"] ?? "usa"),
          players: []
        })
        if (!result.ok) return sendJson(ws, gameMsg("ERROR", { code: "MESSAGE_ERROR", message: result.error }))
      } else {
        const fromBody = body["game_id"] ?? body["gameId"]
        gameId = typeof fromBody === "string" && fromBody ? fromBody : parsed.game_id
      }
      if (!gameId) {
        return sendJson(ws, gameMsg("ERROR", { code: "MESSAGE_ERROR", message: "session not in any game" }))
      }
      writeAttachment<Attachment>(ws, { ...att, game_id: gameId })
      if (type === "CREATE_GAME") {
        // Game exists now; let the GameDO answer with GAME_STATE via a synthetic CONNECT-less path.
        const snapshot = await this.gameStub(gameId).snapshot()
        return sendJson(ws, gameMsg("GAME_STATE", snapshot))
      }
    }
    await this.gameStub(gameId).relayMessage(att.key, raw)
  }

  private gameStub(gameId: string): DurableObjectStub<GameDO> {
    return this.env.GAME.get(this.env.GAME.idFromName(gameId)) as DurableObjectStub<GameDO>
  }

  // --- lobby protocol -------------------------------------------------------------

  private async lobbyIncoming(ws: WebSocket, att: Extract<Attachment, { kind: "lobby" }>, raw: string): Promise<void> {
    const parsed = safeJsonParse(raw)
    const env = parsed === undefined ? undefined : decodeEnvelope(parsed)
    if (!env || env._tag === "None") {
      return this.error(ws, att.temp, Err.InvalidFormat)
    }
    const msg = env.value
    const sessionId = msg.session_id || att.session_id || att.temp
    const session = this.sessions[sessionId]
    if (session) {
      session.last_activity = Date.now()
      if (att.session_id !== sessionId) writeAttachment<Attachment>(ws, { ...att, session_id: sessionId })
    }
    const data = msg.data ?? {}

    switch (msg.type) {
      case C.CONNECT: {
        const d = Schema.decodeUnknownOption(ConnectData)(data)
        const name = d._tag === "Some" ? d.value.player_name.trim() : ""
        if (!name) return this.error(ws, sessionId, Err.PlayerNameRequired)
        let s = this.sessions[sessionId]
        const reconnected = s !== undefined
        if (!s) {
          s = { player_id: crypto.randomUUID(), player_name: name, created_at: Date.now(), last_activity: Date.now() }
          this.sessions[sessionId] = s
        }
        writeAttachment<Attachment>(ws, { ...att, session_id: sessionId })
        await this.save()
        return sendJson(
          ws,
          lobbyMsg(S.CONNECTED, sessionId, {
            player_id: s.player_id,
            player_name: s.player_name,
            session_id: sessionId,
            reconnected,
            message: reconnected ? ConnectedMessages.restored : ConnectedMessages.created
          })
        )
      }
      case C.LIST_LOBBIES:
        return sendJson(ws, lobbyMsg(S.LOBBIES_LISTED, sessionId, { lobbies: this.summaries() }))
      case C.LIST_MAPS: {
        const maps = await this.runtime.runPromise(Effect.flatMap(Assets, (a) => a.listMaps)).catch(() => undefined)
        if (!maps) return this.error(ws, sessionId, Err.NoMapManager)
        return sendJson(ws, lobbyMsg(S.MAPS_LISTED, sessionId, { maps }))
      }
      case C.CREATE_LOBBY: {
        if (!session) return this.error(ws, sessionId, Err.NoSessionCreate)
        const d = Schema.decodeUnknownOption(CreateLobbyData)(data)
        const name = d._tag === "Some" ? d.value.lobby_name.trim() : ""
        if (!name) return this.error(ws, sessionId, Err.LobbyNameRequired)
        const mapId = (d._tag === "Some" && d.value.map_id) || "usa"
        const maps = await this.runtime.runPromise(Effect.flatMap(Assets, (a) => a.listMaps)).catch(() => [])
        if (!maps.some((m) => m.id === mapId)) return this.error(ws, sessionId, Err.InvalidMap)
        const maxPlayers = d._tag === "Some" && d.value.max_players && d.value.max_players > 0 ? Math.floor(d.value.max_players) : 6
        const password = d._tag === "Some" ? (d.value.password ?? "") : ""
        if (session.lobby_id) await this.leaveLobby(sessionId, session)
        const now = new Date().toISOString()
        const lobby: Lobby = {
          id: crypto.randomUUID(),
          name,
          status: "waiting",
          players: {
            [session.player_id]: { id: session.player_id, name: session.player_name, is_host: true, is_ready: true, joined_at: now }
          },
          messages: [],
          max_players: maxPlayers,
          map_id: mapId,
          password,
          created_at: now,
          updated_at: now
        }
        this.addSystemMessage(lobby, SystemMessages.created)
        this.lobbies[lobby.id] = lobby
        session.lobby_id = lobby.id
        await this.save()
        sendJson(ws, lobbyMsg(S.LOBBY_CREATED, sessionId, { lobby: this.toJSON(lobby) }))
        return this.broadcastAll(lobbyMsg(S.LOBBIES_LISTED, "", { lobbies: this.summaries() }))
      }
      case C.JOIN_LOBBY: {
        if (!session) return this.error(ws, sessionId, Err.NoSession)
        const d = Schema.decodeUnknownOption(JoinLobbyData)(data)
        const lobbyId = d._tag === "Some" ? d.value.lobby_id : ""
        if (!lobbyId) return this.error(ws, sessionId, Err.LobbyIdRequired)
        const lobby = this.lobbies[lobbyId]
        if (!lobby) return this.error(ws, sessionId, Err.LobbyNotFound)
        if (lobby.password && lobby.password !== ((d._tag === "Some" && d.value.password) || "")) {
          return this.error(ws, sessionId, Err.IncorrectPassword)
        }
        if (lobby.players[session.player_id] || Object.keys(lobby.players).length >= lobby.max_players || lobby.status !== "waiting") {
          return this.error(ws, sessionId, Err.JoinFailed)
        }
        if (session.lobby_id && session.lobby_id !== lobbyId) await this.leaveLobby(sessionId, session)
        lobby.players[session.player_id] = {
          id: session.player_id,
          name: session.player_name,
          is_host: false,
          is_ready: false,
          joined_at: new Date().toISOString()
        }
        this.addSystemMessage(lobby, SystemMessages.joined(session.player_name))
        session.lobby_id = lobbyId
        await this.save()
        sendJson(ws, lobbyMsg(S.LOBBY_JOINED, sessionId, { lobby: this.toJSON(lobby) }))
        return this.broadcastLobby(lobby)
      }
      case C.LEAVE_LOBBY: {
        if (!session) return this.error(ws, sessionId, Err.NoSession)
        if (!session.lobby_id) return this.error(ws, sessionId, Err.NotInLobby)
        const lobbyId = session.lobby_id
        await this.leaveLobby(sessionId, session)
        return sendJson(ws, lobbyMsg(S.LOBBY_LEFT, sessionId, { lobby_id: lobbyId }))
      }
      case C.CHAT_MESSAGE: {
        if (!session) return this.error(ws, sessionId, Err.NoSession)
        const lobby = session.lobby_id ? this.lobbies[session.lobby_id] : undefined
        if (!lobby) return this.error(ws, sessionId, Err.NotInLobby)
        const d = Schema.decodeUnknownOption(ChatData)(data)
        const content = d._tag === "Some" ? d.value.content.trim() : ""
        if (!content) return this.error(ws, sessionId, Err.ContentRequired)
        this.pushMessage(lobby, { id: crypto.randomUUID(), player_id: session.player_id, player_name: session.player_name, content, created_at: new Date().toISOString() })
        await this.save()
        return this.broadcastLobby(lobby)
      }
      case C.SET_READY: {
        if (!session) return this.error(ws, sessionId, Err.NoSession)
        const d = Schema.decodeUnknownOption(SetReadyData)(data)
        if (d._tag === "None") return this.error(ws, sessionId, Err.ReadyRequired)
        const lobby = session.lobby_id ? this.lobbies[session.lobby_id] : undefined
        const player = lobby?.players[session.player_id]
        if (!lobby || !player) return this.error(ws, sessionId, Err.ReadyFailed)
        player.is_ready = d.value.ready
        this.addSystemMessage(lobby, d.value.ready ? SystemMessages.ready(player.name) : SystemMessages.notReady(player.name))
        await this.save()
        sendJson(ws, lobbyMsg(S.READY_UPDATED, sessionId, { player_id: session.player_id, ready: d.value.ready }))
        return this.broadcastLobby(lobby)
      }
      case C.START_GAME: {
        if (!session) return this.error(ws, sessionId, Err.NoSession)
        const lobby = session.lobby_id ? this.lobbies[session.lobby_id] : undefined
        if (!lobby) return this.error(ws, sessionId, Err.NotInLobby)
        if (!lobby.players[session.player_id]?.is_host) return this.error(ws, sessionId, Err.HostOnly)
        const players = Object.values(lobby.players).sort((a, b) => a.joined_at.localeCompare(b.joined_at))
        if (lobby.status !== "waiting" || players.length < 2 || !players.every((p) => p.is_ready)) {
          return this.error(ws, sessionId, Err.CannotStart)
        }
        lobby.status = "starting"
        this.addSystemMessage(lobby, SystemMessages.starting)
        await this.save()
        const gameId = crypto.randomUUID()
        const result = await this.gameStub(gameId)
          .init({ game_id: gameId, name: lobby.name, map_id: lobby.map_id, players: players.map((p) => ({ id: p.id, name: p.name })) })
          .catch((e: unknown) => ({ ok: false as const, error: e instanceof Error ? e.message : String(e) }))
        if (!result.ok) {
          lobby.status = "waiting"
          await this.save()
          return this.error(ws, sessionId, Err.CreateGameFailed(result.error))
        }
        lobby.game_id = gameId
        await this.save()
        return this.broadcastLobbyMessage(lobby, lobbyMsg(S.GAME_STARTING, "", { lobby: this.toJSON(lobby), game_id: gameId, game_url: `/game?game_id=${gameId}` }))
      }
      default:
        return this.error(ws, sessionId, Err.UnknownType)
    }
  }

  // --- lobby helpers ---------------------------------------------------------------

  private async leaveLobby(sessionId: string, session: Session): Promise<void> {
    const lobby = session.lobby_id ? this.lobbies[session.lobby_id] : undefined
    delete session.lobby_id
    if (!lobby) return
    const leaving = lobby.players[session.player_id]
    delete lobby.players[session.player_id]
    if (leaving) this.addSystemMessage(lobby, SystemMessages.left(leaving.name))
    const remaining = Object.values(lobby.players).sort((a, b) => a.joined_at.localeCompare(b.joined_at))
    if (remaining.length === 0) {
      delete this.lobbies[lobby.id]
      await this.save()
      return
    }
    if (leaving?.is_host) {
      remaining[0]!.is_host = true
      this.addSystemMessage(lobby, SystemMessages.newHost(remaining[0]!.name))
    }
    await this.save()
    this.broadcastLobby(lobby)
    void sessionId
  }

  private async expireSession(sessionId: string): Promise<void> {
    const s = this.sessions[sessionId]
    if (!s) return
    if (s.lobby_id) await this.leaveLobby(sessionId, s)
    delete this.sessions[sessionId]
    await this.save()
  }

  private addSystemMessage(lobby: Lobby, content: string): void {
    this.pushMessage(lobby, { id: crypto.randomUUID(), player_id: "system", player_name: "System", content, created_at: new Date().toISOString() })
  }

  private pushMessage(lobby: Lobby, m: LobbyMessage): void {
    lobby.messages.push(m)
    if (lobby.messages.length > MAX_MESSAGES) lobby.messages.splice(0, lobby.messages.length - MAX_MESSAGES)
    lobby.updated_at = m.created_at
  }

  private toJSON(l: Lobby): LobbyJSON {
    return {
      id: l.id,
      name: l.name,
      status: l.status,
      players: l.players,
      messages: l.messages,
      max_players: l.max_players,
      map_id: l.map_id,
      created_at: l.created_at,
      updated_at: l.updated_at
    }
  }

  private summaries(): Array<LobbySummary> {
    return Object.values(this.lobbies)
      .sort((a, b) => a.created_at.localeCompare(b.created_at))
      .map((l) => ({
        id: l.id,
        name: l.name,
        status: l.status,
        player_count: Object.keys(l.players).length,
        max_players: l.max_players,
        map_id: l.map_id,
        has_password: l.password !== "",
        created_at: l.created_at
      }))
  }

  private socketsFor(playerIds: ReadonlySet<string>): Array<WebSocket> {
    const out: Array<WebSocket> = []
    for (const ws of this.ctx.getWebSockets("lobby")) {
      const att = readAttachment<Attachment>(ws, { kind: "lobby", temp: "" })
      if (att.kind !== "lobby" || !att.session_id) continue
      const s = this.sessions[att.session_id]
      if (s && playerIds.has(s.player_id)) out.push(ws)
    }
    return out
  }

  private broadcastLobby(lobby: Lobby): void {
    this.broadcastLobbyMessage(lobby, lobbyMsg(S.LOBBY_UPDATED, "", { lobby: this.toJSON(lobby) }))
  }

  private broadcastLobbyMessage(lobby: Lobby, msg: unknown): void {
    for (const ws of this.socketsFor(new Set(Object.keys(lobby.players)))) sendJson(ws, msg)
  }

  private broadcastAll(msg: unknown): void {
    for (const ws of this.ctx.getWebSockets("lobby")) sendJson(ws, msg)
  }

  private error(ws: WebSocket, sessionId: string, message: string): void {
    sendJson(ws, lobbyMsg(S.ERROR, sessionId, { message }))
  }

  private async save(): Promise<void> {
    await this.ctx.storage.put({ sessions: this.sessions, lobbies: this.lobbies })
  }
}
