/**
 * `/game` protocol. Envelope: {type, timestamp, session_id?, game_id?, payload?}.
 * Payload shapes match pkg/protocol/messages.go. Additive fields on GameStatePayload
 * (`step`, `auction`, `plant_market`, `winner_id`) are new; clients ignore unknown keys.
 */
import { Schema } from "effect"

export const GameEnvelope = Schema.Struct({
  type: Schema.String,
  timestamp: Schema.optional(Schema.Number),
  session_id: Schema.optional(Schema.String),
  game_id: Schema.optional(Schema.String),
  player_id: Schema.optional(Schema.String),
  payload: Schema.optional(Schema.Unknown),
  // React client dialect sends `data` instead of `payload`; accepted as an alias.
  data: Schema.optional(Schema.Unknown)
})
export type GameEnvelope = typeof GameEnvelope.Type

export const MsgType = {
  CONNECT: "CONNECT",
  DISCONNECT: "DISCONNECT",
  PING: "PING",
  PONG: "PONG",
  CREATE_GAME: "CREATE_GAME",
  JOIN_GAME: "JOIN_GAME",
  LEAVE_GAME: "LEAVE_GAME",
  LIST_GAMES: "LIST_GAMES",
  START_GAME: "START_GAME",
  GAME_STATE: "GAME_STATE",
  BID_PLANT: "BID_PLANT",
  BUY_RESOURCES: "BUY_RESOURCES",
  BUILD_CITY: "BUILD_CITY",
  POWER_CITIES: "POWER_CITIES",
  END_TURN: "END_TURN",
  ERROR: "ERROR",
  PHASE_CHANGE: "PHASE_CHANGE",
  TURN_CHANGE: "TURN_CHANGE",
  PLAYER_JOINED: "PLAYER_JOINED",
  AI_DECISION: "AI_DECISION"
} as const
export type MsgType = (typeof MsgType)[keyof typeof MsgType]

export const GameStatus = Schema.Literals(["LOBBY", "PLAYING", "FINISHED"])
export type GameStatus = typeof GameStatus.Type

export const GamePhase = Schema.Literals([
  "PLAYER_ORDER",
  "AUCTION",
  "BUY_RESOURCES",
  "BUILD_CITIES",
  "BUREAUCRACY",
  "GAME_END"
])
export type GamePhase = typeof GamePhase.Type

/** Resources that exist in the market / player storage. Title case on the wire. */
export const ResourceType = Schema.Literals(["Coal", "Oil", "Garbage", "Uranium"])
export type ResourceType = typeof ResourceType.Type
export const RESOURCE_TYPES: ReadonlyArray<ResourceType> = ["Coal", "Oil", "Garbage", "Uranium"]

export const PlantResourceType = Schema.Literals(["Coal", "Oil", "Garbage", "Uranium", "Hybrid", "Wind"])
export type PlantResourceType = typeof PlantResourceType.Type

// --- client payloads --------------------------------------------------------

export const ConnectPayload = Schema.Struct({
  player_name: Schema.optional(Schema.String),
  player_id: Schema.optional(Schema.String),
  game_id: Schema.optional(Schema.String)
})
export const CreateGamePayload = Schema.Struct({
  name: Schema.String,
  map: Schema.optional(Schema.String),
  max_players: Schema.optional(Schema.Number)
})
export const JoinGamePayload = Schema.Struct({
  game_id: Schema.optional(Schema.String),
  player_name: Schema.String,
  color: Schema.optional(Schema.String)
})
export const BidPlantPayload = Schema.Struct({
  plant_id: Schema.Number,
  bid: Schema.Number
})
export const BuyResourcesPayload = Schema.Struct({
  resources: Schema.Record(Schema.String, Schema.Number)
})
export const BuildCityPayload = Schema.Struct({ city_id: Schema.String })
export const PowerCitiesPayload = Schema.Struct({ power_plants: Schema.Array(Schema.Number) })

// --- server payloads --------------------------------------------------------

export const PowerPlantInfo = Schema.Struct({
  id: Schema.Number,
  cost: Schema.Number,
  capacity: Schema.Number,
  resource_type: PlantResourceType,
  resource_cost: Schema.Number
})
export type PowerPlantInfo = typeof PowerPlantInfo.Type

export const PlayerInfo = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  color: Schema.String,
  money: Schema.Number,
  cities: Schema.Array(Schema.String),
  power_plants: Schema.NullOr(Schema.Array(PowerPlantInfo)),
  resources: Schema.Record(Schema.String, Schema.Number),
  powered_cities: Schema.Number,
  // additive
  connected: Schema.optional(Schema.Boolean),
  has_passed: Schema.optional(Schema.Boolean)
})
export type PlayerInfo = typeof PlayerInfo.Type

export const CityInfo = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  region: Schema.String,
  position: Schema.Tuple([Schema.Number, Schema.Number]),
  slots: Schema.Array(Schema.String)
})
export type CityInfo = typeof CityInfo.Type

export const ConnectionInfo = Schema.Struct({
  city_a: Schema.String,
  city_b: Schema.String,
  cost: Schema.Number
})
export type ConnectionInfo = typeof ConnectionInfo.Type

export const MapInfoWire = Schema.Struct({
  name: Schema.String,
  cities: Schema.Record(Schema.String, CityInfo),
  connections: Schema.Array(ConnectionInfo)
})
export type MapInfoWire = typeof MapInfoWire.Type

/** resources[type][price] = token count. Coal/Oil/Garbage length 9, Uranium length 17. */
export const MarketInfo = Schema.Struct({
  resources: Schema.Record(Schema.String, Schema.Array(Schema.Number))
})
export type MarketInfo = typeof MarketInfo.Type

export const AuctionInfo = Schema.Struct({
  plant_id: Schema.Number,
  current_bid: Schema.Number,
  current_bidder: Schema.String,
  nominated_by: Schema.String,
  participants: Schema.Array(Schema.String),
  passed: Schema.Array(Schema.String)
})
export type AuctionInfo = typeof AuctionInfo.Type

export const GameStatePayload = Schema.Struct({
  game_id: Schema.String,
  name: Schema.String,
  status: GameStatus,
  current_phase: GamePhase,
  current_turn: Schema.String,
  current_round: Schema.Number,
  players: Schema.Record(Schema.String, PlayerInfo),
  map: MapInfoWire,
  market: MarketInfo,
  power_plants: Schema.NullOr(Schema.Array(PowerPlantInfo)),
  turn_order: Schema.Array(Schema.String),
  // additive
  step: Schema.Number,
  auction: Schema.NullOr(AuctionInfo),
  plant_market: Schema.Struct({
    current: Schema.Array(PowerPlantInfo),
    future: Schema.Array(PowerPlantInfo),
    deck_remaining: Schema.Number
  }),
  winner_id: Schema.NullOr(Schema.String),
  map_id: Schema.String,
  active_regions: Schema.Array(Schema.String)
})
export type GameStatePayload = typeof GameStatePayload.Type

export const ErrorPayload = Schema.Struct({ code: Schema.String, message: Schema.String })
export type ErrorPayload = typeof ErrorPayload.Type
export const ErrorCode = {
  MESSAGE_ERROR: "MESSAGE_ERROR",
  INVALID_MESSAGE: "INVALID_MESSAGE",
  SERVER_BUSY: "SERVER_BUSY"
} as const

export const PhaseChangePayload = Schema.Struct({ phase: GamePhase, round: Schema.Number })
export const TurnChangePayload = Schema.Struct({ current_player_id: Schema.String, turn: Schema.Number })
export const PlayerJoinedPayload = Schema.Struct({ player_id: Schema.String, player_name: Schema.String })

/** Engine error strings. Those marked (go) are identical to the Go server. */
export const GameErrors = {
  NotPlaying: "game is not in playing state", // go
  NotYourTurn: "not your turn", // go
  UnknownAction: "unknown action type", // go
  InvalidBidPayload: "invalid payload for bid plant", // go
  InvalidBuyPayload: "invalid payload for buy resources", // go
  InvalidBuildPayload: "invalid payload for build city", // go
  InvalidPowerPayload: "invalid payload for power cities", // go
  NotAuctionPhase: "not in auction phase", // go
  NoActiveAuction: "no active auction", // go
  WrongPlantAuctioned: "this plant is not being auctioned", // go
  NotYourBid: "not your turn to bid", // go
  BidTooLow: "bid must be higher than current bid", // go
  BidNoMoney: "not enough money for this bid", // go
  PlantNotInMarket: "plant not found in market", // go
  NotInCurrentMarket: "plant is in the future market and cannot be auctioned yet",
  AlreadyBoughtPlant: "player already bought a plant this round",
  MustBuyFirstRound: "every player must buy a plant in the first round",
  NotBuyPhase: "not in buy resources phase", // go
  PlayerNotFound: "player not found", // go
  ResourceNotFound: "resource type not found", // go
  NotEnoughResources: "not enough resources available", // go
  NotEnoughMoney: "not enough money", // go
  NoStorage: "not enough storage capacity on your power plants",
  NotBuildPhase: "not in build cities phase", // go
  CityNotFound: "city not found", // go
  CityFull: "city is full", // go
  CityClosedThisStep: "this city has no open slot in the current step",
  CityNotInPlay: "this city is outside the regions in play",
  AlreadyInCity: "player already has a house in this city", // go
  NotConnected: "city not connected to player's network", // go
  NotBureaucracyPhase: "not in bureaucracy phase", // go
  PlantNotOwned: "player does not own specified power plant", // go
  CannotPower: "not enough resources to power this plant", // go
  AlreadyPowered: "player already powered cities this round",
  GameFull: "game is full", // go
  AlreadyStarted: "game has already started", // go
  ColorTaken: "color already taken", // go
  NeedTwoPlayers: "need at least 2 players to start", // go
  NotInGame: "session not in any game", // go
  GameNotFound: (id: string) => `game not found: ${id}`, // go
  UnknownMessage: (t: string) => `unknown message type: ${t}` // go
} as const
