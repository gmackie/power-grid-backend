/** Runs complete games in-process against the pure engine and collects board metrics. */
import { Effect, Exit } from "effect"
import type { PlantDef } from "../domain/deck-default.ts"
import * as Engine from "../domain/engine.ts"
import type { MapData } from "../domain/map.ts"
import { seedRng, type Rng } from "../domain/rng.ts"
import type { GameState } from "../domain/state.ts"
import { RESOURCE_TYPES } from "../protocol/game.ts"
import type { Bot } from "./bots.ts"

export interface GameResult {
  readonly map: string
  readonly seed: number
  readonly players: number
  readonly rounds: number
  readonly finished: boolean
  readonly winner: string | null
  readonly winnerBot: string | null
  readonly finalStep: number
  readonly leadChanges: number
  /** cities of winner minus runner-up */
  readonly margin: number
  readonly finalCities: Record<string, number>
  readonly finalMoney: Record<string, number>
  readonly powered: Record<string, number>
  readonly regionUsage: Record<string, number>
  readonly contestedCities: number
  readonly auctionPremium: number
  readonly auctions: number
  readonly scarcityRounds: Record<string, number>
  readonly botOf: Record<string, string>
  readonly errors: number
  readonly rejections: Record<string, number>
}

export interface RunOptions {
  readonly map: MapData
  readonly deck: ReadonlyArray<PlantDef>
  readonly bots: ReadonlyArray<Bot>
  readonly seed: number
  readonly maxRounds?: number
}

const act = (s: GameState, pid: string, a: Engine.Action, rejections?: Record<string, number>): GameState | undefined => {
  const exit = Effect.runSyncExit(Engine.applyAction(s, pid, a, 0))
  if (Exit.isSuccess(exit)) return exit.value.state
  if (rejections) {
    const msg = String(exit.cause).match(/GameError: ([^"\n\]\)]+)/)?.[1] ?? "unknown"
    const key = `${a._tag}: ${msg}`
    rejections[key] = (rejections[key] ?? 0) + 1
  }
  return undefined
}

export const runGame = (o: RunOptions): GameResult => {
  const maxRounds = o.maxRounds ?? 60
  let s = Engine.createGame({ id: `sim-${o.seed}`, name: "sim", map: o.map, deck: o.deck, seed: o.seed, now: 0 })
  const botOf: Record<string, string> = {}
  o.bots.forEach((b, i) => {
    const id = `p${i + 1}`
    s = Effect.runSync(Engine.join(s, { id, name: b.name }))
    botOf[id] = b.name
  })
  s = Effect.runSync(Engine.start(s, 0)).state
  let rng: Rng = seedRng(o.seed ^ 0x9e3779b9)
  let leader: string | null = null
  let leadChanges = 0
  let premium = 0
  let auctions = 0
  let errors = 0
  const rejections: Record<string, number> = {}
  const scarcity: Record<string, number> = { Coal: 0, Oil: 0, Garbage: 0, Uranium: 0 }
  let lastRound = 0
  let stepsSameRound = 0
  while (s.status === "PLAYING" && s.round <= maxRounds) {
    if (s.round !== lastRound) {
      lastRound = s.round
      stepsSameRound = 0
      for (const t of RESOURCE_TYPES) if (s.market.slots[t].every((n) => n === 0)) scarcity[t]!++
      const ranked = Object.values(s.players).sort((a, b) => b.cities.length - a.cities.length)
      const top = ranked[0]!
      if (top.cities.length > 0 && ranked[1] && top.cities.length > ranked[1].cities.length && leader !== top.id) { if (leader !== null) leadChanges++; leader = top.id }
    }
    if (++stepsSameRound > 2000) { errors++; break } // safety: stuck
    const pid = s.auction ? s.auction.currentBidder : Engine.actingPlayerId(s)
    const bot = o.bots[Number(pid.slice(1)) - 1]!
    const [action, r] = bot.decide(s, pid, rng)
    rng = r
    const before = s.auction
    let next = act(s, pid, action, rejections)
    if (!next) {
      errors++
      next = act(s, pid, s.auction ? { _tag: "BidPlant", plantId: s.auction.plantNumber, bid: 0 } : { _tag: "EndTurn" })
      if (!next) {
        // round-1 forced purchase or similar: buy the cheapest plant at face value
        const cheapest = s.currentMarket[0]
        next = cheapest ? act(s, pid, { _tag: "BidPlant", plantId: cheapest.number, bid: cheapest.number }) : undefined
        if (!next) break
      }
    }
    if (before && !next.auction) { auctions++; premium += before.currentBid - before.plantNumber }
    s = next
  }
  const players = Object.values(s.players)
  const ranked = [...players].sort((a, b) => b.cities.length - a.cities.length)
  const regionUsage: Record<string, number> = {}
  let contested = 0
  for (const [cityId, slots] of Object.entries(s.citySlots)) {
    const region = s.map.cities.find((c) => c.id === cityId)?.region ?? "?"
    regionUsage[region] = (regionUsage[region] ?? 0) + slots.length
    if (slots.length >= 3) contested++
  }
  return {
    map: o.map.id,
    seed: o.seed,
    players: players.length,
    rounds: s.round,
    finished: s.status === "FINISHED",
    winner: s.winnerId,
    winnerBot: s.winnerId ? botOf[s.winnerId]! : null,
    finalStep: s.step,
    leadChanges,
    margin: (ranked[0]?.cities.length ?? 0) - (ranked[1]?.cities.length ?? 0),
    finalCities: Object.fromEntries(players.map((p) => [p.id, p.cities.length])),
    finalMoney: Object.fromEntries(players.map((p) => [p.id, p.money])),
    powered: Object.fromEntries(players.map((p) => [p.id, p.poweredCities])),
    regionUsage,
    contestedCities: contested,
    auctionPremium: auctions ? premium / auctions : 0,
    auctions,
    scarcityRounds: scarcity,
    botOf,
    errors,
    rejections
  }
}

export interface BoardSummary {
  readonly map: string
  readonly games: number
  readonly finishedRate: number
  readonly avgRounds: number
  readonly avgLeadChanges: number
  readonly avgMargin: number
  readonly avgContested: number
  readonly avgPremium: number
  readonly step3Rate: number
  readonly winRates: Record<string, number>
  readonly regionShare: Record<string, number>
  readonly scarcity: Record<string, number>
  /** 0..1 composite: balance across archetypes, closeness, lead changes, region spread. */
  readonly interest: number
  readonly errors: number
  readonly topRejections: Array<[string, number]>
}

const mean = (xs: ReadonlyArray<number>) => (xs.length ? xs.reduce((a, b) => a + b, 0) / xs.length : 0)

export const summarize = (results: ReadonlyArray<GameResult>): BoardSummary => {
  const bots = new Set(results.flatMap((r) => Object.values(r.botOf)))
  const seats: Record<string, number> = {}
  const wins: Record<string, number> = {}
  for (const r of results) {
    for (const b of Object.values(r.botOf)) seats[b] = (seats[b] ?? 0) + 1
    if (r.winnerBot) wins[r.winnerBot] = (wins[r.winnerBot] ?? 0) + 1
  }
  const winRates = Object.fromEntries([...bots].map((b) => [b, seats[b] ? (wins[b] ?? 0) / seats[b] : 0]))
  const regionTotals: Record<string, number> = {}
  for (const r of results) for (const [k, v] of Object.entries(r.regionUsage)) regionTotals[k] = (regionTotals[k] ?? 0) + v
  const regionSum = Object.values(regionTotals).reduce((a, b) => a + b, 0) || 1
  const regionShare = Object.fromEntries(Object.entries(regionTotals).map(([k, v]) => [k, +(v / regionSum).toFixed(3)]))
  const scarcity = Object.fromEntries(RESOURCE_TYPES.map((t) => [t, +mean(results.map((r) => r.scarcityRounds[t] ?? 0)).toFixed(2)]))
  // interest: balanced win rates (low variance), small margins, many lead changes, even region use
  const wr = Object.values(winRates)
  const wrVar = mean(wr.map((x) => (x - mean(wr)) ** 2))
  const balance = Math.max(0, 1 - Math.sqrt(wrVar) * 3)
  const closeness = Math.max(0, 1 - mean(results.map((r) => r.margin)) / 6)
  const drama = Math.min(1, mean(results.map((r) => r.leadChanges)) / 4)
  const rs = Object.values(regionShare)
  const spread = rs.length ? Math.max(0, 1 - Math.sqrt(mean(rs.map((x) => (x - 1 / rs.length) ** 2))) * rs.length) : 0
  const finishedRate = mean(results.map((r) => (r.finished ? 1 : 0)))
  const interest = +(((balance + closeness + drama + spread) / 4) * finishedRate).toFixed(3)
  return {
    map: results[0]?.map ?? "?",
    games: results.length,
    finishedRate: +finishedRate.toFixed(3),
    avgRounds: +mean(results.map((r) => r.rounds)).toFixed(1),
    avgLeadChanges: +mean(results.map((r) => r.leadChanges)).toFixed(2),
    avgMargin: +mean(results.map((r) => r.margin)).toFixed(2),
    avgContested: +mean(results.map((r) => r.contestedCities)).toFixed(2),
    avgPremium: +mean(results.map((r) => r.auctionPremium)).toFixed(2),
    step3Rate: +mean(results.map((r) => (r.finalStep === 3 ? 1 : 0))).toFixed(3),
    winRates: Object.fromEntries(Object.entries(winRates).map(([k, v]) => [k, +v.toFixed(3)])),
    regionShare,
    scarcity,
    interest,
    errors: results.reduce((a, r) => a + r.errors, 0),
    topRejections: (() => { const t: Record<string, number> = {}; for (const r of results) for (const [k, v] of Object.entries(r.rejections)) t[k] = (t[k] ?? 0) + v; return Object.entries(t).sort((a, b) => b[1] - a[1]).slice(0, 4) })()
  }
}
