import { Effect, Exit } from "effect"
import { describe, expect, it } from "vitest"
import { DEFAULT_DECK } from "../../src/domain/deck-default.ts"
import * as Engine from "../../src/domain/engine.ts"
import { decodeMapData } from "../../src/domain/map.ts"
import { actingPlayerId } from "../../src/domain/engine.ts"
import type { GameState } from "../../src/domain/state.ts"

import usaJson from "../../../maps/usa.json"

const usa = decodeMapData(usaJson)

const run = <A>(eff: Effect.Effect<A, Engine.GameError>): A => Effect.runSync(eff)
const expectFail = <A>(eff: Effect.Effect<A, Engine.GameError>, message: string) => {
  const exit = Effect.runSyncExit(eff)
  expect(Exit.isFailure(exit)).toBe(true)
  if (Exit.isFailure(exit)) {
    const msg = String(exit.cause).includes(message) || JSON.stringify(exit.cause).includes(message)
    expect(msg, `expected failure "${message}" but got ${String(exit.cause)}`).toBe(true)
  }
}

const newGame = (players = 3, seed = 42): GameState => {
  let s = Engine.createGame({ id: "g1", name: "Test", map: usa, deck: DEFAULT_DECK, seed, now: 1000 })
  for (let i = 0; i < players; i++) s = run(Engine.join(s, { id: `p${i + 1}`, name: `Player ${i + 1}` }))
  return s
}

const started = (players = 3, seed = 42) => run(Engine.start(newGame(players, seed), 2000))

const act = (s: GameState, playerId: string, action: Engine.Action) => run(Engine.applyAction(s, playerId, action, 3000))

/** Everyone buys the cheapest available plant at face value in turn order. */
const runAuctionRound = (s0: GameState): GameState => {
  let s = s0
  while (s.phase === "AUCTION") {
    const pid = s.auction ? s.auction.currentBidder : actingPlayerId(s)
    if (s.auction) {
      s = act(s, pid, { _tag: "BidPlant", plantId: s.auction.plantNumber, bid: 0 }).state
    } else {
      const plant = s.currentMarket[0]!
      s = act(s, pid, { _tag: "BidPlant", plantId: plant.number, bid: plant.number }).state
    }
  }
  return s
}

describe("setup", () => {
  it("creates a game with the standard initial resource market", () => {
    const s = newGame()
    expect(s.market.slots.Coal.slice(1)).toEqual([3, 3, 3, 3, 3, 3, 3, 3])
    expect(s.market.slots.Oil.slice(1)).toEqual([0, 0, 3, 3, 3, 3, 3, 3])
    expect(s.market.slots.Garbage.slice(1)).toEqual([0, 0, 0, 0, 0, 0, 3, 3])
    expect(s.market.slots.Uranium.filter((n) => n > 0)).toEqual([1, 1])
    expect(s.market.slots.Uranium[14]).toBe(1)
    expect(s.market.slots.Uranium[16]).toBe(1)
    expect(s.market.slots.Coal).toHaveLength(9)
    expect(s.market.slots.Uranium).toHaveLength(17)
  })

  it("assigns distinct colors and rejects a 7th player", () => {
    let s = newGame(6)
    expect(new Set(Object.values(s.players).map((p) => p.color)).size).toBe(6)
    expectFail(Engine.join(s, { id: "p7", name: "Seven" }), "game is full")
    s = run(Engine.leave(s, "p6"))
    expect(Object.keys(s.players)).toHaveLength(5)
  })

  it("needs two players and lays out the plant market 3-6 / 7-10", () => {
    expectFail(Engine.start(newGame(1), 0), "need at least 2 players")
    const { state, events } = started(3)
    expect(state.status).toBe("PLAYING")
    expect(state.phase).toBe("AUCTION")
    expect(state.currentMarket.map((p) => p.number)).toEqual([3, 4, 5, 6])
    expect(state.futureMarket.map((p) => p.number)).toEqual([7, 8, 9, 10])
    expect(state.deck[0]?.number).toBe(13)
    expect(state.deck[state.deck.length - 1]).toBeNull()
    // 42 plants - 8 market - 13 - 8 removed (3 players) + step3 card
    expect(state.deck).toHaveLength(42 - 8 - 8 + 1)
    expect(state.turnOrder).toHaveLength(3)
    expect(events.map((e) => e._tag)).toContain("PhaseChange")
  })

  it("is deterministic for a given seed", () => {
    const a = started(4, 7).state
    const b = started(4, 7).state
    expect(a.turnOrder).toEqual(b.turnOrder)
    expect(a.deck.map((p) => p?.number ?? "S3")).toEqual(b.deck.map((p) => p?.number ?? "S3"))
  })
})

describe("auction", () => {
  it("rejects bids out of turn, below face value, and in the future market", () => {
    const s = started().state
    const first = actingPlayerId(s)
    const other = s.turnOrder.find((id) => id !== first)!
    expectFail(Engine.applyAction(s, other, { _tag: "BidPlant", plantId: 3, bid: 3 }, 0), "not your turn")
    expectFail(Engine.applyAction(s, first, { _tag: "BidPlant", plantId: 3, bid: 2 }, 0), "higher than current bid")
    expectFail(Engine.applyAction(s, first, { _tag: "BidPlant", plantId: 7, bid: 7 }, 0), "future market")
    expectFail(Engine.applyAction(s, first, { _tag: "BidPlant", plantId: 99, bid: 99 }, 0), "plant not found in market")
    expectFail(Engine.applyAction(s, first, { _tag: "BidPlant", plantId: 0, bid: 0 }, 0), "first round")
  })

  it("runs a bidding war and gives the plant to the high bidder", () => {
    const s0 = started(3).state
    const [a, b, c] = s0.turnOrder as [string, string, string]
    let s = act(s0, a, { _tag: "BidPlant", plantId: 5, bid: 5 }).state
    expect(s.auction?.currentBidder).toBe(b)
    s = act(s, b, { _tag: "BidPlant", plantId: 5, bid: 8 }).state
    expect(s.auction?.currentBidder).toBe(c)
    s = act(s, c, { _tag: "BidPlant", plantId: 5, bid: 0 }).state // c passes
    expect(s.auction?.currentBidder).toBe(a)
    const out = act(s, a, { _tag: "BidPlant", plantId: 5, bid: 0 }) // a passes -> b wins at 8
    s = out.state
    expect(s.auction).toBeNull()
    expect(s.players[b]!.plants.map((p) => p.number)).toEqual([5])
    expect(s.players[b]!.money).toBe(50 - 8)
    expect(s.currentMarket.map((p) => p.number)).toEqual([3, 4, 6, 7])
    expect(s.futureMarket.map((p) => p.number)).toEqual([8, 9, 10, 13])
    expect(out.events.some((e) => e._tag === "PlantBought")).toBe(true)
    // nomination continues with the next player after the nominator (a) who hasn't bought: b bought, so c
    expect(actingPlayerId(s)).toBe(c)
  })

  it("moves to buy resources in reverse order once everyone has a plant", () => {
    const s = runAuctionRound(started(3).state)
    expect(s.phase).toBe("BUY_RESOURCES")
    expect(Object.values(s.players).every((p) => p.plants.length === 1)).toBe(true)
    expect(actingPlayerId(s)).toBe(s.turnOrder[s.turnOrder.length - 1])
  })

  it("auto-discards the lowest existing plant when a 4th is won", () => {
    const s0 = started(2, 3).state
    const [a, b] = s0.turnOrder as [string, string]
    const loaded: GameState = {
      ...s0,
      players: {
        ...s0.players,
        [a]: { ...s0.players[a]!, plants: [20, 30, 40].map((n) => ({ number: n, capacity: 5, resourceType: "Coal" as const, resourceCost: 2 })) }
      }
    }
    let s = act(loaded, a, { _tag: "BidPlant", plantId: 3, bid: 3 }).state
    s = act(s, b, { _tag: "BidPlant", plantId: 3, bid: 0 }).state
    expect(s.players[a]!.plants.map((p) => p.number)).toEqual([3, 30, 40])
  })
})

describe("resources and building", () => {
  const toBuild = () => {
    let s = runAuctionRound(started(3, 42).state)
    // Everyone skips buying resources
    for (let i = 0; i < 3; i++) s = act(s, actingPlayerId(s), { _tag: "EndTurn" }).state
    return s
  }

  it("enforces storage capacity and money when buying", () => {
    const s = runAuctionRound(started(3, 42).state)
    const pid = actingPlayerId(s)
    const plant = s.players[pid]!.plants[0]!
    expectFail(Engine.applyAction(s, pid, { _tag: "BuyResources", resources: { Wood: 1 } }, 0), "resource type not found")
    expectFail(Engine.applyAction(s, pid, { _tag: "BuyResources", resources: { Uranium: 5 } }, 0), "not enough resources available")
    if (plant.resourceType === "Coal") {
      const ok = act(s, pid, { _tag: "BuyResources", resources: { Coal: 2 * plant.resourceCost } }).state
      expect(ok.players[pid]!.resources.Coal).toBe(2 * plant.resourceCost)
      expect(ok.players[pid]!.money).toBeLessThan(s.players[pid]!.money)
      expectFail(Engine.applyAction(s, pid, { _tag: "BuyResources", resources: { Coal: 2 * plant.resourceCost + 1 } }, 0), "storage capacity")
    }
  })

  it("charges slot + connection cost and respects step-1 single slot", () => {
    const s = toBuild()
    expect(s.phase).toBe("BUILD_CITIES")
    const pid = actingPlayerId(s)
    const money = s.players[pid]!.money
    let n = act(s, pid, { _tag: "BuildCity", cityId: "seattle" }).state
    expect(n.players[pid]!.money).toBe(money - 10)
    expect(n.citySlots["seattle"]).toEqual([pid])
    // portland is adjacent to seattle in usa.json
    const conn = usa.connections.find((c) => (c.from === "seattle" && c.to === "portland") || (c.from === "portland" && c.to === "seattle"))!
    n = act(n, pid, { _tag: "BuildCity", cityId: "portland" }).state
    expect(n.players[pid]!.money).toBe(money - 10 - 10 - conn.cost)
    expectFail(Engine.applyAction(n, pid, { _tag: "BuildCity", cityId: "seattle" }, 0), "already has a house")
    // another player cannot take a second slot in step 1
    n = act(n, pid, { _tag: "EndTurn" }).state
    const other = actingPlayerId(n)
    expectFail(Engine.applyAction(n, other, { _tag: "BuildCity", cityId: "seattle" }, 0), "no open slot")
    expectFail(Engine.applyAction(n, other, { _tag: "BuildCity", cityId: "nowhere" }, 0), "city not found")
  })
})

describe("bureaucracy and rounds", () => {
  it("pays income, refills the market, and starts round 2 with most-cities-first order", () => {
    let s = runAuctionRound(started(3, 42).state)
    for (let i = 0; i < 3; i++) s = act(s, actingPlayerId(s), { _tag: "EndTurn" }).state // skip resources
    const builder = actingPlayerId(s)
    s = act(s, builder, { _tag: "BuildCity", cityId: "seattle" }).state
    for (let i = 0; i < 3; i++) s = act(s, actingPlayerId(s), { _tag: "EndTurn" }).state // end building
    expect(s.phase).toBe("BUREAUCRACY")
    const moneyBefore = Object.fromEntries(Object.values(s.players).map((p) => [p.id, p.money]))
    let out: Engine.Outcome = { state: s, events: [] }
    for (let i = 0; i < 3; i++) out = act(out.state, actingPlayerId(out.state), { _tag: "EndTurn" })
    s = out.state
    expect(s.round).toBe(2)
    expect(s.phase).toBe("AUCTION")
    for (const p of Object.values(s.players)) expect(p.money).toBe(moneyBefore[p.id]! + 10)
    expect(s.turnOrder[0]).toBe(builder)
    expect(out.events.some((e) => e._tag === "PhaseChange" && e.phase === "PLAYER_ORDER")).toBe(true)
    // bureaucracy market update: highest future plant went under the deck
    expect(s.futureMarket.length).toBe(4)
    expect(s.market.bank.Coal).toBeLessThan(24)
  })

  it("cannot power twice and rejects unowned plants", () => {
    let s = runAuctionRound(started(2, 5).state)
    for (let i = 0; i < 2; i++) s = act(s, actingPlayerId(s), { _tag: "EndTurn" }).state
    for (let i = 0; i < 2; i++) s = act(s, actingPlayerId(s), { _tag: "EndTurn" }).state
    const pid = actingPlayerId(s)
    expectFail(Engine.applyAction(s, pid, { _tag: "PowerCities", plantIds: [50] }, 0), "does not own")
    s = act(s, pid, { _tag: "PowerCities", plantIds: [] }).state
    expectFail(Engine.applyAction(s, pid, { _tag: "PowerCities", plantIds: [] }, 0), "already powered")
  })
})

describe("wire payload", () => {
  it("matches the Go GameStatePayload field names", () => {
    const s = started(3).state
    const p = Engine.toPayload(s)
    expect(Object.keys(p)).toEqual(
      expect.arrayContaining(["game_id", "name", "status", "current_phase", "current_turn", "current_round", "players", "map", "market", "power_plants", "turn_order"])
    )
    const player = Object.values(p.players)[0]!
    expect(player.power_plants).toBeNull()
    expect(Object.keys(player)).toEqual(expect.arrayContaining(["id", "name", "color", "money", "cities", "power_plants", "resources", "powered_cities"]))
    expect(p.market.resources["Coal"]).toHaveLength(9)
    expect(p.market.resources["Uranium"]).toHaveLength(17)
    expect(p.power_plants).toHaveLength(8)
    expect(p.map.connections[0]).toEqual(expect.objectContaining({ city_a: expect.any(String), city_b: expect.any(String), cost: expect.any(Number) }))
    expect(Object.values(p.map.cities)[0]!.position).toHaveLength(2)
    expect(p.current_turn).toBe(actingPlayerId(s))
  })
})
