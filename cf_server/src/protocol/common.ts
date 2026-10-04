import { Schema } from "effect"

/** Unix seconds, as the Go server emitted. */
export const nowSeconds = (): number => Math.floor(Date.now() / 1000)

/** Lenient JSON object schema used for envelopes whose body is validated per message type. */
export const JsonObject = Schema.Record(Schema.String, Schema.Unknown)
export type JsonObject = typeof JsonObject.Type

export const safeJsonParse = (text: string): unknown | undefined => {
  try {
    return JSON.parse(text)
  } catch {
    return undefined
  }
}
