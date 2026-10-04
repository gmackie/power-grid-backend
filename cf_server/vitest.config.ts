import { defineConfig } from "vitest/config"

// Unit tests run in plain Node. Durable Object / D1 integration tests live under
// test/workers and use vitest.workers.config.ts (Cloudflare workers pool).
export default defineConfig({
  test: {
    include: ["test/unit/**/*.test.ts"],
    environment: "node"
  }
})
