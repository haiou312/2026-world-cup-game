# 2026 World Cup Game 🏆

A party-style elimination game around the **FIFA World Cup 2026**. The host adds
the players, spins two wheels to randomly assign each player a national team, and
then everyone follows the real tournament — players are knocked out as their team
loses, until a champion remains.

No accounts, no passwords for players. Everything is public to view; only the
host-only **Settings** and **Spin Wheel** sections sit behind one shared password.

---

## How to play

1. **Host** opens **Settings** (enters the shared password) and adds the participants.
2. **Host** opens **Spin Wheel**: spin the left wheel to pick a person, then the
   right wheel to draw them a random country. Both disappear from the wheels once
   paired. (Teams are unique within each round of 48; the country pool refreshes
   after all 48 are drawn.)
3. **Everyone** opens **Participants**, searches their name, and taps it to see
   their team, group, fixtures and how far they've gone.
4. **Bracket** shows the live group standings and the knockout tree, converging
   on the eventual champion.

Real match data comes from [football-data.org](https://www.football-data.org)
and is refreshed by a daily background sync (no live polling).

---

## Pages

| Page | Route | Access |
|------|-------|--------|
| Participants (search → a player's team) | `/` | public |
| Tournament bracket + group standings | `/bracket` | public |
| Spin Wheel (the draw) | `/wheel` | host password |
| Settings (participants, data token, sync) | `/settings` | host password |

UI is English by default with a 中文 toggle.

---

## Tech stack

- **Frontend** — Vue 3 + Vite + TypeScript, Pinia, vue-router, vue-i18n
- **Backend** — Go (chi router, pgx/pgxpool, [sqlc](https://sqlc.dev), goose
  migrations, robfig/cron, log/slog)
- **Database** — PostgreSQL 16
- **Web** — nginx (serves the built frontend, proxies `/api` → backend)
- **Data** — football-data.org v4 API, synced daily into the DB
- **Packaging** — Docker Compose (`db` + `backend` + `web`)

---

## Quick start

```bash
git clone <repo> && cd 2026-world-cup-game
docker compose --profile full up -d --build
# open http://localhost:8745
```

First run: open **Settings** (password is `SETTINGS_PASSWORD` from `.env`), paste
a football-data.org API token — the backend seeds teams/fixtures in the
background and refreshes daily after that.

See **[DEPLOY.md](DEPLOY.md)** for hosting (VPS / fly.io), the data token, and ops.

---

## Configuration (`.env`)

`.env` is committed (this is a small game with throwaway credentials — change them
if you care). Keys:

| Key | What |
|-----|------|
| `POSTGRES_USER` / `POSTGRES_PASSWORD` / `POSTGRES_DB` | database |
| `DATABASE_URL` | backend → Postgres (compose overrides host to `db`) |
| `PORT` | backend HTTP port (8080) |
| `SETTINGS_PASSWORD` | password for the Settings + Spin Wheel sections |
| `FD_COMPETITION` | football-data competition code (`WC`) |

The football-data.org **token is NOT in `.env`** — it's entered in Settings and
stored in the DB (`app_settings`).

---

## Local development

```bash
# 1. a Postgres to talk to (exposes 6573 on the host)
docker compose up -d db

# 2. backend — load env from the repo .env, then run (serves on :8080)
cd backend
set -a; source ../.env; set +a
go run ./cmd/server

# 3. frontend — Vite dev server, proxies /api → :8080
cd frontend && npm install && npm run dev   # http://localhost:5173
```

> The committed `.env` is consumed by docker-compose; a bare `go run` from
> `backend/` doesn't pick it up automatically, hence the `source ../.env`.

A local **tournament simulator** fills the DB day-by-day for testing without the
real API:

```bash
./sim.sh reset          # back to pre-tournament
./sim.sh through 26     # advance the DB to "day 26"
./sim.sh full           # jump to the final / champion
```

---

## Tests

```bash
# backend unit tests (fast, no Docker)
cd backend && go test ./...

# backend integration tests (testcontainers — needs Docker)
TESTCONTAINERS_RYUK_DISABLED=true go test -tags integration ./...

# frontend tests
cd frontend && npm test
```

Covered: draw concurrency + round reset, sync upsert pipeline, the R32 skeleton,
**knockout resolution (anti-scramble)**, champion/elimination logic, team-name
matching, the participants/assign API + password gate, and a few frontend units
(spin reel reset, search filter, stage labels).

---

## API

Public reads (no auth):

```
GET  /api/health
GET  /api/teams
GET  /api/bracket
GET  /api/participants
GET  /api/fixtures
```

Host-only (header `X-Settings-Password: <SETTINGS_PASSWORD>`):

```
POST   /api/settings/verify
POST   /api/participants            {name}
DELETE /api/participants/{id}
POST   /api/assign                  {participant_id}   # the country spin
POST   /api/reset                                      # clear all draws
GET    /api/settings/sync-status
PUT    /api/settings/api-key        {key}              # football-data token
POST   /api/settings/sync                              # manual full sync
```

---

## Project layout

```
backend/
  cmd/server/        HTTP server entrypoint
  cmd/simulate/      local tournament simulator
  internal/
    config/          env config
    db/              pgx pool, goose migration runner, sqlc-generated code
    domain/          draw, progress, R32 skeleton, stages
    handlers/        bracket, participants, assign, settings
    sync/            football-data client, scheduler, upsert
    testsupport/     testcontainers helpers
  migrations/        goose SQL migrations
  queries/           sqlc query files
frontend/
  src/views/         Home, Bracket, Wheel, Settings
  src/components/     SpinReel, KnockoutBracket, GroupStandings, MatchRow, ...
  src/stores/        gate (settings password)
  src/i18n/          en + zh
  src/api/, src/lib/
nginx/nginx.conf
docker-compose.yml
sim.sh
```

---

## How the bracket stays correct

Match results, scores and group positions all come straight from football-data —
we don't compute them. The fixed official 2026 Round-of-32 structure is the one
hardcoded piece (`domain.R32Skeleton`, asserted by a test); from the Round of 16
onward each tie's two teams are read directly off the real fixtures, so the tree
never scrambles. See the anti-scramble unit test in `internal/handlers`.
