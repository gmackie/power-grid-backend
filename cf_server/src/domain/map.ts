/** Map asset format (go_server/maps/*.json) and the connection-cost graph. */
import { Schema } from "effect"
import type { CityInfo, ConnectionInfo } from "../protocol/game.ts"
import type { MapInfo } from "../protocol/lobby.ts"

export const MapData = Schema.Struct({
  id: Schema.String,
  name: Schema.String,
  description: Schema.optional(Schema.String),
  playerCount: Schema.Struct({
    min: Schema.Number,
    max: Schema.Number,
    recommended: Schema.optional(Schema.Array(Schema.Number))
  }),
  regions: Schema.Array(Schema.Struct({ id: Schema.String, name: Schema.String, color: Schema.optional(Schema.String) })),
  cities: Schema.Array(
    Schema.Struct({
      id: Schema.String,
      name: Schema.String,
      region: Schema.String,
      x: Schema.Number,
      y: Schema.Number
    })
  ),
  connections: Schema.Array(Schema.Struct({ from: Schema.String, to: Schema.String, cost: Schema.Number })),
  gameRules: Schema.optional(
    Schema.Struct({
      step2Trigger: Schema.optional(Schema.Number),
      step3Trigger: Schema.optional(Schema.Number),
      startingMoney: Schema.optional(Schema.Number),
      resourceSupply: Schema.optional(Schema.Record(Schema.String, Schema.Number)),
      earningsTable: Schema.optional(Schema.Array(Schema.Number)),
      winConditions: Schema.optional(Schema.Record(Schema.String, Schema.Number))
    })
  )
})
export type MapData = typeof MapData.Type
export const decodeMapData = Schema.decodeUnknownSync(MapData)

export const toMapInfo = (m: MapData): MapInfo => ({
  id: m.id,
  name: m.name,
  description: m.description ?? "",
  playerCount: {
    min: m.playerCount.min,
    max: m.playerCount.max,
    recommended: m.playerCount.recommended ?? []
  },
  regionCount: m.regions.length,
  cityCount: m.cities.length
})

export interface MapGraph {
  readonly data: MapData
  readonly cityIds: ReadonlyArray<string>
  readonly adjacency: ReadonlyMap<string, ReadonlyArray<{ readonly to: string; readonly cost: number }>>
}

export const buildGraph = (data: MapData): MapGraph => {
  const adjacency = new Map<string, Array<{ to: string; cost: number }>>()
  for (const c of data.cities) adjacency.set(c.id, [])
  for (const conn of data.connections) {
    adjacency.get(conn.from)?.push({ to: conn.to, cost: conn.cost })
    adjacency.get(conn.to)?.push({ to: conn.from, cost: conn.cost })
  }
  return { data, cityIds: data.cities.map((c) => c.id), adjacency }
}

/**
 * Cheapest connection cost from any city in `owned` to `target` (Dijkstra over
 * connection costs). Returns undefined if unreachable. Cost is 0 when owned is empty
 * (first city is free) or when target is already owned.
 */
export const connectionCost = (graph: MapGraph, owned: ReadonlyArray<string>, target: string): number | undefined => {
  if (owned.length === 0) return 0
  if (owned.includes(target)) return 0
  const dist = new Map<string, number>()
  const visited = new Set<string>()
  for (const id of owned) dist.set(id, 0)
  // Simple O(V^2) Dijkstra; maps have < 100 cities.
  for (;;) {
    let best: string | undefined
    let bestD = Infinity
    for (const [id, d] of dist) {
      if (!visited.has(id) && d < bestD) {
        best = id
        bestD = d
      }
    }
    if (best === undefined) return undefined
    if (best === target) return bestD
    visited.add(best)
    for (const edge of graph.adjacency.get(best) ?? []) {
      const nd = bestD + edge.cost
      if (nd < (dist.get(edge.to) ?? Infinity)) dist.set(edge.to, nd)
    }
  }
}

export const toWireCities = (data: MapData, slots: Readonly<Record<string, ReadonlyArray<string>>>): Record<string, CityInfo> => {
  const out: Record<string, CityInfo> = {}
  for (const c of data.cities) {
    out[c.id] = { id: c.id, name: c.name, region: c.region, position: [c.x, c.y], slots: [...(slots[c.id] ?? [])] }
  }
  return out
}

export const toWireConnections = (data: MapData): Array<ConnectionInfo> =>
  data.connections.map((c) => ({ city_a: c.from, city_b: c.to, cost: c.cost }))
