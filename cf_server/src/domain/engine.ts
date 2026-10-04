/**
 * Pure Power Grid engine. All functions take a GameState and return a new one (or throw
 * GameError, caught at the `apply*` boundary into Effect). Wire shapes are produced by
 * `toPayload`. Randomness comes from the seeded RNG stored in state.
 */
import { Data, Effect } from "effect"
import type { PlantDef } from "./deck-default.ts"
import { GameErrors, RESOURCE_TYPES, type GamePhase, type GameStatePayload, type PlayerInfo, type PowerPlantInfo, type ResourceType } from "../protocol/game.ts"
import { buildGraph, connectionCost, toWireCities, toWireConnections, type MapData } from "./map.ts"
import * as Market from "./market.ts"
import * as Plants from "./plants.ts"
import { seedRng, shuffle, type Rng } from "./rng.ts"
import {
  CITY_SLOT_COST,
  DEFAULT_STEP2_TRIGGER,
  MAX_PLANTS_PER_PLAYER,
  MAX_PLAYERS,
  MIN_PLAYERS,
  PLAYER_COLORS,
  STARTING_MONEY,
  incomeFor,
  slotsOpenInStep,
  winCitiesFor
} from "./rules.ts"
import { currentPlayerId, emptyStore, playerCount, type GameState, type Plant, type PlayerState, type ResourceStore } from "./state.ts"

export class GameError extends Data.TaggedError("GameError")<{ readonly message: string }> {}
const fail = (message: string): never => {
  throw new GameError({ message })
}

export type Event =
  | { readonly _tag: "PhaseChange"; readonly phase: GamePhase; readonly round: number }
  | { readonly _tag: "TurnChange"; readonly playerId: string; readonly turn: number }
  | { readonly _tag: "StepChange"; readonly step: number }
  | { readonly _tag: "PlantBought"; readonly playerId: string; readonly plant: number; readonly price: number }
  | { readonly _tag: "GameEnd"; readonly winnerId: string | null }

export type Action =
  | { readonly _tag: "BidPlant"; readonly plantId: number; readonly bid: number }
  | { readonly _tag: "BuyResources"; readonly resources: Readonly<Record<string, number>> }
  | { readonly _tag: "BuildCity"; readonly cityId: string }
  | { readonly _tag: "PowerCities"; readonly plantIds: ReadonlyArray<number> }
  | { readonly _tag: "EndTurn" }

export interface Outcome {
  readonly state: GameState
  readonly events: ReadonlyArray<Event>
}

// --- helpers ------------------------------------------------------------------

const rngOf = (s: GameState): Rng => seedRng(s.seed)
const withRng = (s: GameState, rng: Rng): GameState => ({ ...s, seed: rng.state })

const marketState = (s: GameState): Plants.MarketState => ({
  current: s.currentMarket,
  future: s.futureMarket,
  deck: s.deck,
  step: s.step,
  step3Drawn: false,
  rng: rngOf(s)
})
const applyMarket = (s: GameState, m: Plants.MarketState): GameState => ({
  ...withRng(s, m.rng),
  currentMarket: m.current,
  futureMarket: m.future,
  deck: m.deck,
  step3Pending: s.step3Pending || m.step3Drawn
})

/** Acting order: resources and building run in reverse player order. */
export const actingOrder = (s: GameState): ReadonlyArray<string> =>
  s.phase === "BUY_RESOURCES" || s.phase === "BUILD_CITIES" ? [...s.turnOrder].reverse() : s.turnOrder

export const actingPlayerId = (s: GameState): string => actingOrder(s)[s.turnIndex] ?? ""

const updatePlayer = (s: GameState, id: string, f: (p: PlayerState) => PlayerState): GameState => {
  const p = s.players[id] ?? fail(GameErrors.PlayerNotFound)
  return { ...s, players: { ...s.players, [id]: f(p) } }
}

const requirePlayer = (s: GameState, id: string): PlayerState => s.players[id] ?? fail(GameErrors.PlayerNotFound)

const step2Trigger = (s: GameState): number => s.map.gameRules?.step2Trigger ?? DEFAULT_STEP2_TRIGGER

/** Player order: most cities first, tiebreak highest plant number. Round 1: random. */
const determineOrder = (s: GameState): GameState => {
  const ids = Object.keys(s.players)
  if (s.round === 1 && s.turnOrder.length === 0) {
    const [order, rng] = shuffle(rngOf(s), ids)
    return withRng({ ...s, turnOrder: order }, rng)
  }
  const maxPlant = (p: PlayerState) => p.plants.reduce((m, pl) => Math.max(m, pl.number), 0)
  const order = [...ids].sort((a, b) => {
    const pa = s.players[a]!
    const pb = s.players[b]!
    if (pb.cities.length !== pa.cities.length) return pb.cities.length - pa.cities.length
    if (maxPlant(pb) !== maxPlant(pa)) return maxPlant(pb) - maxPlant(pa)
    return s.turnOrder.indexOf(a) - s.turnOrder.indexOf(b)
  })
  return { ...s, turnOrder: order }
}

// --- lifecycle ------------------------------------------------------------------

export interface CreateOptions {
  readonly id: string
  readonly name: string
  readonly map: MapData
  readonly deck: ReadonlyArray<PlantDef>
  readonly seed: number
  readonly now: number
}

export const createGame = (o: CreateOptions): GameState => {
  const supply = o.map.gameRules?.resourceSupply
  const initial: Partial<Record<ResourceType, number>> = {}
  if (supply) {
    for (const t of RESOURCE_TYPES) {
      const v = supply[t.toLowerCase()]
      if (v !== undefined) initial[t] = v
    }
  }
  return {
    id: o.id,
    name: o.name,
    mapId: o.map.id,
    map: o.map,
    status: "LOBBY",
    phase: "PLAYER_ORDER",
    round: 1,
    step: 1,
    turnIndex: 0,
    turnOrder: [],
    players: {},
    seating: [],
    citySlots: {},
    market: Market.initialMarket(initial),
    currentMarket: [],
    futureMarket: [],
    deck: o.deck.map(Plants.toPlant),
    auction: null,
    endTriggered: false,
    step3Pending: false,
    auctionHadPurchase: false,
    winnerId: null,
    createdAt: o.now,
    updatedAt: o.now,
    seed: o.seed
  }
}

export const addPlayer = (s: GameState, p: { readonly id: string; readonly name: string; readonly color?: string | undefined }): GameState => {
  if (s.status !== "LOBBY") fail(GameErrors.AlreadyStarted)
  if (s.players[p.id]) return s
  if (playerCount(s) >= MAX_PLAYERS) fail(GameErrors.GameFull)
  const taken = new Set(Object.values(s.players).map((x) => x.color))
  let color = p.color
  if (color && taken.has(color)) fail(GameErrors.ColorTaken)
  if (!color) color = PLAYER_COLORS.find((c) => !taken.has(c)) ?? `color${playerCount(s)}`
  const player: PlayerState = {
    id: p.id,
    name: p.name,
    color,
    money: s.map.gameRules?.startingMoney ?? STARTING_MONEY,
    plants: [],
    resources: emptyStore(),
    cities: [],
    poweredCities: 0,
    doneThisPhase: false,
    poweredThisRound: false,
    connected: false
  }
  return { ...s, players: { ...s.players, [p.id]: player }, seating: [...s.seating, p.id] }
}

export const removePlayer = (s: GameState, id: string): GameState => {
  if (s.status !== "LOBBY") fail("cannot remove player after game has started")
  if (!s.players[id]) fail("player not in game")
  const { [id]: _removed, ...rest } = s.players
  return { ...s, players: rest, seating: s.seating.filter((x) => x !== id) }
}

export const setConnected = (s: GameState, id: string, connected: boolean): GameState =>
  s.players[id] ? updatePlayer(s, id, (p) => ({ ...p, connected })) : s

export const startGame = (s0: GameState, now: number): Outcome => {
  if (s0.status !== "LOBBY") fail(GameErrors.AlreadyStarted)
  if (playerCount(s0) < MIN_PLAYERS) fail(GameErrors.NeedTwoPlayers)
  const defs: Array<PlantDef> = s0.deck.filter((p): p is Plant => p !== null)
  const setup = Plants.setupDeck(defs, playerCount(s0), rngOf(s0))
  let s: GameState = {
    ...withRng(s0, setup.rng),
    status: "PLAYING",
    phase: "PLAYER_ORDER",
    round: 1,
    turnIndex: 0,
    currentMarket: setup.current,
    futureMarket: setup.future,
    deck: setup.deck,
    updatedAt: now
  }
  s = determineOrder(s)
  const events: Array<Event> = [{ _tag: "PhaseChange", phase: "PLAYER_ORDER", round: 1 }]
  return beginAuction(s, events)
}

// --- phase transitions -----------------------------------------------------------

const resetPhaseFlags = (s: GameState): GameState => ({
  ...s,
  players: Object.fromEntries(
    Object.entries(s.players).map(([id, p]) => [id, { ...p, doneThisPhase: false }])
  )
})

const turnEvent = (s: GameState): Event => ({ _tag: "TurnChange", playerId: actingPlayerId(s), turn: s.turnIndex })

const beginAuction = (s0: GameState, events: Array<Event>): Outcome => {
  const s = { ...resetPhaseFlags(s0), phase: "AUCTION" as const, turnIndex: 0, auction: null }
  events.push({ _tag: "PhaseChange", phase: "AUCTION", round: s.round }, turnEvent(s))
  return { state: s, events }
}

const beginBuyResources = (s0: GameState, events: Array<Event>): Outcome => {
  const s = { ...resetPhaseFlags(s0), phase: "BUY_RESOURCES" as const, turnIndex: 0, auction: null }
  events.push({ _tag: "PhaseChange", phase: "BUY_RESOURCES", round: s.round }, turnEvent(s))
  return { state: s, events }
}

const beginBuild = (s0: GameState, events: Array<Event>): Outcome => {
  const s = { ...resetPhaseFlags(s0), phase: "BUILD_CITIES" as const, turnIndex: 0 }
  events.push({ _tag: "PhaseChange", phase: "BUILD_CITIES", round: s.round }, turnEvent(s))
  return { state: s, events }
}

const beginBureaucracy = (s0: GameState, events: Array<Event>): Outcome => {
  let s: GameState = {
    ...resetPhaseFlags(s0),
    phase: "BUREAUCRACY",
    turnIndex: 0,
    players: Object.fromEntries(
      Object.entries(s0.players).map(([id, p]) => [id, { ...p, doneThisPhase: false, poweredThisRound: false }])
    )
  }
  // Step 2 check happens at the end of building.
  const maxCities = Math.max(0, ...Object.values(s.players).map((p) => p.cities.length))
  if (s.step === 1 && maxCities >= step2Trigger(s)) {
    s = applyMarket({ ...s, step: 2 }, Plants.removeLowest({ ...marketState(s), step: 2 }))
    events.push({ _tag: "StepChange", step: 2 })
  }
  if (maxCities >= winCitiesFor(playerCount(s), s.map.gameRules?.winConditions)) {
    s = { ...s, endTriggered: true }
  }
  events.push({ _tag: "PhaseChange", phase: "BUREAUCRACY", round: s.round }, turnEvent(s))
  return { state: s, events }
}

const applyPendingStep3 = (s: GameState, events: Array<Event>): GameState => {
  if (!s.step3Pending) return s
  const m = Plants.enterStep3({ ...marketState(s), step: 3 })
  events.push({ _tag: "StepChange", step: 3 })
  return { ...applyMarket({ ...s, step: 3, step3Pending: false }, m), step3Pending: false }
}

const finishBureaucracy = (s0: GameState, events: Array<Event>): Outcome => {
  let s = s0
  // Players who never powered get income for 0 cities.
  for (const p of Object.values(s.players)) {
    if (!p.poweredThisRound) {
      s = updatePlayer(s, p.id, (x) => ({ ...x, money: x.money + incomeFor(0), poweredCities: 0, poweredThisRound: true }))
    }
  }
  if (s.endTriggered) {
    const winner = pickWinner(s)
    s = { ...s, phase: "GAME_END", status: "FINISHED", winnerId: winner }
    events.push({ _tag: "PhaseChange", phase: "GAME_END", round: s.round }, { _tag: "GameEnd", winnerId: winner })
    return { state: s, events }
  }
  s = { ...s, market: Market.refill(s.market, s.step, playerCount(s)) }
  s = applyMarket(s, Plants.bureaucracyUpdate(marketState(s)))
  s = applyPendingStep3(s, events)
  s = determineOrder({ ...s, round: s.round + 1 })
  events.push({ _tag: "PhaseChange", phase: "PLAYER_ORDER", round: s.round })
  return beginAuction(s, events)
}

const pickWinner = (s: GameState): string | null => {
  const ranked = Object.values(s.players).sort(
    (a, b) => b.poweredCities - a.poweredCities || b.money - a.money || b.cities.length - a.cities.length
  )
  return ranked[0]?.id ?? null
}

/** Advance to the next acting player, or to the next phase when everyone has acted. */
const advanceTurn = (s0: GameState, events: Array<Event>): Outcome => {
  const order = actingOrder(s0)
  const next = s0.turnIndex + 1
  if (next < order.length) {
    const s = { ...s0, turnIndex: next }
    events.push(turnEvent(s))
    return { state: s, events }
  }
  switch (s0.phase) {
    case "BUY_RESOURCES":
      return beginBuild(s0, events)
    case "BUILD_CITIES":
      return beginBureaucracy(s0, events)
    case "BUREAUCRACY":
      return finishBureaucracy(s0, events)
    default:
      return { state: s0, events }
  }
}

// --- auction ------------------------------------------------------------------

const nextNominator = (s: GameState): number | undefined => {
  const order = s.turnOrder
  for (let i = 0; i < order.length; i++) {
    const idx = (s.turnIndex + i) % order.length
    const p = s.players[order[idx]!]!
    if (!p.doneThisPhase) return idx
  }
  return undefined
}

const afterNominationStep = (s0: GameState, events: Array<Event>, boughtThisPhase: boolean): Outcome => {
  const idx = nextNominator(s0)
  if (idx === undefined) {
    let s = s0
    if (!boughtThisPhase && !s.auctionHadPurchase) {
      // Nobody bought a plant this round: lowest plant leaves the market.
      s = applyMarket(s, Plants.removeLowest(marketState(s)))
    }
    s = applyPendingStep3({ ...s, auctionHadPurchase: false }, events)
    return beginBuyResources(s, events)
  }
  const s = { ...s0, turnIndex: idx, auction: null }
  events.push(turnEvent(s))
  return { state: s, events }
}

const addPlantToPlayer = (p: PlayerState, plant: Plant): PlayerState => {
  // Over the limit: no DISCARD message exists in the protocol, so drop the lowest-numbered
  // plant the player already owned (never the one just won).
  const existing = [...p.plants].sort(Plants.byNumber)
  const trimmed = existing.length >= MAX_PLANTS_PER_PLAYER ? existing.slice(1) : existing
  const plants = [...trimmed, plant].sort(Plants.byNumber)
  const kept = plants
  return { ...p, plants: kept, resources: clampResources(kept, p.resources) }
}

const resolveAuction = (s0: GameState, events: Array<Event>, winnerId: string): Outcome => {
  const a = s0.auction ?? fail(GameErrors.NoActiveAuction)
  const plant = [...s0.currentMarket, ...s0.futureMarket].find((p) => p.number === a.plantNumber) ?? fail(GameErrors.PlantNotInMarket)
  let s = updatePlayer(s0, winnerId, (p) => ({ ...addPlantToPlayer(p, plant), money: p.money - a.currentBid, doneThisPhase: true }))
  s = applyMarket(s, Plants.takeFromMarket(marketState(s), plant.number))
  s = { ...s, auction: null, auctionHadPurchase: true }
  events.push({ _tag: "PlantBought", playerId: winnerId, plant: plant.number, price: a.currentBid })
  // Nomination continues with the next player after the nominator who hasn't bought.
  const nomIdx = s.turnOrder.indexOf(a.nominatedBy)
  return afterNominationStep({ ...s, turnIndex: (nomIdx + 1) % s.turnOrder.length }, events, true)
}

const bidPlant = (s: GameState, playerId: string, plantId: number, bid: number): Outcome => {
  if (s.phase !== "AUCTION") fail(GameErrors.NotAuctionPhase)
  const events: Array<Event> = []
  const player = requirePlayer(s, playerId)

  if (s.auction === null) {
    if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
    if (player.doneThisPhase) fail(GameErrors.AlreadyBoughtPlant)
    if (bid <= 0) {
      if (s.round === 1) fail(GameErrors.MustBuyFirstRound)
      const passed = updatePlayer(s, playerId, (p) => ({ ...p, doneThisPhase: true }))
      return afterNominationStep({ ...passed, turnIndex: (passed.turnIndex + 1) % passed.turnOrder.length }, events, false)
    }
    const plant = [...s.currentMarket, ...s.futureMarket].find((p) => p.number === plantId) ?? fail(GameErrors.PlantNotInMarket)
    if (!s.currentMarket.some((p) => p.number === plantId)) fail(GameErrors.NotInCurrentMarket)
    if (bid < plant.number) fail(GameErrors.BidTooLow)
    if (bid > player.money) fail(GameErrors.BidNoMoney)
    const eligible = s.turnOrder.filter((id) => !s.players[id]!.doneThisPhase)
    const start = eligible.indexOf(playerId)
    const participants = [...eligible.slice(start), ...eligible.slice(0, start)]
    const opened: GameState = {
      ...s,
      auction: {
        plantNumber: plantId,
        nominatedBy: playerId,
        currentBid: bid,
        currentBidder: participants[1] ?? playerId,
        participants
      }
    }
    if (participants.length === 1) return resolveAuction(opened, events, playerId)
    return { state: opened, events }
  }

  const a = s.auction
  if (a.plantNumber !== plantId) fail(GameErrors.WrongPlantAuctioned)
  if (a.currentBidder !== playerId) fail(GameErrors.NotYourBid)
  if (bid <= 0) {
    const participants = a.participants.filter((id) => id !== playerId)
    const highBidder = a.participants[(a.participants.indexOf(playerId) - 1 + a.participants.length) % a.participants.length]!
    if (participants.length === 1) return resolveAuction({ ...s, auction: { ...a, participants } }, events, participants[0]!)
    const nextIdx = (a.participants.indexOf(playerId) + 1) % a.participants.length
    const nextBidder = a.participants[nextIdx]!
    void highBidder
    return { state: { ...s, auction: { ...a, participants, currentBidder: nextBidder } }, events }
  }
  if (bid <= a.currentBid) fail(GameErrors.BidTooLow)
  if (bid > player.money) fail(GameErrors.BidNoMoney)
  const idx = a.participants.indexOf(playerId)
  const nextBidder = a.participants[(idx + 1) % a.participants.length]!
  return { state: { ...s, auction: { ...a, currentBid: bid, currentBidder: nextBidder } }, events }
}

// --- resources ----------------------------------------------------------------

const storageCapacity = (plants: ReadonlyArray<Plant>): { own: Record<ResourceType, number>; hybrid: number } => {
  const own: Record<ResourceType, number> = { Coal: 0, Oil: 0, Garbage: 0, Uranium: 0 }
  let hybrid = 0
  for (const p of plants) {
    if (p.resourceType === "Hybrid") hybrid += 2 * p.resourceCost
    else if (p.resourceType !== "Wind") own[p.resourceType] += 2 * p.resourceCost
  }
  return { own, hybrid }
}

const fitsStorage = (plants: ReadonlyArray<Plant>, store: ResourceStore): boolean => {
  const cap = storageCapacity(plants)
  if (store.Garbage > cap.own.Garbage || store.Uranium > cap.own.Uranium) return false
  const overflow = Math.max(0, store.Coal - cap.own.Coal) + Math.max(0, store.Oil - cap.own.Oil)
  return overflow <= cap.hybrid
}

/** After losing a plant, drop resources that no longer fit (standard rules let the player choose; we trim). */
const clampResources = (plants: ReadonlyArray<Plant>, store: ResourceStore): ResourceStore => {
  const cap = storageCapacity(plants)
  const next = { ...store }
  next.Garbage = Math.min(next.Garbage, cap.own.Garbage)
  next.Uranium = Math.min(next.Uranium, cap.own.Uranium)
  let hybridLeft = cap.hybrid
  const coalOver = Math.max(0, next.Coal - cap.own.Coal)
  const useCoal = Math.min(coalOver, hybridLeft)
  next.Coal = Math.min(next.Coal, cap.own.Coal + useCoal)
  hybridLeft -= useCoal
  const oilOver = Math.max(0, next.Oil - cap.own.Oil)
  next.Oil = Math.min(next.Oil, cap.own.Oil + Math.min(oilOver, hybridLeft))
  return next
}

const buyResources = (s: GameState, playerId: string, resources: Readonly<Record<string, number>>): Outcome => {
  if (s.phase !== "BUY_RESOURCES") fail(GameErrors.NotBuyPhase)
  if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
  const player = requirePlayer(s, playerId)
  let total = 0
  let market = s.market
  const store = { ...player.resources }
  for (const [key, amount] of Object.entries(resources)) {
    if (!(RESOURCE_TYPES as ReadonlyArray<string>).includes(key)) fail(GameErrors.ResourceNotFound)
    const type = key as ResourceType
    if (amount <= 0) continue
    const cost = Market.purchaseCost(market, type, amount) ?? fail(GameErrors.NotEnoughResources)
    total += cost
    market = Market.removeTokens(market, type, amount)
    store[type] += amount
  }
  if (total > player.money) fail(GameErrors.NotEnoughMoney)
  if (!fitsStorage(player.plants, store)) fail(GameErrors.NoStorage)
  const next = updatePlayer({ ...s, market }, playerId, (p) => ({ ...p, money: p.money - total, resources: store }))
  return { state: next, events: [] }
}

// --- building -----------------------------------------------------------------

const buildCity = (s: GameState, playerId: string, cityId: string): Outcome => {
  if (s.phase !== "BUILD_CITIES") fail(GameErrors.NotBuildPhase)
  if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
  const player = requirePlayer(s, playerId)
  if (!s.map.cities.some((c) => c.id === cityId)) fail(GameErrors.CityNotFound)
  const slots = s.citySlots[cityId] ?? []
  if (slots.includes(playerId)) fail(GameErrors.AlreadyInCity)
  if (slots.length >= CITY_SLOT_COST.length) fail(GameErrors.CityFull)
  if (slots.length >= slotsOpenInStep(s.step)) fail(GameErrors.CityClosedThisStep)
  const graph = buildGraph(s.map)
  const conn = connectionCost(graph, player.cities, cityId) ?? fail(GameErrors.NotConnected)
  const total = conn + CITY_SLOT_COST[slots.length]!
  if (total > player.money) fail(GameErrors.NotEnoughMoney)
  const next: GameState = {
    ...updatePlayer(s, playerId, (p) => ({ ...p, money: p.money - total, cities: [...p.cities, cityId] })),
    citySlots: { ...s.citySlots, [cityId]: [...slots, playerId] }
  }
  return { state: next, events: [] }
}

// --- bureaucracy ----------------------------------------------------------------

const powerCities = (s: GameState, playerId: string, plantIds: ReadonlyArray<number>): Outcome => {
  if (s.phase !== "BUREAUCRACY") fail(GameErrors.NotBureaucracyPhase)
  if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
  const player = requirePlayer(s, playerId)
  if (player.poweredThisRound) fail(GameErrors.AlreadyPowered)
  const store = { ...player.resources }
  const used: Partial<Record<ResourceType, number>> = {}
  let capacity = 0
  const seen = new Set<number>()
  for (const id of plantIds) {
    if (seen.has(id)) continue
    seen.add(id)
    const plant = player.plants.find((p) => p.number === id) ?? fail(GameErrors.PlantNotOwned)
    if (plant.resourceType === "Wind") {
      capacity += plant.capacity
      continue
    }
    if (plant.resourceType === "Hybrid") {
      let need = plant.resourceCost
      const coal = Math.min(store.Coal, need)
      store.Coal -= coal
      need -= coal
      const oil = Math.min(store.Oil, need)
      store.Oil -= oil
      need -= oil
      if (need > 0) fail(GameErrors.CannotPower)
      used.Coal = (used.Coal ?? 0) + coal
      used.Oil = (used.Oil ?? 0) + oil
    } else {
      const type = plant.resourceType
      if (store[type] < plant.resourceCost) fail(GameErrors.CannotPower)
      store[type] -= plant.resourceCost
      used[type] = (used[type] ?? 0) + plant.resourceCost
    }
    capacity += plant.capacity
  }
  const powered = Math.min(capacity, player.cities.length)
  const income = incomeFor(powered)
  const next: GameState = {
    ...updatePlayer(s, playerId, (p) => ({
      ...p,
      resources: store,
      money: p.money + income,
      poweredCities: powered,
      poweredThisRound: true
    })),
    market: Market.returnToBank(s.market, used)
  }
  return { state: next, events: [] }
}

// --- end turn -------------------------------------------------------------------

const endTurn = (s: GameState, playerId: string): Outcome => {
  const events: Array<Event> = []
  switch (s.phase) {
    case "AUCTION": {
      if (s.auction) {
        if (s.auction.currentBidder !== playerId) fail(GameErrors.NotYourBid)
        return bidPlant(s, playerId, s.auction.plantNumber, 0)
      }
      if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
      return bidPlant(s, playerId, 0, 0)
    }
    case "BUREAUCRACY": {
      if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
      const p = requirePlayer(s, playerId)
      const withIncome = p.poweredThisRound ? s : powerCities(s, playerId, []).state
      return advanceTurn(withIncome, events)
    }
    case "BUY_RESOURCES":
    case "BUILD_CITIES": {
      if (actingPlayerId(s) !== playerId) fail(GameErrors.NotYourTurn)
      return advanceTurn(s, events)
    }
    case "PLAYER_ORDER":
    case "GAME_END":
      return { state: s, events }
  }
}

// --- public boundary ----------------------------------------------------------------

const applyActionSync = (s: GameState, playerId: string, action: Action, now: number): Outcome => {
  if (s.status !== "PLAYING") fail(GameErrors.NotPlaying)
  requirePlayer(s, playerId)
  const out = (() => {
    switch (action._tag) {
      case "BidPlant":
        return bidPlant(s, playerId, action.plantId, action.bid)
      case "BuyResources":
        return buyResources(s, playerId, action.resources)
      case "BuildCity":
        return buildCity(s, playerId, action.cityId)
      case "PowerCities":
        return powerCities(s, playerId, action.plantIds)
      case "EndTurn":
        return endTurn(s, playerId)
    }
  })()
  return { state: { ...out.state, updatedAt: now }, events: out.events }
}

const lift =
  <Args extends ReadonlyArray<unknown>, A>(f: (...args: Args) => A) =>
  (...args: Args): Effect.Effect<A, GameError> =>
    Effect.try({
      try: () => f(...args),
      catch: (e) => (e instanceof GameError ? e : new GameError({ message: e instanceof Error ? e.message : String(e) }))
    })

export const applyAction = lift(applyActionSync)
export const start = lift(startGame)
export const join = lift(addPlayer)
export const leave = lift(removePlayer)

// --- wire -------------------------------------------------------------------------

export const plantInfo = (p: Plant): PowerPlantInfo => ({
  id: p.number,
  cost: p.number,
  capacity: p.capacity,
  resource_type: p.resourceType,
  resource_cost: p.resourceCost
})

const playerInfo = (p: PlayerState): PlayerInfo => ({
  id: p.id,
  name: p.name,
  color: p.color,
  money: p.money,
  cities: [...p.cities],
  power_plants: p.plants.length === 0 ? null : p.plants.map(plantInfo),
  resources: { ...p.resources },
  powered_cities: p.poweredCities,
  connected: p.connected,
  has_passed: p.doneThisPhase
})

export const toPayload = (s: GameState): GameStatePayload => {
  const marketPlants = [...s.currentMarket, ...s.futureMarket]
  return {
    game_id: s.id,
    name: s.name,
    status: s.status,
    current_phase: s.phase,
    current_turn: s.auction ? s.auction.currentBidder : s.status === "PLAYING" ? actingPlayerId(s) : currentPlayerId(s),
    current_round: s.round,
    players: Object.fromEntries(Object.values(s.players).map((p) => [p.id, playerInfo(p)])),
    map: {
      name: s.map.name,
      cities: toWireCities(s.map, s.citySlots),
      connections: toWireConnections(s.map)
    },
    market: { resources: Market.toWire(s.market) },
    power_plants: marketPlants.length === 0 ? null : marketPlants.map(plantInfo),
    turn_order: [...s.turnOrder],
    step: s.step,
    auction: s.auction
      ? {
          plant_id: s.auction.plantNumber,
          current_bid: s.auction.currentBid,
          current_bidder: s.auction.currentBidder,
          nominated_by: s.auction.nominatedBy,
          participants: [...s.auction.participants],
          passed: s.turnOrder.filter((id) => !s.auction!.participants.includes(id))
        }
      : null,
    plant_market: {
      current: s.currentMarket.map(plantInfo),
      future: s.futureMarket.map(plantInfo),
      deck_remaining: s.deck.length
    },
    winner_id: s.winnerId,
    map_id: s.mapId
  }
}
