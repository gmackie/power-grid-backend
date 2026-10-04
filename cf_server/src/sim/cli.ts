/**
 * Board lab: play many bot games per board, compare archetypes, optionally evolve weights.
 *
 *   node src/sim/cli.ts --games 40 --players 4                 # all boards, archetype tournament
 *   node src/sim/cli.ts --maps megalopolis,frontier --evolve 6  # evolve heuristic weights per board
 *   node src/sim/cli.ts --out sim/reports/custom.md
 */
import { mkdirSync, readdirSync, readFileSync, writeFileSync } from "node:fs"
import { dirname, join, resolve } from "node:path"
import { fileURLToPath } from "node:url"
import { DEFAULT_DECK } from "../domain/deck-default.ts"
import { decodeMapData, type MapData } from "../domain/map.ts"
import { seedRng, nextFloat, type Rng } from "../domain/rng.ts"
import { ARCHETYPES, DEFAULT_WEIGHTS, heuristicBot, mutate, type Bot, type Weights } from "./bots.ts"
import { runGame, summarize, type BoardSummary, type GameResult } from "./run.ts"

const here = dirname(fileURLToPath(import.meta.url))
const args = process.argv.slice(2)
const opt = (k: string, d: string) => { const i = args.indexOf(`--${k}`); return i >= 0 && args[i + 1] ? args[i + 1]! : d }
const GAMES = Number(opt("games", "30"))
const PLAYERS = Number(opt("players", "4"))
const EVOLVE = Number(opt("evolve", "0"))
const MAX_ROUNDS = Number(opt("maxRounds", "60"))
const mapsDir = resolve(here, "../../../maps")
const wantMaps = opt("maps", "").split(",").filter(Boolean)
const stamp = new Date().toISOString().slice(0, 16).replace(/[:T]/g, "-")
const outPath = resolve(here, "../..", opt("out", `sim/reports/${stamp}.md`))

const maps: Array<MapData> = readdirSync(mapsDir)
  .filter((f) => f.endsWith(".json") && (wantMaps.length === 0 || wantMaps.includes(f.replace(".json", ""))))
  .map((f) => decodeMapData(JSON.parse(readFileSync(join(mapsDir, f), "utf8"))))

const pickSeats = (pool: ReadonlyArray<Bot>, n: number, rng0: Rng): [Array<Bot>, Rng] => {
  let rng = rng0
  const idx = pool.map((_, i) => i)
  for (let i = idx.length - 1; i > 0; i--) { const [f, r] = nextFloat(rng); rng = r; const j = Math.floor(f * (i + 1)); [idx[i], idx[j]] = [idx[j]!, idx[i]!] }
  return [idx.slice(0, n).map((i) => pool[i]!), rng]
}

const tournament = (map: MapData, pool: ReadonlyArray<Bot>, games: number, seed0: number): Array<GameResult> => {
  const out: Array<GameResult> = []
  let rng = seedRng(seed0)
  for (let g = 0; g < games; g++) {
    const [bots, r] = pickSeats(pool, Math.min(PLAYERS, pool.length), rng)
    rng = r
    out.push(runGame({ map, deck: DEFAULT_DECK, bots, seed: seed0 * 1000 + g, maxRounds: MAX_ROUNDS }))
  }
  return out
}

const fmtW = (w: Weights) => Object.entries(w).map(([k, v]) => `${k}=${v}`).join(" ")

const lines: Array<string> = [`# Board lab report ${new Date().toISOString()}`, "", `games/board=${GAMES} players=${PLAYERS} evolve=${EVOLVE} maxRounds=${MAX_ROUNDS}`, ""]
const summaries: Array<BoardSummary> = []
const t0 = Date.now()

for (const map of maps) {
  const results = tournament(map, ARCHETYPES, GAMES, map.id.length * 101 + 7)
  const sum = summarize(results)
  summaries.push(sum)
  lines.push(`## ${map.name} (${map.id})`, "", map.description ?? "", "")
  lines.push(`| metric | value |`, `|---|---|`)
  lines.push(`| finished | ${(sum.finishedRate * 100).toFixed(0)}% |`, `| avg rounds | ${sum.avgRounds} |`, `| lead changes | ${sum.avgLeadChanges} |`, `| win margin (cities) | ${sum.avgMargin} |`, `| contested cities (3 houses) | ${sum.avgContested} |`, `| auction premium over face | ${sum.avgPremium} |`, `| reached step 3 | ${(sum.step3Rate * 100).toFixed(0)}% |`, `| **interest** | **${sum.interest}** |`, `| engine rejections | ${sum.errors} |`, "")
  lines.push(`Win rate by archetype: ${Object.entries(sum.winRates).sort((a, b) => b[1] - a[1]).map(([k, v]) => `${k} ${(v * 100).toFixed(0)}%`).join(", ")}`, "")
  lines.push(`Region share of houses: ${Object.entries(sum.regionShare).sort((a, b) => b[1] - a[1]).map(([k, v]) => `${k} ${(v * 100).toFixed(0)}%`).join(", ")}`, "")
  if (sum.topRejections.length) lines.push(`Bot rejections: ${sum.topRejections.map(([k, v]) => `${k} (${v})`).join("; ")}`, "")
  lines.push(`Rounds with an empty market: ${Object.entries(sum.scarcity).map(([k, v]) => `${k} ${v}`).join(", ")}`, "")

  if (EVOLVE > 0) {
    let rng = seedRng(map.id.length * 7)
    let pop: Array<{ w: Weights; fit: number }> = []
    for (let i = 0; i < 8; i++) { const [w, r] = mutate(DEFAULT_WEIGHTS, rng, 0.8); rng = r; pop.push({ w, fit: 0 }) }
    pop[0] = { w: DEFAULT_WEIGHTS, fit: 0 }
    const history: Array<string> = []
    for (let gen = 0; gen < EVOLVE; gen++) {
      const bots = pop.map((p, i) => heuristicBot(`g${gen}-${i}`, p.w))
      const res = tournament(map, bots, Math.max(12, Math.floor(GAMES / 2)), gen * 31 + map.id.length)
      const wins: Record<string, number> = {}; const seats: Record<string, number> = {}
      for (const r of res) { for (const b of Object.values(r.botOf)) seats[b] = (seats[b] ?? 0) + 1; if (r.winnerBot) wins[r.winnerBot] = (wins[r.winnerBot] ?? 0) + 1 }
      pop = pop.map((p, i) => ({ ...p, fit: (wins[`g${gen}-${i}`] ?? 0) / Math.max(1, seats[`g${gen}-${i}`] ?? 0) })).sort((a, b) => b.fit - a.fit)
      history.push(`gen ${gen}: best ${(pop[0]!.fit * 100).toFixed(0)}% → ${fmtW(pop[0]!.w)}`)
      const elite = pop.slice(0, 3)
      const next: Array<{ w: Weights; fit: number }> = [...elite]
      while (next.length < 8) { const parent = elite[next.length % elite.length]!; const [w, r] = mutate(parent.w, rng); rng = r; next.push({ w, fit: 0 }) }
      pop = next
    }
    // how does the evolved strategy do against the archetypes?
    const champion = heuristicBot("evolved", pop[0]!.w)
    const vs = summarize(tournament(map, [champion, ...ARCHETYPES], GAMES, 999))
    lines.push(`### Evolved strategy`, "", ...history.map((h) => `- ${h}`), "", `Champion vs archetypes: ${Object.entries(vs.winRates).sort((a, b) => b[1] - a[1]).map(([k, v]) => `${k} ${(v * 100).toFixed(0)}%`).join(", ")}`, "")
  }
}

lines.push(`## Ranking by interest`, "", `| board | interest | rounds | lead changes | margin | contested | premium |`, `|---|---|---|---|---|---|---|`)
for (const s of [...summaries].sort((a, b) => b.interest - a.interest)) lines.push(`| ${s.map} | ${s.interest} | ${s.avgRounds} | ${s.avgLeadChanges} | ${s.avgMargin} | ${s.avgContested} | ${s.avgPremium} |`)
lines.push("", `_${maps.length} boards, ${Date.now() - t0} ms_`)

mkdirSync(dirname(outPath), { recursive: true })
writeFileSync(outPath, lines.join("\n") + "\n")
writeFileSync(outPath.replace(/\.md$/, ".json"), JSON.stringify(summaries, null, 2))
console.log(lines.join("\n"))
console.log(`\nwrote ${outPath}`)
