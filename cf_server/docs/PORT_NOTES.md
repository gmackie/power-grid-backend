# Go server → Cloudflare (Effect 4 + Durable Objects + D1) port notes

Reference distilled from reading `go_server/` (Oct 2026). Everything below is what the Go
code *does*, not what the board game says. Section 5 lists where the port intentionally
diverges.

## 1. Transport: two WebSocket endpoints, two envelopes

| Endpoint | Purpose | Envelope |
|---|---|---|
| `/ws`   | lobby   | `{type, session_id?, player_id?, timestamp, data?}` |
| `/game` | in-game | `{type, timestamp, session_id?, game_id?, payload?}` |

`timestamp` is Unix **seconds**. `/ws` broadcasts (`LOBBY_UPDATED`, `GAME_STARTING`,
post-create `LOBBIES_LISTED`) omit `session_id`. Messages emitted by the game engine
(`GAME_STATE`, `PHASE_CHANGE`, `TURN_CHANGE`) also omit `session_id`.

HTTP: `GET /` → `{"name":"Power Grid Game Server","version":"0.1.0","status":"running"}`,
`GET /health` → `{"status":"healthy"}`, `GET /ready` → `{"status":"ready"}`,
`GET /maps` → `{"maps": MapInfo[]}` (CORS `*`), `GET /maps/{id}` → full MapData (was
unreachable in Go due to mux pattern; port makes it work). `/metrics` (Prometheus) dropped.
Analytics `/api/*`, admin `/admin/*`, simulated `/api/admin/simulated/*`: see §4.

## 2. `/ws` lobby protocol

### Client → server
| type | data |
|---|---|
| `CONNECT` | `{player_name}` (required) |
| `CREATE_LOBBY` | `{lobby_name (req), max_players? (≤0/missing → 6), map_id? (default "usa", must exist), password?}` — Go ignored password; port honours it |
| `JOIN_LOBBY` | `{lobby_id (req), password?}` |
| `LEAVE_LOBBY` | — |
| `CHAT_MESSAGE` | `{content}` (non-empty) |
| `LIST_LOBBIES` | — (no session needed) |
| `LIST_MAPS` | — (no session needed) |
| `SET_READY` | `{ready: boolean}` |
| `START_GAME` | — (host only) |

### Server → client
| type | data |
|---|---|
| `CONNECTED` | `{player_id, player_name, session_id, reconnected: bool, message}` — message is `"New session created successfully"` or `"Session restored successfully"` |
| `ERROR` | `{message}` |
| `LOBBY_CREATED` / `LOBBY_JOINED` / `LOBBY_UPDATED` | `{lobby: Lobby}` |
| `LOBBY_LEFT` | `{lobby_id}` |
| `LOBBIES_LISTED` | `{lobbies: LobbySummary[]}` (always array) |
| `MAPS_LISTED` | `{maps: MapInfo[]}` |
| `READY_UPDATED` | `{player_id, ready}` |
| `GAME_STARTING` | `{lobby: Lobby, game_id, game_url: "/game"}` |

`Lobby`: `{id, name, status: "waiting"|"starting"|"in_game"|"ended", players: {[player_id]: LobbyPlayer}, messages: LobbyMessage[], max_players, map_id, created_at (RFC3339Nano), updated_at}`.
`LobbyPlayer`: `{id, name, is_host, is_ready, joined_at}`. Host starts `is_ready:true`; joiners `false`.
New lobby IDs are six-character codes using `ABCDEFGHJKLMNPQRSTUVWXYZ23456789`.
Codes are randomly generated and checked against existing lobbies before allocation.
Joining accepts lowercase codes and surrounding whitespace; existing UUID lobby IDs remain valid.
Game, player, and session IDs retain their existing formats.

`LobbyMessage`: `{id (uuid), player_id, player_name, content, created_at}`. System: `player_id:"system"`, `player_name:"System"`.
System texts: `"Lobby created. Waiting for players..."`, `"<n> joined the lobby"`, `"<n> left the lobby"`, `"<n> is now the host"`, `"<n> is ready"`, `"<n> is not ready"`, `"Game is starting..."`.
`LobbySummary`: `{id, name, status, player_count, max_players, map_id, has_password, created_at}`.
`MapInfo` (camelCase!): `{id, name, description, playerCount: {min, max, recommended: number[]}, regionCount, cityCount}`.

### Error strings (exact)
`Invalid message format` · `Unknown message type` · `No active session found. Please send CONNECT message first.` · `Player session not found. Please send CONNECT message first.` · `Player session not found` · `Player name is required` · `Lobby name is required` · `Invalid map selected` · `Lobby ID is required` · `Lobby not found` · `Incorrect password` · `Failed to join lobby` · `Player not in a lobby` · `Message content is required` · `Map manager not available` · `Ready status is required` · `Failed to update ready status` · `Only the host can start the game` · `Cannot start the game` · `Failed to create game: <err>`

### Session semantics
- **Client-chosen `session_id`.** Server adopts whatever the client sends (or mints a UUID if empty). `CONNECT` on an unknown id mints `player_id` (uuid v4) → `reconnected:false`. `CONNECT` on a known id returns stored `player_id`/`player_name` → `reconnected:true` (new name ignored).
- Any message with a known `session_id` rebinds that session to the current socket. Lobby membership survives disconnect. Nothing is pushed on reconnect.
- Idle sessions (30 min) are evicted from lobbies; empty lobbies deleted.

### Broadcast matrix
| Event | Sender | Others |
|---|---|---|
| CONNECT | CONNECTED | — |
| CREATE_LOBBY | LOBBY_CREATED | LOBBIES_LISTED to **all** `/ws` sockets |
| JOIN_LOBBY | LOBBY_JOINED | LOBBY_UPDATED to lobby members (incl. joiner) |
| LEAVE_LOBBY | LOBBY_LEFT | LOBBY_UPDATED to remaining |
| CHAT_MESSAGE | — | LOBBY_UPDATED to all members (incl. sender) |
| SET_READY | READY_UPDATED | LOBBY_UPDATED to all members |
| START_GAME | — | GAME_STARTING to all members |

Start rules: host only; ≥2 players; all ready. Lobby → `starting`. Game gets a fresh uuid;
players added with colors from `["red","blue","green","yellow","purple","black"]`.

## 3. `/game` protocol

### Client → server (`payload`)
| type | payload | notes |
|---|---|---|
| `PING` | — | reply `PONG` (no payload) |
| `DISCONNECT` | — | |
| `CONNECT` | `{player_name, player_id, game_id}` | joins room, replies `GAME_STATE`; `PLAYER_JOINED` to room |
| `CREATE_GAME` | `{name, map, max_players}` | |
| `JOIN_GAME` | `{game_id, player_name, color}` | |
| `START_GAME` | — | |
| `BID_PLANT` | `{plant_id, bid}` | AUCTION only |
| `BUY_RESOURCES` | `{resources: {[type]: n}}` | your turn, BUY_RESOURCES phase |
| `BUILD_CITY` | `{city_id}` | your turn, BUILD_CITIES phase |
| `POWER_CITIES` | `{power_plants: number[]}` | your turn, BUREAUCRACY |
| `END_TURN` | — | your turn |

### Server → client
| type | payload |
|---|---|
| `ERROR` | `{code, message}`; codes `MESSAGE_ERROR`, `INVALID_MESSAGE` ("Could not parse message"), `SERVER_BUSY` |
| `PONG` | none |
| `GAME_STATE` | GameStatePayload |
| `PLAYER_JOINED` | `{player_id, player_name}` |
| `PHASE_CHANGE` | `{phase, round}` |
| `TURN_CHANGE` | `{current_player_id, turn}` (turn = index into turn_order) |

After END_TURN that wraps: `PHASE_CHANGE` → `TURN_CHANGE` → `GAME_STATE`. Other actions: `GAME_STATE` only.

### GameStatePayload
```
game_id, name, status: "LOBBY"|"PLAYING"|"FINISHED",
current_phase: "PLAYER_ORDER"|"AUCTION"|"BUY_RESOURCES"|"BUILD_CITIES"|"BUREAUCRACY"|"GAME_END",
current_turn: string (player id, "" if none), current_round: number,
players: {[id]: PlayerInfo}, map: MapInfo, market: MarketInfo,
power_plants: PowerPlantInfo[] | null, turn_order: string[]
```
`PlayerInfo`: `{id, name, color, money, cities: string[], power_plants: PowerPlantInfo[]|null, resources: {[type]: n}, powered_cities}`. Start money 50. `power_plants` is `null` when empty.
`PowerPlantInfo`: `{id (=number), cost, capacity, resource_type, resource_cost}`. Types: `"Coal"|"Oil"|"Garbage"|"Uranium"|"Hybrid"|"Wind"` (Title case).
`MapInfo` (game): `{name, cities: {[id]: CityInfo}, connections: ConnectionInfo[]}`.
`CityInfo`: `{id, name, region, position: [x, y], slots: string[]}`. `ConnectionInfo`: `{city_a, city_b, cost}`.
`MarketInfo`: `{resources: {[type]: number[]}}` — index = price, value = count. Coal/Oil/Garbage length 9, Uranium length 17.
Initial market: Coal/Oil 3 each at prices 3..8; Garbage 3 each at 4..8; Uranium 1 each at 10,12,14,16.

### Engine error strings (exact)
`game is not in playing state` · `not your turn` · `unknown action type` · `invalid payload for bid plant` · `not in auction phase` · `no active auction` · `this plant is not being auctioned` · `not your turn to bid` · `bid must be higher than current bid` · `not enough money for this bid` · `plant not found in market` · `not in buy resources phase` · `player not found` · `resource type not found` · `not enough resources available` · `not enough money` · `not in build cities phase` · `city not found` · `city is full` · `player already has a house in this city` · `city not connected to player's network` · `not in bureaucracy phase` · `player does not own specified power plant` · `not enough resources to power this plant` · `game is full` · `game has already started` · `color already taken` · `need at least 2 players to start` · `session not in any game` · `game not found: <id>`

### Tables
Income by cities powered (cap 150): `[10,22,33,44,54,64,73,82,90,98,105,112,118,124,129,134,138,142,145,148,150]`
Game-end city target by player count: `{2:21, 3:17, 4:17, 5:15, 6:14}`.
Resource refill per bureaucracy (Go table; step ≥2 adds +1 each):
| players | coal | oil | garbage | uranium |
|---|---|---|---|---|
| 2–3 | 3 | 2 | 1 | 1 |
| 4–5 | 4 | 2 | 2 | 1 |
| 6 | 5 | 3 | 3 | 1 |
Slot cost `10 + 5*slots_taken`; connection cost = cheapest edge from any owned city (Go: direct edge only).

### Power plant deck (Go, 42 plants: number/cost, capacity, type, resource cost)
3:1 Oil 2 · 4:1 Coal 2 · 5:1 Hybrid 2 · 6:1 Garbage 1 · 7:2 Oil 3 · 8:2 Coal 3 · 9:1 Oil 1 · 10:2 Coal 2 · 11:2 Uranium 1 · 12:2 Hybrid 2 · 13:1 Wind 0 · 14:2 Garbage 2 · 15:3 Coal 2 · 16:3 Oil 2 · 17:2 Uranium 1 · 18:2 Wind 0 · 19:3 Garbage 2 · 20:5 Coal 3 · 21:4 Hybrid 2 · 22:2 Wind 0 · 23:3 Uranium 1 · 24:4 Garbage 2 · 25:5 Coal 2 · 26:5 Oil 2 · 27:3 Wind 0 · 28:4 Uranium 1 · 29:4 Hybrid 1 · 30:6 Garbage 3 · 31:6 Coal 3 · 32:6 Oil 3 · 33:4 Wind 0 · 34:5 Uranium 1 · 35:5 Oil 1 · 36:7 Coal 3 · 37:4 Wind 0 · 38:7 Garbage 3 · 39:6 Uranium 1 · 40:6 Oil 2 · 42:6 Coal 2 · 44:5 Wind 0 · 46:7 Hybrid 3 · 50:6 Wind 0

### Map file format (`go_server/maps/*.json`)
`{id, name, description, playerCount:{min,max,recommended[]}, regions:[{id,name,color}], cities:[{id,name,region,x,y}] (x,y normalised 0..1), connections:[{from,to,cost}], gameRules:{step2Trigger, step3Trigger, startingMoney, resourceSupply:{coal,oil,garbage,uranium}, resourcePrices:{res:[20]}, earningsTable:[21], winConditions:{"2":21,...}}}`

## 4. Persistence (SQLite in Go → D1)
Go schema (migrations 001–004): `players`, `games`, `game_participants`, `achievements`,
`player_achievements`, `game_states`, `player_actions`, `player_power_plants`,
`player_cities`, `resource_transactions`, `player_statistics`, `game_events`, views
`leaderboard`, `game_summary`, `player_game_history`, 4 triggers. **In Go nothing ever
wrote game data** (the tracking service had no callers), so the DB only held seeded
achievements. The port wires persistence for real; see `migrations/`.

Live analytics read endpoints (GET, CORS `*`, errors `{error,status,timestamp}`):
`/api/players/{name}` (+`/stats`, `/achievements`, `/performance?days`, `/competitors?days`, `/progression?days`),
`/api/leaderboard?limit`, `/api/analytics/games?days`, `/api/analytics/achievements`,
`/api/analytics/advanced?days`, `/api/analytics/activity?days`, `/api/analytics/player-types`,
`/api/analytics/maps?days`, `/api/health`. Several were stubs in Go.

Admin (`/admin/*`) was unauthenticated; the port requires a bearer token (`ADMIN_TOKEN`).
Simulated-games endpoints spawned local `./ai_client` processes — not portable to Workers;
dropped (tracked as follow-up).

## 5. Known Go bugs / gaps the port fixes (do not reproduce)
1. Engine ignored map choice; always a 4-city test map. Port loads the real map from D1.
2. Auctions could never start (`AuctionState` always nil). Port implements nomination/bidding.
3. No current/future plant market; whole 42-card deck sent, purchases never removed plants.
4. No Step 3, no winner computation, Step 2 at 7 cities (maps say 6).
5. Player order sorted fewest-cities-first (inverted). Port: most cities first, tiebreak highest plant.
6. Resource purchase charged cheapest-first but removed most-expensive tokens.
7. Income collectable repeatedly in one phase; players who skipped POWER_CITIES got nothing.
8. Lobby→game handoff: players added from a lobby had no socket bound, so engine broadcasts
   reached nobody on the `/game CONNECT` path.
9. Hybrid plants unpowerable (needed a `"Hybrid"` resource).
10. Session cleanup deadlock; `/game` session-id overwrite leak.
11. Building allowed 3 houses per city in Step 1 (should be 1 / 2 / 3 by step).
12. Discards: 4th plant auto-dropped the cheapest with no player choice.
