# Power Grid server on Cloudflare (Effect 4 + Durable Objects + D1)

TypeScript replacement for `go_server/`. Wire-compatible with the existing LÖVE and React
clients on the `/ws` lobby socket and the `/game` game socket (see `docs/PORT_NOTES.md`
for the exact message catalogue and the list of Go bugs deliberately not reproduced).

## Architecture

```
src/index.ts            Worker fetch(): routes WebSocket upgrades to Durable Objects,
                        everything else to the Effect HttpRouter (src/http/router.ts)
src/do/LobbyHubDO.ts    one instance ("global"): /ws lobby protocol, sessions, lobbies,
                        idle-session cleanup alarm, relay for bare /game sockets
src/do/GameDO.ts        one instance per game: engine state in DO storage, hibernating
                        WebSockets, admin spectator feed, D1 recording via Effect
src/domain/             pure engine: rules tables, deck, resource market, map graph,
                        engine.ts (phases, auction, buying, building, bureaucracy)
src/protocol/           Effect Schemas for both envelopes + every payload, error strings
src/db/                 Effect services over @effect/sql-d1: Assets (maps, decks),
                        Records (players, games, actions, events, achievements), Analytics
migrations/             D1 schema (0001) and generated seed (0002: maps, deck, achievements)
```

Durable Object storage holds live state (sessions, lobbies, in-progress games). D1 holds
everything that outlives a game or that an admin may change: players, game records and
statistics, achievements, map assets and power plant decks.

Effect usage: services are `Context.Service` classes provided through `Layer`s; each DO
owns a `ManagedRuntime` built from `makeAppLayer(env)`; the engine is pure and lifts into
`Effect` at its boundary (`Engine.applyAction` etc. fail with `GameError`); the HTTP API is
an `HttpApi` contract implemented with `HttpApiBuilder`, served through `HttpRouter.toWebHandler`.
`src/http/client.ts` creates an Effect `HttpApiClient` from that same contract.
OpenAPI is generated at `/openapi.json`, with interactive documentation at `/docs`.

## Endpoints

| Path | Protocol | Notes |
|---|---|---|
| `GET /ws` (upgrade) | lobby | identical to Go `handlers/lobby_handler.go` |
| `GET /game?game_id=<id>` (upgrade) | game | preferred; `GAME_STARTING.game_url` carries this URL |
| `GET /game` (upgrade) | game | bare URL (LÖVE client); first `CONNECT`/`JOIN_GAME`/`CREATE_GAME` picks the game, relayed through the hub |
| `GET /ws/admin/game/<id>?token=` (upgrade) | game | read-only `GAME_STATE` feed |
| `/`, `/health`, `/ready` | HTTP | same JSON as Go |
| `GET /maps`, `GET /maps/:id` | HTTP | from D1 `maps` |
| `GET /api/players[/:name[/stats|/achievements|/history]]` | HTTP | |
| `GET /api/achievements`, `/api/leaderboard` | HTTP | |
| `GET /api/analytics/{games,achievements,activity}` | HTTP | |
| `GET /api/games[/:id[/live]]`, `GET /api/lobbies` | HTTP | |
| `/admin/*` | HTTP | `Authorization: Bearer $ADMIN_TOKEN` (sessions, kick, lobbies, maps PUT/DELETE, decks) |
| `/api/admin/simulated/*` | HTTP | 501, local AI processes do not exist on Workers |

Additive fields on `GAME_STATE` (clients ignore unknown keys): `step`, `auction`,
`plant_market {current, future, deck_remaining}`, `winner_id`, `map_id`, and per-player
`connected`, `has_passed`. Additive message types: `STEP_CHANGE`, `PLANT_BOUGHT`, `GAME_END`.
`power_plants` now carries the 8 (step 3: 6) market plants instead of the whole deck.

## Board lab (agent simulation)

```bash
pnpm sim -- --games 40 --players 4 --evolve 6          # all boards → sim/reports/<timestamp>.md + .json
pnpm sim -- --maps megalopolis,frontier --games 100
```

`src/sim/bots.ts` holds bot archetypes (balanced, expander, hoarder, sniper, greenie, random)
and an evolvable weighted heuristic; `src/sim/run.ts` plays full games against the pure engine
and computes per-board metrics plus a composite interest score. See `../maps/README.md`.

## Develop

```bash
pnpm install
pnpm seed:build                        # regenerate migrations/0002_seed.sql from maps
cp .dev.vars.example .dev.vars         # set ADMIN_TOKEN
pnpm db:migrate:local                  # apply D1 migrations to the local database
pnpm dev                               # wrangler dev on http://localhost:8787
pnpm typecheck
pnpm test:unit                         # engine tests (Node)
pnpm test:workers                      # Durable Objects + D1 under miniflare
```

Point the React client at it with `VITE_WS_URL=ws://localhost:8787/ws` and the LÖVE client
at `ws://localhost:8787/game`.

## Deploy

Production web app, HTTP API, and WebSockets: `https://power.gmac.io`.
The legacy Worker URL remains available at `https://power-grid-server.gmac.workers.dev` (D1 `powergrid`, created 2026-10-04).
The admin token is in `.admin-token.local` (gitignored) on the machine that deployed.

```bash
wrangler d1 create powergrid           # once; database_id is already in wrangler.jsonc
pnpm db:migrate:remote
wrangler secret put ADMIN_TOKEN
pnpm build:web /path/to/react_client   # builds web-dist with the production WebSocket URL
pnpm deploy                          # uses wrangler.web.jsonc, including assets and domain route
```

The deployment requires a freshly built `web-dist`; a backend-only deployment would remove
the website assets. CI must build the matching React checkout before deploying.
`.github/workflows/deploy-cf.yml` requires
`CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID` secrets.

## History

`docs/2026-10-04-migration-plan-draft.html` is the phased plan written during the first (lobby-only)
slice; `docs/PORT_NOTES.md` is the current reference. Phases 1 to 3 of that plan are implemented
here; phase 4 (client adapter, staged cutover) remains.

## Not ported

- Prometheus `/metrics`, OTel middleware (use Workers observability / Logpush).
- Simulated games / AI client launcher (spawned local processes).
- File-based analytics mode, SQLite maintenance endpoints, log streaming over WebSocket.
The deployed React client uses a separate game socket and adapts the server `GAME_STATE`
payload to its UI store. Lobby registration, readiness, game start, bidding, and plant purchase
were verified against the production domain. A complete browser-played game is not yet covered
by an automated end-to-end test.
