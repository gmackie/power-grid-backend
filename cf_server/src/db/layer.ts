import { D1Client } from "@effect/sql-d1"
import { Layer } from "effect"

/** SqlClient + D1Client backed by the Worker's D1 binding. */
export const D1Live = (db: D1Database) => D1Client.layer({ db })

export type D1Layer = ReturnType<typeof D1Live>
export { Layer }
