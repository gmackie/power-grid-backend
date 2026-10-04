/** Effect runtime shared by Durable Objects and the Worker: D1 + repositories. */
import { Layer, ManagedRuntime } from "effect"
import { AssetsLive } from "../db/assets.ts"
import { D1Live } from "../db/layer.ts"
import { RecordsLive } from "../db/records.ts"
import type { Env } from "../env.ts"

export const makeAppLayer = (env: Env) => Layer.mergeAll(AssetsLive, RecordsLive).pipe(Layer.provide(D1Live(env.DB)))

export type AppLayer = ReturnType<typeof makeAppLayer>

export const makeRuntime = (env: Env) => ManagedRuntime.make(makeAppLayer(env))
export type AppRuntime = ReturnType<typeof makeRuntime>
