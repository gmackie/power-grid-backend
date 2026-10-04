/**
 * Bot players for the simulation harness. A bot is a pure function from (state, playerId)
 * to an engine Action. `HeuristicBot` is parameterised by a weight vector so the evolve
 * loop can search strategy space per board.
 */
import type { Action } from "../domain/engine.ts"
import { actingPlayerId, cityInPlay } from "../domain/engine.ts"
import { buildGraph, connectionCost, type MapGraph } from "../domain/map.ts"
import * as Market from "../domain/market.ts"
import { CITY_SLOT_COST, slotsOpenInStep } from "../domain/rules.ts"
import type { GameState, Plant, PlayerState } from "../domain/state.ts"
import { RESOURCE_TYPES, type ResourceType } from "../protocol/game.ts"
import { nextFloat, type Rng } from "../domain/rng.ts"

export interface Weights {
  /** How far above face value the bot will chase a plant (0 = never raise). */
  aggression: number
  /** Value per city of capacity when scoring plants. */
  capacity: number
  /** Penalty per unit of per-use resource cost. */
  fuelCost: number
  /** Preference for green/wind plants (no fuel). */
  green: number
  /** Cash reserve kept after building, as a multiple of next-round fuel bill. */
  reserve: number
  /** Extra fuel bought beyond one round of consumption (0..1 of capacity). */
  buffer: number
  /** How many cities to try to build per turn beyond powered capacity. */
  overbuild: number
  /** Probability of passing nomination once holding >= 2 plants (after round 1). */
  passNomination: number
}

export interface Bot {
  readonly name: string
  readonly weights?: Weights
  decide(state: GameState, playerId: string, rng: Rng): [Action, Rng]
}

export const DEFAULT_WEIGHTS: Weights = { aggression: 0.3, capacity: 10, fuelCost: 2, green: 4, reserve: 1, buffer: 0.3, overbuild: 1, passNomination: 0.4 }

const graphCache = new WeakMap<object, MapGraph>()
const graphOf = (s: GameState): MapGraph => {
  let g = graphCache.get(s.map)
  if (!g) { g = buildGraph(s.map); graphCache.set(s.map, g) }
  return g
}

const plantValue = (p: Plant, w: Weights): number =>
  p.capacity * w.capacity - p.resourceCost * w.fuelCost + (p.resourceType === "Wind" ? w.green : 0) - p.number * 0.4

/** One round of fuel for every plant (hybrid counted as coal). */
const fuelNeed = (p: PlayerState): Record<ResourceType, number> => {
  const need: Record<ResourceType, number> = { Coal: 0, Oil: 0, Garbage: 0, Uranium: 0 }
  for (const pl of p.plants) {
    if (pl.resourceType === "Wind") continue
    if (pl.resourceType === "Hybrid") need.Coal += pl.resourceCost
    else need[pl.resourceType] += pl.resourceCost
  }
  return need
}

const storageCap = (p: PlayerState): Record<ResourceType, number> => {
  const cap: Record<ResourceType, number> = { Coal: 0, Oil: 0, Garbage: 0, Uranium: 0 }
  for (const pl of p.plants) {
    if (pl.resourceType === "Wind") continue
    if (pl.resourceType === "Hybrid") cap.Coal += 2 * pl.resourceCost
    else cap[pl.resourceType] += 2 * pl.resourceCost
  }
  return cap
}

const powerablePlants = (p: PlayerState): Array<number> => {
  const store = { ...p.resources }
  const ids: Array<number> = []
  for (const pl of [...p.plants].sort((a, b) => b.capacity - a.capacity)) {
    if (pl.resourceType === "Wind") { ids.push(pl.number); continue }
    if (pl.resourceType === "Hybrid") {
      const avail = store.Coal + store.Oil
      if (avail < pl.resourceCost) continue
      let need = pl.resourceCost
      const c = Math.min(store.Coal, need); store.Coal -= c; need -= c; store.Oil -= need
      ids.push(pl.number); continue
    }
    if (store[pl.resourceType] >= pl.resourceCost) { store[pl.resourceType] -= pl.resourceCost; ids.push(pl.number) }
  }
  return ids
}

const capacityOf = (p: PlayerState): number => p.plants.reduce((s, pl) => s + pl.capacity, 0)

const cheapestBuild = (s: GameState, p: PlayerState): { cityId: string; cost: number } | undefined => {
  const g = graphOf(s)
  let best: { cityId: string; cost: number } | undefined
  for (const c of s.map.cities) {
    if (!cityInPlay(s, c.id)) continue
    const slots = s.citySlots[c.id] ?? []
    if (slots.includes(p.id) || slots.length >= slotsOpenInStep(s.step)) continue
    const conn = connectionCost(g, p.cities, c.id)
    if (conn === undefined) continue
    const cost = conn + CITY_SLOT_COST[slots.length]!
    if (!best || cost < best.cost) best = { cityId: c.id, cost }
  }
  return best
}

export const heuristicBot = (name: string, w: Weights = DEFAULT_WEIGHTS): Bot => ({
  name,
  weights: w,
  decide(s, pid, rng0) {
    const me = s.players[pid]!
    let rng = rng0
    const roll = () => { const [f, r] = nextFloat(rng); rng = r; return f }
    switch (s.phase) {
      case "AUCTION": {
        if (s.auction) {
          const a = s.auction
          if (a.currentBidder !== pid) return [{ _tag: "EndTurn" }, rng]
          const plant = [...s.currentMarket, ...s.futureMarket].find((p) => p.number === a.plantNumber)
          const cap = plant ? plant.number + plantValue(plant, w) * w.aggression : 0
          const next = a.currentBid + 1
          if (next <= cap && next <= me.money - 10) return [{ _tag: "BidPlant", plantId: a.plantNumber, bid: next }, rng]
          return [{ _tag: "BidPlant", plantId: a.plantNumber, bid: 0 }, rng]
        }
        if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
        if (s.round > 1 && me.plants.length >= 2 && roll() < w.passNomination) return [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
        if (s.currentMarket.length === 0) return [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
        const affordable = s.currentMarket.filter((p) => p.number <= me.money - 8)
        if (affordable.length === 0) return s.round === 1 ? [{ _tag: "BidPlant", plantId: s.currentMarket[0]!.number, bid: s.currentMarket[0]!.number }, rng] : [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
        const owned = capacityOf(me)
        const pick = affordable.reduce((b, p) => (plantValue(p, w) + (p.capacity > owned ? 2 : 0) > plantValue(b, w) ? p : b))
        if (s.round > 1 && me.plants.length === 3 && plantValue(pick, w) <= Math.min(...me.plants.map((p) => plantValue(p, w)))) return [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
        return [{ _tag: "BidPlant", plantId: pick.number, bid: pick.number }, rng]
      }
      case "BUY_RESOURCES": {
        if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
        const need = fuelNeed(me)
        const cap = storageCap(me)
        const wanted: Record<string, number> = {}
        let budget = me.money - 12
        for (const t of RESOURCE_TYPES) {
          const target = Math.min(cap[t], Math.ceil(need[t] * (1 + w.buffer)))
          let amount = Math.max(0, target - me.resources[t])
          while (amount > 0) {
            const cost = Market.purchaseCost(s.market, t, amount)
            if (cost !== undefined && cost <= budget) { budget -= cost; wanted[t] = amount; break }
            amount--
          }
        }
        if (Object.keys(wanted).length === 0 || me.resources.Coal + me.resources.Oil + me.resources.Garbage + me.resources.Uranium > 0 && Object.values(wanted).every((v) => v === 0)) return [{ _tag: "EndTurn" }, rng]
        return [{ _tag: "BuyResources", resources: wanted }, rng]
      }
      case "BUILD_CITIES": {
        if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
        const fuelBill = Object.entries(fuelNeed(me)).reduce((sum, [t, n]) => sum + (Market.purchaseCost(s.market, t as ResourceType, n) ?? n * 8), 0)
        const reserve = fuelBill * w.reserve
        const target = capacityOf(me) + w.overbuild
        if (me.cities.length >= target && me.cities.length > 0) return [{ _tag: "EndTurn" }, rng]
        const best = cheapestBuild(s, me)
        if (!best || me.money - best.cost < reserve) return [{ _tag: "EndTurn" }, rng]
        return [{ _tag: "BuildCity", cityId: best.cityId }, rng]
      }
      case "BUREAUCRACY": {
        if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
        if (me.poweredThisRound) return [{ _tag: "EndTurn" }, rng]
        return [{ _tag: "PowerCities", plantIds: powerablePlants(me) }, rng]
      }
      default:
        return [{ _tag: "EndTurn" }, rng]
    }
  }
})

export const randomBot = (name: string): Bot => ({
  name,
  decide(s, pid, rng0) {
    let rng = rng0
    const roll = () => { const [f, r] = nextFloat(rng); rng = r; return f }
    const me = s.players[pid]!
    if (s.phase === "AUCTION") {
      if (s.auction) {
        if (s.auction.currentBidder !== pid) return [{ _tag: "EndTurn" }, rng]
        const raise = roll() < 0.4 && s.auction.currentBid + 1 < me.money
        return [{ _tag: "BidPlant", plantId: s.auction.plantNumber, bid: raise ? s.auction.currentBid + 1 : 0 }, rng]
      }
      if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
      if (s.currentMarket.length === 0) return [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
      const pick = s.currentMarket[Math.floor(roll() * s.currentMarket.length)]!
      if (s.round > 1 && roll() < 0.5) return [{ _tag: "BidPlant", plantId: 0, bid: 0 }, rng]
      return [{ _tag: "BidPlant", plantId: pick.number, bid: pick.number }, rng]
    }
    if (actingPlayerId(s) !== pid) return [{ _tag: "EndTurn" }, rng]
    if (s.phase === "BUY_RESOURCES") {
      const need = fuelNeed(me)
      const cap = storageCap(me)
      const wanted: Record<string, number> = {}
      let budget = me.money - 5
      for (const t of RESOURCE_TYPES) {
        let a = Math.min(need[t], Math.max(0, cap[t] - me.resources[t]))
        while (a > 0) { const c = Market.purchaseCost(s.market, t, a); if (c !== undefined && c <= budget) { budget -= c; break } a-- }
        if (a > 0 && roll() < 0.8) wanted[t] = a
      }
      return Object.keys(wanted).length ? [{ _tag: "BuyResources", resources: wanted }, rng] : [{ _tag: "EndTurn" }, rng]
    }
    if (s.phase === "BUILD_CITIES") {
      const best = cheapestBuild(s, me)
      if (best && best.cost < me.money - 5 && roll() < 0.7) return [{ _tag: "BuildCity", cityId: best.cityId }, rng]
      return [{ _tag: "EndTurn" }, rng]
    }
    if (s.phase === "BUREAUCRACY" && !me.poweredThisRound) return [{ _tag: "PowerCities", plantIds: powerablePlants(me) }, rng]
    return [{ _tag: "EndTurn" }, rng]
  }
})

/** Named archetypes to compare on every board. */
export const ARCHETYPES: ReadonlyArray<Bot> = [
  heuristicBot("balanced"),
  heuristicBot("expander", { ...DEFAULT_WEIGHTS, overbuild: 3, reserve: 0.4, capacity: 12 }),
  heuristicBot("hoarder", { ...DEFAULT_WEIGHTS, buffer: 0.9, reserve: 1.6, overbuild: 0 }),
  heuristicBot("sniper", { ...DEFAULT_WEIGHTS, aggression: 0.9, capacity: 14, fuelCost: 1 }),
  heuristicBot("greenie", { ...DEFAULT_WEIGHTS, green: 20, fuelCost: 5 }),
  randomBot("random")
]

export const mutate = (w: Weights, rng0: Rng, scale = 0.35): [Weights, Rng] => {
  let rng = rng0
  const out = { ...w }
  for (const k of Object.keys(out) as Array<keyof Weights>) {
    const [f, r] = nextFloat(rng); rng = r
    if (f < 0.5) continue
    const [g, r2] = nextFloat(rng); rng = r2
    const factor = 1 + (g - 0.5) * 2 * scale
    out[k] = Math.max(0, +(out[k] * factor).toFixed(3))
  }
  if (out.passNomination > 0.95) out.passNomination = 0.95
  return [out, rng]
}
