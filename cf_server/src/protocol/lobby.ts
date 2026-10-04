/**
 * `/ws` lobby protocol. Envelope: {type, session_id?, player_id?, timestamp, data?}.
 * Field names and error strings are byte-for-byte those of handlers/lobby_handler.go.
 */
import { Schema } from "effect"
import { JsonObject } from "./common.ts"

export const LobbyEnvelope = Schema.Struct({
  type: Schema.String,
  session_id: Schema.optional(Schema.String),
  player_id: Schema.optional(Schema.String),
  timestamp: Schema.optional(Schema.Number),
  data: Schema.optional(JsonObject)
})
export type LobbyEnvelope = typeof LobbyEnvelope.Type

export const LobbyClientType = {
  CONNECT: "CONNECT",
  DISCONNECT: "DISCONNECT",
  CREATE_LOBBY: "CREATE_LOBBY",
  JOIN_LOBBY: "JOIN_LOBBY",
  LEAVE_LOBBY: "LEAVE_LOBBY",
  CHAT_MESSAGE: "CHAT_MESSAGE",
  LIST_LOBBIES: "LIST_LOBBIES",
  LIST_MAPS: "LIST_MAPS",
  SET_READY: "SET_READY",
  START_GAME: "START_GAME"
} as const

export const LobbyServerType = {
  CONNECTED: "CONNECTED",
  ERROR: "ERROR",
  LOBBY_CREATED: "LOBBY_CREATED",
  LOBBY_JOINED: "LOBBY_JOINED",
  LOBBY_LEFT: "LOBBY_LEFT",
  LOBBIES_LISTED: "LOBBIES_LISTED",
  MAPS_LISTED: "MAPS_LISTED",
  LOBBY_UPDATED: "LOBBY_UPDATED",
  READY_UPDATED: "READY_UPDATED",
  GAME_STARTING: "GAME_STARTING"
} as const

// --- client payloads ------------------------------------------------------

export const ConnectData = Schema.Struct({ player_name: Schema.String })
export const CreateLobbyData = Schema.Struct({
  lobby_name: Schema.String,
  max_players: Schema.optional(Schema.Number),
  map_id: Schema.optional(Schema.String),
  password: Schema.optional(Schema.String)
})
export const JoinLobbyData = Schema.Struct({
  lobby_id: Schema.String,
  password: Schema.optional(Schema.String)
})
export const ChatData = Schema.Struct({ content: Schema.String })
export const SetReadyData = Schema.Struct({ ready: Schema.Boolean })

// --- server payload shapes --------------------------------------------------

export const LobbyStatus = Schema.Literals(["waiting", "starting", "in_game", "ended"])
export type LobbyStatus = typeof LobbyStatus.Type

export const LobbyPlayer = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  is_host: Schema.Boolean,
  is_ready: Schema.Boolean,
  joined_at: Schema.String
})
export type LobbyPlayer = typeof LobbyPlayer.Type

export const LobbyMessage = Schema.Struct({
  id: Schema.String,
  player_id: Schema.String,
  player_name: Schema.String,
  content: Schema.String,
  created_at: Schema.String
})
export type LobbyMessage = typeof LobbyMessage.Type

export const LobbyJSON = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  status: LobbyStatus,
  players: Schema.Record(Schema.String, LobbyPlayer),
  messages: Schema.Array(LobbyMessage),
  max_players: Schema.Number,
  map_id: Schema.String,
  created_at: Schema.String,
  updated_at: Schema.String
})
export type LobbyJSON = typeof LobbyJSON.Type

export const LobbySummary = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  status: LobbyStatus,
  player_count: Schema.Number,
  max_players: Schema.Number,
  map_id: Schema.String,
  has_password: Schema.Boolean,
  created_at: Schema.String
})
export type LobbySummary = typeof LobbySummary.Type

/** Note: camelCase, matching internal/maps/map_manager.go MapInfo. */
export const MapInfo = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  description: Schema.String,
  playerCount: Schema.Struct({
    min: Schema.Number,
    max: Schema.Number,
    recommended: Schema.Array(Schema.Number)
  }),
  regionCount: Schema.Number,
  cityCount: Schema.Number
})
export type MapInfo = typeof MapInfo.Type

export const LobbyErrors = {
  InvalidFormat: "Invalid message format",
  UnknownType: "Unknown message type",
  NoSessionCreate: "No active session found. Please send CONNECT message first.",
  NoSessionCreateInner: "Player session not found. Please send CONNECT message first.",
  NoSession: "Player session not found",
  PlayerNameRequired: "Player name is required",
  LobbyNameRequired: "Lobby name is required",
  InvalidMap: "Invalid map selected",
  LobbyIdRequired: "Lobby ID is required",
  LobbyNotFound: "Lobby not found",
  IncorrectPassword: "Incorrect password",
  JoinFailed: "Failed to join lobby",
  NotInLobby: "Player not in a lobby",
  ContentRequired: "Message content is required",
  NoMapManager: "Map manager not available",
  ReadyRequired: "Ready status is required",
  ReadyFailed: "Failed to update ready status",
  HostOnly: "Only the host can start the game",
  CannotStart: "Cannot start the game",
  CreateGameFailed: (err: string) => `Failed to create game: ${err}`
} as const

export const SystemMessages = {
  created: "Lobby created. Waiting for players...",
  joined: (n: string) => `${n} joined the lobby`,
  left: (n: string) => `${n} left the lobby`,
  newHost: (n: string) => `${n} is now the host`,
  ready: (n: string) => `${n} is ready`,
  notReady: (n: string) => `${n} is not ready`,
  starting: "Game is starting..."
} as const

export const ConnectedMessages = {
  created: "New session created successfully",
  restored: "Session restored successfully"
} as const
