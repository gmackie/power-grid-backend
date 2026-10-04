/// <reference types="@cloudflare/vitest-pool-workers/types" />
import type { Env as WorkerEnv } from "../../src/env.ts"
import type { D1Migration } from "@cloudflare/vitest-pool-workers"

declare global {
  namespace Cloudflare {
    interface Env extends WorkerEnv {
      TEST_MIGRATIONS: D1Migration[]
    }
  }
}
