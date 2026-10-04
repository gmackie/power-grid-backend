/** Power plant deck and plant market operations. */
import type { PlantDef } from "./deck-default.ts"
import { DECK_REMOVAL } from "./rules.ts"
import { type Rng, shuffle } from "./rng.ts"
import type { Plant } from "./state.ts"

export const toPlant = (d: PlantDef): Plant => ({
  number: d.number,
  capacity: d.capacity,
  resourceType: d.resourceType,
  resourceCost: d.resourceCost
})

export const byNumber = (a: Plant, b: Plant): number => a.number - b.number

export interface DeckSetup {
  readonly current: Array<Plant>
  readonly future: Array<Plant>
  readonly deck: Array<Plant | null>
  readonly rng: Rng
}

/**
 * Standard setup: lowest 8 plants form the market (4 current, 4 future). Plant 13 goes on
 * top of the shuffled remainder, DECK_REMOVAL[players] cards are removed first, and the
 * Step 3 card (null) goes to the bottom.
 */
export const setupDeck = (defs: ReadonlyArray<PlantDef>, players: number, rng0: Rng): DeckSetup => {
  const all = defs.map(toPlant).sort(byNumber)
  const market = all.slice(0, 8)
  const rest = all.slice(8)
  const thirteen = rest.find((p) => p.number === 13)
  const others = rest.filter((p) => p.number !== 13)
  const [shuffled, rng] = shuffle(rng0, others)
  const remove = DECK_REMOVAL[players] ?? 0
  const kept = shuffled.slice(Math.min(remove, shuffled.length))
  const deck: Array<Plant | null> = thirteen ? [thirteen, ...kept, null] : [...kept, null]
  return { current: market.slice(0, 4), future: market.slice(4, 8), deck, rng }
}

export interface MarketState {
  readonly current: ReadonlyArray<Plant>
  readonly future: ReadonlyArray<Plant>
  readonly deck: ReadonlyArray<Plant | null>
  readonly step: 1 | 2 | 3
  readonly step3Drawn: boolean
  readonly rng: Rng
}

/** Re-sort current/future after a change. Step 3: 6-card market, all current. */
const arrange = (m: MarketState, pool: ReadonlyArray<Plant>): MarketState => {
  const sorted = [...pool].sort(byNumber)
  if (m.step === 3) return { ...m, current: sorted.slice(0, 6), future: [] }
  return { ...m, current: sorted.slice(0, 4), future: sorted.slice(4, 8) }
}

/**
 * Draw the top card into the market. Drawing the Step 3 card marks step3Drawn, shuffles
 * the remaining deck and removes the lowest current plant (standard rules).
 */
export const draw = (m: MarketState): MarketState => {
  const [top, ...rest] = m.deck
  if (top === undefined) return m
  if (top === null) {
    const [shuffled, rng] = shuffle(m.rng, rest.filter((p): p is Plant => p !== null))
    const pool = [...m.current, ...m.future].sort(byNumber).slice(1)
    return arrange({ ...m, deck: shuffled, rng, step3Drawn: true }, pool)
  }
  return arrange({ ...m, deck: rest }, [...m.current, ...m.future, top])
}

/** Remove a purchased plant and draw its replacement. */
export const takeFromMarket = (m: MarketState, plantNumber: number): MarketState => {
  const pool = [...m.current, ...m.future].filter((p) => p.number !== plantNumber)
  return draw(arrange(m, pool))
}

/** Remove the lowest plant from the market (unsold / step change) and draw. */
export const removeLowest = (m: MarketState): MarketState => {
  const pool = [...m.current, ...m.future].sort(byNumber).slice(1)
  return draw(arrange(m, pool))
}

/** Bureaucracy market update. Step 1/2: highest future plant under the deck. Step 3: lowest current removed. */
export const bureaucracyUpdate = (m: MarketState): MarketState => {
  if (m.step === 3) return removeLowest(m)
  const pool = [...m.current, ...m.future].sort(byNumber)
  const highest = pool[pool.length - 1]
  if (!highest) return m
  const next: MarketState = { ...m, deck: [...m.deck, highest] }
  return draw(arrange(next, pool.slice(0, -1)))
}

/** Enter step 3: six-plant market. */
export const enterStep3 = (m: MarketState): MarketState =>
  arrange({ ...m, step: 3 }, [...m.current, ...m.future])
