/** Serializable engine state. Stored as JSON in Durable Object storage. */
import type { GamePhase, GameStatus, PlantResourceType, ResourceType } from "../protocol/game.ts"
import type { MapData } from "./map.ts"
import type { ResourceMarket } from "./market.ts"

export interface Plant {
  readonly number: number
  readonly capacity: number
  readonly resourceType: PlantResourceType
  readonly resourceCost: number
}

export type ResourceStore = Readonly<Record<ResourceType, number>>
export const emptyStore = (): ResourceStore => ({ Coal: 0, Oil: 0, Garbage: 0, Uranium: 0 })

export interface PlayerState {
  readonly id: string
  readonly name: string
  readonly color: string
  readonly money: number
  readonly plants: ReadonlyArray<Plant>
  readonly resources: ResourceStore
  readonly cities: ReadonlyArray<string>
  readonly poweredCities: number
  /** Auction phase: has bought a plant (or passed nomination) this round. */
  readonly doneThisPhase: boolean
  /** Bureaucracy: has already powered this round. */
  readonly poweredThisRound: boolean
  readonly connected: boolean
}

export interface AuctionState {
  readonly plantNumber: number
  readonly nominatedBy: string
  readonly currentBid: number
  readonly currentBidder: string
  /** Players still bidding, in turn order. */
  readonly participants: ReadonlyArray<string>
}

export interface GameState {
  readonly id: string
  readonly name: string
  readonly mapId: string
  readonly map: MapData
  readonly status: GameStatus
  readonly phase: GamePhase
  readonly round: number
  readonly step: 1 | 2 | 3
  /** Index into turnOrder of the acting player. */
  readonly turnIndex: number
  readonly turnOrder: ReadonlyArray<string>
  readonly players: Readonly<Record<string, PlayerState>>
  /** Order players were added (used for colors / seating). */
  readonly seating: ReadonlyArray<string>
  readonly citySlots: Readonly<Record<string, ReadonlyArray<string>>>
  /** Region ids in play (standard: 3/3/4/5/5 contiguous regions for 2-6 players). Empty = whole map. */
  readonly activeRegions: ReadonlyArray<string>
  readonly market: ResourceMarket
  readonly currentMarket: ReadonlyArray<Plant>
  readonly futureMarket: ReadonlyArray<Plant>
  /** Remaining draw pile; `null` entry = Step 3 card. */
  readonly deck: ReadonlyArray<Plant | null>
  readonly auction: AuctionState | null
  /** True once a player reached the city target; game ends after this bureaucracy. */
  readonly endTriggered: boolean
  /** Step 3 card drawn; step becomes 3 at the end of the current phase. */
  readonly step3Pending: boolean
  /** A plant was bought during the current auction phase. */
  readonly auctionHadPurchase: boolean
  readonly winnerId: string | null
  readonly createdAt: number
  readonly updatedAt: number
  /** RNG seed for deterministic replays/tests. */
  readonly seed: number
}

export const currentPlayerId = (s: GameState): string => s.turnOrder[s.turnIndex] ?? ""

export const playerCount = (s: GameState): number => Object.keys(s.players).length
