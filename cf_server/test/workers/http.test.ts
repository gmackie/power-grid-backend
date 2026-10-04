import { SELF, env } from "cloudflare:test"
import { describe, expect, it } from "vitest"

const get = (path: string, init?: RequestInit) => SELF.fetch(`http://example.com${path}`, init)

describe("HTTP API", () => {
  it("serves the Go-compatible liveness endpoints", async () => {
    expect(await (await get("/health")).json()).toEqual({ status: "healthy" })
    expect(await (await get("/ready")).json()).toEqual({ status: "ready" })
    const home = (await (await get("/")).json()) as Record<string, unknown>
    expect(home["status"]).toBe("running")
    expect(home["name"]).toBe(env.SERVER_NAME)
  })

  it("lists seeded maps from D1 with the camelCase MapInfo shape", async () => {
    const res = await get("/maps")
    expect(res.headers.get("Access-Control-Allow-Origin")).toBe("*")
    const body = (await res.json()) as { maps: Array<Record<string, unknown>> }
    const ids = body.maps.map((m) => m["id"]).sort()
    expect(ids).toEqual(expect.arrayContaining(["germany", "usa"]))
    const usa = body.maps.find((m) => m["id"] === "usa")!
    expect(usa["playerCount"]).toEqual({ min: 2, max: 6, recommended: [3, 4, 5, 6] })
    expect(usa["cityCount"]).toBe(36)
    expect(usa["regionCount"]).toBe(6)
  })

  it("serves a full map by id and 404s unknown maps", async () => {
    const usa = (await (await get("/maps/usa")).json()) as { cities: Array<unknown>; connections: Array<unknown> }
    expect(usa.cities).toHaveLength(36)
    expect(usa.connections).toHaveLength(65)
    expect((await get("/maps/mars")).status).toBe(404)
  })

  it("exposes achievements and an empty leaderboard", async () => {
    const achievements = (await (await get("/api/achievements")).json()) as Array<unknown>
    expect(achievements.length).toBeGreaterThanOrEqual(19)
    const lb = (await (await get("/api/leaderboard?limit=5")).json()) as { leaderboard: Array<unknown>; limit: number }
    expect(lb.limit).toBe(5)
    expect(lb.leaderboard).toEqual([])
    expect((await get("/api/players/nobody")).status).toBe(404)
  })

  it("protects admin routes with the bearer token", async () => {
    expect((await get("/admin/sessions")).status).toBe(401)
    const ok = await get("/admin/sessions", { headers: { Authorization: "Bearer test-token" } })
    expect(ok.status).toBe(200)
    expect(((await ok.json()) as { sessions: Array<unknown> }).sessions).toEqual([])
  })

  it("lets admins upload a map asset that then appears in the list", async () => {
    const map = {
      name: "Tiny",
      description: "two cities",
      playerCount: { min: 2, max: 2, recommended: [2] },
      regions: [{ id: "r", name: "R", color: "#fff" }],
      cities: [
        { id: "a", name: "A", region: "r", x: 0.1, y: 0.1 },
        { id: "b", name: "B", region: "r", x: 0.9, y: 0.9 }
      ],
      connections: [{ from: "a", to: "b", cost: 5 }]
    }
    const put = await get("/admin/maps/tiny", {
      method: "PUT",
      headers: { Authorization: "Bearer test-token", "Content-Type": "application/json" },
      body: JSON.stringify(map)
    })
    expect(put.status).toBe(200)
    const list = (await (await get("/maps")).json()) as { maps: Array<{ id: string }> }
    expect(list.maps.map((m) => m.id)).toContain("tiny")
    const del = await get("/admin/maps/tiny", { method: "DELETE", headers: { Authorization: "Bearer test-token" } })
    expect(del.status).toBe(200)
    const after = (await (await get("/maps")).json()) as { maps: Array<{ id: string }> }
    expect(after.maps.map((m) => m.id)).not.toContain("tiny")
  })
})
