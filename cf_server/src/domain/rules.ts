/**
 * Power Grid rule tables. Where the Go server had a value it is noted; otherwise the
 * standard Friedemann Friese rules are used (see docs/PORT_NOTES.md §5).
 */
import type { ResourceType } from "../protocol/game.ts"

export const STARTING_MONEY = 50
export const MAX_PLAYERS = 6
export const MIN_PLAYERS = 2
export const MAX_PLANTS_PER_PLAYER = 3
export const PLAYER_COLORS: ReadonlyArray<string> = ["red", "blue", "green", "yellow", "purple", "black"]

/** Income by number of cities powered. Index 20+ is capped at 150. (go) */
export const INCOME_TABLE: ReadonlyArray<number> = [
  10, 22, 33, 44, 54, 64, 73, 82, 90, 98, 105, 112, 118, 124, 129, 134, 138, 142, 145, 148, 150
]
export const incomeFor = (poweredCities: number): number =>
  INCOME_TABLE[Math.min(Math.max(poweredCities, 0), INCOME_TABLE.length - 1)]!

/** Cities needed to trigger game end, by player count. (go + maps/*.json winConditions) */
export const WIN_CITIES: Readonly<Record<number, number>> = { 2: 21, 3: 17, 4: 17, 5: 15, 6: 14 }
export const winCitiesFor = (players: number, override?: Readonly<Record<string, number>>): number =>
  override?.[String(players)] ?? WIN_CITIES[players] ?? 17

/** Step 2 begins at the end of the building phase in which a player reaches this many cities. */
export const DEFAULT_STEP2_TRIGGER = 7

/** Plants removed from the shuffled deck at setup, by player count. */
export const DECK_REMOVAL: Readonly<Record<number, number>> = { 2: 8, 3: 8, 4: 4, 5: 0, 6: 0 }

/** Regions in play by player count (standard rules). Null = whole map. */
export const REGIONS_IN_PLAY: Readonly<Record<number, number>> = { 2: 3, 3: 3, 4: 4, 5: 5, 6: 5 }

/** Price slot for the n-th house in a city (0-based). */
export const CITY_SLOT_COST: ReadonlyArray<number> = [10, 15, 20]
/** How many houses a city may hold in each step. */
export const slotsOpenInStep = (step: number): number => Math.min(Math.max(step, 1), 3)

export const MARKET_PRICES: Readonly<Record<ResourceType, { readonly maxPrice: number; readonly perSlot: number }>> = {
  Coal: { maxPrice: 8, perSlot: 3 },
  Oil: { maxPrice: 8, perSlot: 3 },
  Garbage: { maxPrice: 8, perSlot: 3 },
  Uranium: { maxPrice: 16, perSlot: 1 }
}
/** Uranium only has slots at 1..8 and 10, 12, 14, 16. */
export const marketSlotCapacity = (type: ResourceType, price: number): number => {
  const cfg = MARKET_PRICES[type]
  if (price < 1 || price > cfg.maxPrice) return 0
  if (type === "Uranium" && price > 8 && price % 2 === 1) return 0
  return cfg.perSlot
}
/** Wire array length is maxPrice + 1 (index = price, index 0 unused). (go) */
export const marketArrayLength = (type: ResourceType): number => MARKET_PRICES[type].maxPrice + 1

/** Total tokens in the game box. */
export const RESOURCE_SUPPLY: Readonly<Record<ResourceType, number>> = {
  Coal: 24,
  Oil: 24,
  Garbage: 24,
  Uranium: 12
}

/** Tokens placed in the market at setup (standard; maps/*.json resourceSupply agrees). */
export const INITIAL_MARKET: Readonly<Record<ResourceType, number>> = {
  Coal: 24,
  Oil: 18,
  Garbage: 6,
  Uranium: 2
}

/**
 * Resource replenishment per bureaucracy phase: [step][playerCount] → tokens.
 * Standard rules table.
 */
const REFILL: Readonly<Record<ResourceType, Readonly<Record<number, ReadonlyArray<number>>>>> = {
  //            players: 2  3  4  5  6
  Coal: { 1: [3, 4, 5, 5, 7], 2: [4, 5, 6, 7, 9], 3: [3, 3, 4, 5, 6] },
  Oil: { 1: [2, 2, 3, 4, 5], 2: [2, 3, 4, 5, 6], 3: [4, 4, 5, 6, 7] },
  Garbage: { 1: [1, 1, 2, 3, 3], 2: [2, 2, 3, 3, 5], 3: [3, 3, 4, 5, 6] },
  Uranium: { 1: [1, 1, 1, 2, 2], 2: [1, 1, 2, 3, 3], 3: [1, 1, 2, 2, 3] }
}
export const refillAmount = (type: ResourceType, step: number, players: number): number => {
  const row = REFILL[type][Math.min(Math.max(step, 1), 3)]!
  return row[Math.min(Math.max(players, 2), 6) - 2]!
}

export const STEP3_CARD = -1 as const
