/**
 * Worker entry point. WebSocket upgrades go to Durable Objects; everything else is served
 * by the Effect HttpRouter in src/http/router.ts.
 *
 *   /ws                      lobby protocol            -> LobbyHubDO ("global")
 *   /game?game_id=<id>       game protocol             -> GameDO(<id>)
 *   /game                    game protocol, no id      -> LobbyHubDO relay (first message picks the game)
 *   /ws/admin/game/<id>      read-only spectator feed  -> GameDO(<id>)   (admin token)
 */
import type { Env } from "./env.ts"
import { makeHttpHandler } from "./http/router.ts"

export { GameDO } from "./do/GameDO.ts"
export { LobbyHubDO } from "./do/LobbyHubDO.ts"

const handlers = new WeakMap<Env, ReturnType<typeof makeHttpHandler>>()

const CORS_HEADERS: Record<string, string> = {
  "Access-Control-Allow-Origin": "*",
  "Access-Control-Allow-Methods": "GET, POST, PUT, DELETE, OPTIONS",
  "Access-Control-Allow-Headers": "Content-Type, Authorization"
}

const withCors = (res: Response): Response => {
  const headers = new Headers(res.headers)
  for (const [k, v] of Object.entries(CORS_HEADERS)) headers.set(k, v)
  return new Response(res.body, { status: res.status, statusText: res.statusText, headers })
}

const isUpgrade = (request: Request) => request.headers.get("Upgrade")?.toLowerCase() === "websocket"

const isAuthorized = (request: Request, url: URL, env: Env): boolean => {
  if (!env.ADMIN_TOKEN) return false
  const header = request.headers.get("Authorization") ?? ""
  const bearer = header.startsWith("Bearer ") ? header.slice(7) : ""
  return bearer === env.ADMIN_TOKEN || url.searchParams.get("token") === env.ADMIN_TOKEN
}

const isAdminPath = (path: string) => path.startsWith("/ws/admin/")

/** Browser origin allowlist (worker/ slice idea). Non-browser clients send no Origin and pass. */
const originAllowed = (request: Request, env: Env): boolean => {
  const origin = request.headers.get("Origin")
  const allowed = (env.ALLOWED_ORIGINS ?? "*").split(",").map((s) => s.trim()).filter(Boolean)
  if (!origin || allowed.length === 0 || allowed.includes("*")) return true
  return allowed.includes(origin)
}

export default {
  async fetch(request: Request, env: Env): Promise<Response> {
    const url = new URL(request.url)

    if (!originAllowed(request, env)) return new Response("Origin not allowed", { status: 403 })

    if (request.method === "OPTIONS") return withCors(new Response(null, { status: 204 }))

    if (isAdminPath(url.pathname) && !isAuthorized(request, url, env)) {
      return withCors(Response.json({ error: "unauthorized", status: 401 }, { status: 401 }))
    }

    if (isUpgrade(request)) {
      if (url.pathname === "/ws") {
        return env.LOBBY_HUB.get(env.LOBBY_HUB.idFromName("global")).fetch(request)
      }
      if (url.pathname === "/game") {
        const gameId = url.searchParams.get("game_id")
        if (gameId) {
          return env.GAME.get(env.GAME.idFromName(gameId)).fetch(request)
        }
        return env.LOBBY_HUB.get(env.LOBBY_HUB.idFromName("global")).fetch(request)
      }
      const spectator = /^\/ws\/admin\/game\/([^/]+)$/.exec(url.pathname)
      if (spectator?.[1]) {
        return env.GAME.get(env.GAME.idFromName(spectator[1])).fetch(request)
      }
      return new Response("unknown websocket endpoint", { status: 404 })
    }

    // Static hosting is enabled only in the web deployment configuration.
    const apiPath = /^\/(api|admin|maps|health|ready|docs|openapi\.json|ws|game)(\/|$)/.test(url.pathname)
    if (env.ASSETS && !apiPath) return env.ASSETS.fetch(request)

    let h = handlers.get(env)
    if (!h) {
      h = makeHttpHandler(env)
      handlers.set(env, h)
    }
    return withCors(await h.handler(request))
  }
} satisfies ExportedHandler<Env>
