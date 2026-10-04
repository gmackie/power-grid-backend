/** Game assets stored in D1: maps and power plant decks. */
import { Context, Effect, Layer, Schema } from "effect"
import { SqlClient } from "effect/sql"
import type { SqlError } from "effect/sql/SqlError"
import type { PlantDef } from "../domain/deck-default.ts"
import { DEFAULT_DECK } from "../domain/deck-default.ts"
import { decodeMapData, toMapInfo, type MapData } from "../domain/map.ts"
import type { MapInfo } from "../protocol/lobby.ts"

export class AssetError extends Schema.TaggedError<AssetError>()("AssetError", { message: Schema.String }) {}

export interface DeckSummary {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly is_default: boolean
  readonly plant_count: number
}

export interface AssetsShape {
  readonly listMaps: Effect.Effect<ReadonlyArray<MapInfo>, SqlError>
  readonly getMap: (id: string) => Effect.Effect<MapData, SqlError | AssetError>
  readonly getDeck: (deckId?: string) => Effect.Effect<ReadonlyArray<PlantDef>, SqlError>
  // admin
  readonly upsertMap: (map: MapData) => Effect.Effect<void, SqlError>
  readonly setMapActive: (id: string, active: boolean) => Effect.Effect<boolean, SqlError>
  readonly listDecks: Effect.Effect<ReadonlyArray<DeckSummary>, SqlError>
  readonly upsertDeck: (deck: { readonly id: string; readonly name: string; readonly description?: string | undefined; readonly is_default?: boolean | undefined; readonly plants: ReadonlyArray<PlantDef> }) => Effect.Effect<void, SqlError>
}

export class Assets extends Context.Service<Assets, AssetsShape>()("Assets") {}

const MapRow = Schema.Struct({ id: Schema.String, data: Schema.String })
const PlantRow = Schema.Struct({
  number: Schema.Number,
  capacity: Schema.Number,
  resource_type: Schema.Literals(["Coal", "Oil", "Garbage", "Uranium", "Hybrid", "Wind"]),
  resource_cost: Schema.Number
})

export const AssetsLive = Layer.effect(
  Assets,
  Effect.gen(function* () {
    const sql = yield* SqlClient.SqlClient
    const decodeMapRows = Schema.decodeUnknownSync(Schema.Array(MapRow))
    const decodePlantRows = Schema.decodeUnknownSync(Schema.Array(PlantRow))

    const listMaps: AssetsShape["listMaps"] = Effect.map(
      sql`SELECT id, data FROM maps WHERE is_active = TRUE ORDER BY name`,
      (rows) => decodeMapRows(rows).map((r) => toMapInfo(decodeMapData(JSON.parse(r.data))))
    )

    const getMap: AssetsShape["getMap"] = (id) =>
      Effect.flatMap(sql`SELECT id, data FROM maps WHERE id = ${id} AND is_active = TRUE`, (rows) => {
        const row = decodeMapRows(rows)[0]
        return row
          ? Effect.succeed(decodeMapData(JSON.parse(row.data)))
          : Effect.fail(new AssetError({ message: `map not found: ${id}` }))
      })

    const getDeck: AssetsShape["getDeck"] = (deckId) =>
      Effect.map(
        deckId
          ? sql`SELECT number, capacity, resource_type, resource_cost FROM power_plants WHERE deck_id = ${deckId} ORDER BY number`
          : sql`SELECT p.number, p.capacity, p.resource_type, p.resource_cost FROM power_plants p
                JOIN power_plant_decks d ON d.id = p.deck_id WHERE d.is_default = TRUE ORDER BY p.number`,
        (rows) => {
          const plants = decodePlantRows(rows)
          if (plants.length === 0) return DEFAULT_DECK
          return plants.map((p) => ({
            number: p.number,
            capacity: p.capacity,
            resourceType: p.resource_type,
            resourceCost: p.resource_cost
          }))
        }
      )

    const upsertMap: AssetsShape["upsertMap"] = (map) =>
      Effect.asVoid(
        sql`INSERT INTO maps (id, name, description, player_min, player_max, data, is_active, version)
            VALUES (${map.id}, ${map.name}, ${map.description ?? ""}, ${map.playerCount.min}, ${map.playerCount.max}, ${JSON.stringify(map)}, TRUE, 1)
            ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, player_min = EXCLUDED.player_min,
              player_max = EXCLUDED.player_max, data = EXCLUDED.data, is_active = TRUE, version = maps.version + 1, updated_at = CURRENT_TIMESTAMP`
      )

    const setMapActive: AssetsShape["setMapActive"] = (id, active) =>
      Effect.map(
        sql`UPDATE maps SET is_active = ${active}, updated_at = CURRENT_TIMESTAMP WHERE id = ${id} RETURNING id`,
        (rows) => rows.length > 0
      )

    const DeckRow = Schema.Struct({
      id: Schema.String,
      name: Schema.String,
      description: Schema.String,
      is_default: Schema.Unknown,
      plant_count: Schema.Number
    })
    const listDecks: AssetsShape["listDecks"] = Effect.map(
      sql`SELECT d.id, d.name, d.description, d.is_default, COUNT(p.number) AS plant_count
          FROM power_plant_decks d LEFT JOIN power_plants p ON p.deck_id = d.id GROUP BY d.id ORDER BY d.is_default DESC, d.name`,
      (rows) => Schema.decodeUnknownSync(Schema.Array(DeckRow))(rows).map((r) => ({ ...r, is_default: r.is_default === 1 || r.is_default === true }))
    )

    const upsertDeck: AssetsShape["upsertDeck"] = (deck) =>
      Effect.gen(function* () {
        if (deck.is_default) yield* sql`UPDATE power_plant_decks SET is_default = FALSE`
        yield* sql`INSERT INTO power_plant_decks (id, name, description, is_default) VALUES (${deck.id}, ${deck.name}, ${deck.description ?? ""}, ${deck.is_default ?? false})
                   ON CONFLICT(id) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description, is_default = EXCLUDED.is_default`
        yield* sql`DELETE FROM power_plants WHERE deck_id = ${deck.id}`
        for (const p of deck.plants) {
          yield* sql`INSERT INTO power_plants (deck_id, number, capacity, resource_type, resource_cost) VALUES (${deck.id}, ${p.number}, ${p.capacity}, ${p.resourceType}, ${p.resourceCost})`
        }
      })

    return Assets.of({ listMaps, getMap, getDeck, upsertMap, setMapActive, listDecks, upsertDeck })
  })
)
