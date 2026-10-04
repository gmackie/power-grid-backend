// Custom gameboard generator. Same mechanics as the originals; twists live in topology and
// in gameRules (starting money, step triggers, resource supply, win conditions).
//   node maps/tools/gen-map.mjs            # writes maps/<id>.json for every design below
import { writeFileSync } from "node:fs"
import { dirname, join } from "node:path"
import { fileURLToPath } from "node:url"

const here = dirname(fileURLToPath(import.meta.url))
const rngOf = (seed) => { let t = seed >>> 0; return () => { t = (t + 0x6d2b79f5) >>> 0; let x = Math.imul(t ^ (t >>> 15), t | 1); x ^= x + Math.imul(x ^ (x >>> 7), x | 61); return ((x ^ (x >>> 14)) >>> 0) / 4294967296 } }
const slug = (s) => s.toLowerCase().replace(/[^a-z0-9]+/g, "_").replace(/^_|_$/g, "")
const dist = (a, b) => Math.hypot(a.x - b.x, a.y - b.y)
const BASE_RULES = { step2Trigger: 7, step3Trigger: 18, startingMoney: 50, resourceSupply: { coal: 24, oil: 18, garbage: 6, uranium: 2 }, earningsTable: [10, 22, 33, 44, 54, 64, 73, 82, 90, 98, 105, 112, 118, 124, 129, 134, 138, 142, 145, 148, 150], winConditions: { 2: 21, 3: 17, 4: 17, 5: 15, 6: 14 } }

/** Place n cities around a centre with jitter, min spacing. */
const scatter = (rng, cx, cy, rx, ry, n, names, region, minGap = 0.06) => {
  const out = []
  let guard = 0
  while (out.length < n && guard++ < 4000) {
    const a = rng() * Math.PI * 2, r = Math.sqrt(rng())
    const c = { x: +(cx + Math.cos(a) * rx * r).toFixed(3), y: +(cy + Math.sin(a) * ry * r).toFixed(3) }
    if (c.x < 0.05 || c.x > 0.95 || c.y < 0.08 || c.y > 0.9) continue
    if (out.every((o) => dist(o, c) >= minGap)) out.push({ id: slug(names[out.length]), name: names[out.length], region, ...c })
  }
  return out
}

/** Connect cities: nearest-neighbour spanning tree + extra short edges. Cost from distance. */
const connectWithin = (rng, cities, costOf, extraPerCity = 1.2) => {
  const edges = new Map()
  const key = (a, b) => [a.id, b.id].sort().join("|")
  const add = (a, b) => { const k = key(a, b); if (!edges.has(k)) edges.set(k, { from: a.id, to: b.id, cost: costOf(dist(a, b), rng) }) }
  const inTree = [cities[0]]
  const rest = cities.slice(1)
  while (rest.length) {
    let best, bi = -1, bd = Infinity
    for (let i = 0; i < rest.length; i++) for (const t of inTree) { const d = dist(rest[i], t); if (d < bd) { bd = d; best = t; bi = i } }
    add(rest[bi], best); inTree.push(rest[bi]); rest.splice(bi, 1)
  }
  const want = Math.round(cities.length * extraPerCity)
  const pairs = []
  for (let i = 0; i < cities.length; i++) for (let j = i + 1; j < cities.length; j++) pairs.push([cities[i], cities[j], dist(cities[i], cities[j])])
  pairs.sort((a, b) => a[2] - b[2])
  let added = 0
  for (const [a, b] of pairs) { if (added >= want) break; if (!edges.has(key(a, b))) { add(a, b); added++ } }
  return [...edges.values()]
}

/** k cheapest links between two regions. */
const bridge = (rng, A, B, k, costOf) => {
  const pairs = []
  for (const a of A) for (const b of B) pairs.push([a, b, dist(a, b)])
  pairs.sort((x, y) => x[2] - y[2])
  return pairs.slice(0, k).map(([a, b, d]) => ({ from: a.id, to: b.id, cost: costOf(d, rng) }))
}

const linear = (lo, hi) => (d, rng) => Math.max(lo, Math.min(hi, Math.round(lo + d * 60 + (rng() - 0.5) * 4)))

const DESIGNS = {
  archipelago: () => {
    const rng = rngOf(11)
    const islands = [
      ["Northreach", "#4169E1", 0.22, 0.22, ["Gullhaven", "Kelpmoor", "Brinefall", "Tidewatch", "Fogmere", "Saltrock", "Driftwood"]],
      ["Ember Isle", "#FF6347", 0.72, 0.2, ["Cinderport", "Ashgate", "Lavaquay", "Basalt", "Sootcombe", "Fumarole", "Kilnbay"]],
      ["Greenholm", "#32CD32", 0.5, 0.5, ["Mossbridge", "Fernhollow", "Dewhurst", "Sporefield", "Lichenby", "Vinecross", "Bramblewick"]],
      ["Westmarch Cays", "#FFD700", 0.2, 0.74, ["Sunreef", "Coralgate", "Lagoona", "Pearlcove", "Shellhaven", "Palmstead", "Sandspit"]],
      ["Stormcape", "#9932CC", 0.78, 0.74, ["Galeport", "Thunderhead", "Squallrock", "Lightning Point", "Rainmoor", "Hailstone", "Windward"]]
    ]
    const regions = islands.map(([name, color]) => ({ id: slug(name), name, color }))
    const cities = islands.flatMap(([name, , cx, cy, names]) => scatter(rng, cx, cy, 0.15, 0.13, 7, names, slug(name), 0.055))
    const byRegion = (id) => cities.filter((c) => c.region === id)
    const connections = regions.flatMap((r) => connectWithin(rng, byRegion(r.id), linear(2, 8), 1.0))
    const ferries = [[0, 2], [1, 2], [3, 2], [4, 2], [0, 1], [3, 4], [0, 3], [1, 4]]
    for (const [i, j] of ferries) connections.push(...bridge(rng, byRegion(regions[i].id), byRegion(regions[j].id), 1, () => 18 + Math.floor(rng() * 8)))
    return {
      id: "archipelago", name: "Archipelago", description: "Five islands with cheap local grids and costly ferry crossings. Cash is tight; pick your island.",
      playerCount: { min: 2, max: 6, recommended: [3, 4, 5] }, regions, cities, connections,
      gameRules: { ...BASE_RULES, startingMoney: 60, step2Trigger: 6, winConditions: { 2: 19, 3: 16, 4: 15, 5: 14, 6: 13 } }
    }
  },

  megalopolis: () => {
    const rng = rngOf(23)
    const core = { id: "core", name: "The Core", color: "#9932CC" }
    const coreNames = ["Hub Central", "Transit One", "Gridlock", "Neon Row", "Skyline", "Undercity", "Vaultside", "Pylon Square", "Circuit Park", "Datum"]
    const coreCities = coreNames.map((n, i) => { const a = (i / coreNames.length) * Math.PI * 2; return { id: slug(n), name: n, region: "core", x: +(0.5 + Math.cos(a) * 0.14).toFixed(3), y: +(0.5 + Math.sin(a) * 0.12).toFixed(3) } })
    const outer = [
      ["North Sprawl", "#4169E1", 0.5, 0.14, ["Frostgate", "Hillcrest", "Bramwell", "Northwick", "Pinefield", "Aldercross"]],
      ["East Sprawl", "#32CD32", 0.86, 0.5, ["Daybreak", "Eastmoor", "Riverbend", "Sunfield", "Oakridge", "Millrun"]],
      ["South Sprawl", "#FF6347", 0.5, 0.86, ["Heatwell", "Southgate", "Dunmere", "Coppertown", "Ironvale", "Kiln Row"]],
      ["West Sprawl", "#FFD700", 0.14, 0.5, ["Duskport", "Westhaven", "Goldmarch", "Harborlight", "Fairwater", "Lantern Hill"]]
    ]
    const regions = [core, ...outer.map(([name, color]) => ({ id: slug(name), name, color }))]
    const cities = [...coreCities, ...outer.flatMap(([name, , cx, cy, names]) => scatter(rng, cx, cy, 0.14, 0.11, 6, names, slug(name), 0.06))]
    const byRegion = (id) => cities.filter((c) => c.region === id)
    // ring in the core, dirt cheap
    const connections = coreCities.map((c, i) => ({ from: c.id, to: coreCities[(i + 1) % coreCities.length].id, cost: i % 3 === 0 ? 0 : 2 }))
    for (let i = 0; i < coreCities.length; i += 2) connections.push({ from: coreCities[i].id, to: coreCities[(i + 5) % coreCities.length].id, cost: 4 })
    for (const r of regions.slice(1)) connections.push(...connectWithin(rng, byRegion(r.id), linear(4, 12), 0.8))
    for (const r of regions.slice(1)) connections.push(...bridge(rng, byRegion(r.id), coreCities, 2, () => 10 + Math.floor(rng() * 6)))
    for (const [i, j] of [[1, 2], [2, 3], [3, 4], [4, 1]]) connections.push(...bridge(rng, byRegion(regions[i].id), byRegion(regions[j].id), 1, () => 13 + Math.floor(rng() * 5)))
    return {
      id: "megalopolis", name: "Megalopolis", description: "A dense, nearly free city core ringed by sprawl. Everyone wants the Core; the Core fills fast.",
      playerCount: { min: 2, max: 6, recommended: [4, 5, 6] }, regions, cities, connections,
      gameRules: { ...BASE_RULES, step2Trigger: 8, resourceSupply: { coal: 24, oil: 18, garbage: 9, uranium: 2 }, winConditions: { 2: 22, 3: 18, 4: 18, 5: 16, 6: 15 } }
    }
  },

  frontier: () => {
    const rng = rngOf(37)
    const bands = [
      ["Old Coast", "#4169E1", 0.12, ["Harrowgate", "Portsend", "Kingsbury", "Millford", "Ashby", "Stonebridge", "Lowmarsh", "Brightwater"]],
      ["Heartland", "#32CD32", 0.32, ["Wheatridge", "Silo City", "Barrowfield", "Cornwell", "Prairie Rose", "Hollins", "Grange"]],
      ["The Divide", "#FFD700", 0.52, ["Summit", "Snowline", "Pass Fork", "Cragmoor", "Timberline", "Ridgeback"]],
      ["Badlands", "#FF6347", 0.72, ["Dustbowl", "Red Mesa", "Bonecreek", "Sunstroke", "Gulch", "Tumbleton"]],
      ["Far Shore", "#9932CC", 0.9, ["Lastlight", "Edgewater", "New Hope", "Terminus", "Outpost Nine"]]
    ]
    const regions = bands.map(([name, color]) => ({ id: slug(name), name, color }))
    const cities = bands.flatMap(([name, , cx, names]) => scatter(rng, cx, 0.5, 0.08, 0.36, names.length, names, slug(name), 0.07))
    const byRegion = (id) => cities.filter((c) => c.region === id)
    const connections = []
    bands.forEach(([name], i) => connections.push(...connectWithin(rng, byRegion(slug(name)), linear(2 + i * 2, 8 + i * 4), 0.9)))
    for (let i = 0; i < bands.length - 1; i++) connections.push(...bridge(rng, byRegion(regions[i].id), byRegion(regions[i + 1].id), 2, () => 8 + i * 4 + Math.floor(rng() * 4)))
    return {
      id: "frontier", name: "Frontier", description: "A long march west to east: cheap and crowded on the Old Coast, costly and empty on the Far Shore. Uranium is plentiful out there.",
      playerCount: { min: 2, max: 6, recommended: [2, 3, 4] }, regions, cities, connections,
      gameRules: { ...BASE_RULES, startingMoney: 45, resourceSupply: { coal: 20, oil: 14, garbage: 6, uranium: 6 }, winConditions: { 2: 20, 3: 16, 4: 15, 5: 14, 6: 13 } }
    }
  }
}

const validate = (m) => {
  const ids = new Set(m.cities.map((c) => c.id))
  if (ids.size !== m.cities.length) throw new Error(`${m.id}: duplicate city ids`)
  const adj = new Map([...ids].map((id) => [id, []]))
  for (const c of m.connections) { if (!ids.has(c.from) || !ids.has(c.to)) throw new Error(`${m.id}: bad edge ${c.from}-${c.to}`); adj.get(c.from).push(c.to); adj.get(c.to).push(c.from) }
  const seen = new Set([m.cities[0].id]); const stack = [m.cities[0].id]
  while (stack.length) for (const n of adj.get(stack.pop())) if (!seen.has(n)) { seen.add(n); stack.push(n) }
  if (seen.size !== ids.size) throw new Error(`${m.id}: graph not connected (${seen.size}/${ids.size})`)
  const dedup = new Map(); for (const c of m.connections) dedup.set([c.from, c.to].sort().join("|"), c)
  m.connections = [...dedup.values()]
}

for (const [id, make] of Object.entries(DESIGNS)) {
  const m = make(); validate(m)
  writeFileSync(join(here, "..", `${id}.json`), JSON.stringify(m, null, 2) + "\n")
  console.log(`${id}: ${m.cities.length} cities, ${m.regions.length} regions, ${m.connections.length} connections`)
}
