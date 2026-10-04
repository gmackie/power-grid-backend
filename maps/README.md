# Gameboards

Map assets consumed by `cf_server` (seeded into D1 `maps`) and rendered to art by
`artgen/boards.mjs`. Same mechanics on every board; twists live in topology and `gameRules`.

| id | origin | twist |
|---|---|---|
| `usa`, `germany` | original boards | reference balance |
| `archipelago` | `tools/gen-map.mjs` | five islands, cheap local grids, 18-25 ferry links, $60 start, lower city targets |
| `megalopolis` | `tools/gen-map.mjs` | 10-city core ring at cost 0-4 ringed by sprawl, step 2 at 8 cities, more garbage |
| `frontier` | `tools/gen-map.mjs` | costs rise west→east, $45 start, uranium 6 / coal 20 |

Workflow for a new or rebalanced board:

```bash
node maps/tools/gen-map.mjs                   # edit DESIGNS, regenerate JSON (seeded, reproducible)
cd cf_server && node scripts/build-seed.ts     # refresh 0002_seed.sql (fresh databases)
#   existing databases: add an idempotent migration like migrations/0003_custom_boards.sql
pnpm db:migrate:local && pnpm db:migrate:remote
pnpm sim -- --games 40 --evolve 6              # board lab report in cf_server/sim/reports/
cd ../../artgen && node boards.mjs             # board art → love_client/assets, react_client/public/art
```

Board lab metrics (see `cf_server/src/sim/run.ts`): finish rate, rounds, lead changes, win
margin, contested cities, auction premium, step-3 rate, win rate per bot archetype, region
share of houses, market scarcity, and a composite **interest** score (archetype balance ×
closeness × drama × region spread). `--evolve N` hill-climbs the heuristic bot's weights per
board and reports the strategy it converges on and how it fares against the archetypes.
