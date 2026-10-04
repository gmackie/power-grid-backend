import { cloudflareTest, readD1Migrations } from "@cloudflare/vitest-pool-workers"
import { defineConfig } from "vitest/config"

// Integration tests: real Durable Objects + D1 under miniflare.
//   pnpm test:workers
export default defineConfig(async () => {
  const migrations = await readD1Migrations("./migrations")
  return {
    plugins: [
      cloudflareTest({
        wrangler: { configPath: "./wrangler.jsonc" },
        miniflare: {
          bindings: { TEST_MIGRATIONS: migrations, ADMIN_TOKEN: "test-token" }
        }
      })
    ],
    test: {
      include: ["test/workers/**/*.test.ts"],
      setupFiles: ["./test/workers/apply-migrations.ts"],
      testTimeout: 30_000
    }
  }
})
