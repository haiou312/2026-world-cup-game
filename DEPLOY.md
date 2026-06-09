# Deployment

This is a 3-container Docker Compose app (`db` + `backend` + `web`). Anywhere you
can run Docker, you can run this — clone, bring it up, done.

---

## Prerequisites

- Docker + Docker Compose
- A free **football-data.org** API token — register at
  <https://www.football-data.org/client/register> (the free tier covers WC 2026).

---

## 1. Run it

```bash
git clone <repo> && cd 2026-world-cup-game
docker compose --profile full up -d --build
```

`.env` ships in the repo with working defaults, so there is nothing to create or
fill in first. The initial `--build` compiles the Go backend and the frontend, so
it takes a few minutes; later starts are fast.

That starts:

| Service | Container | Port |
|---------|-----------|------|
| Postgres 16 | `wc_db` | 6573 (host) |
| Go backend | `wc_backend` | internal (proxied by nginx) |
| nginx + frontend | `wc_web` | **8745 → 80** |

The backend runs DB migrations automatically on start, so there's no manual
schema step. Verify everything came up:

```bash
docker compose --profile full ps          # all three Up; wc_db shows "healthy"
curl -s localhost:8745/api/health         # {"status":"ok"}
```

Then open **http://localhost:8745** (on a remote server, `http://<server-ip>:8745`).

---

## 2. First-run setup (host)

1. Open **Settings** → enter the password (`SETTINGS_PASSWORD` in `.env`,
   default `change_me_admin`).
2. Paste your **football-data.org token** and save. The backend seeds teams +
   fixtures in the background (2 API calls) and then **refreshes once a day**.
3. Add your **participants**, then go to **Spin Wheel** to run the draw.

> Before the token is set, the Participants/Bracket pages are empty and the Round
> of 32 shows seed labels like `1E vs 3 ABCDF`. That's the expected pre-tournament
> view, **not a bug**. After saving the token, wait ~10s and refresh.

> The token is stored in the database (`app_settings`), **not** in `.env`, so it
> survives redeploys.

---

## 3. Configuration

`.env` is committed with throwaway defaults. To change anything (e.g. the host
password or DB password), edit `.env` and restart:

```bash
docker compose --profile full up -d        # re-reads .env, recreates as needed
```

Key you'll most likely change: `SETTINGS_PASSWORD`.

---

## 4. Data persistence & backups

- Postgres data lives in the named volume `db_data` — it survives
  `docker compose down` and redeploys. It is **deleted** by
  `docker compose down -v`, so avoid `-v` unless you mean it.
- Participants + draws live in the DB; teams/fixtures/standings re-sync from
  football-data, but the **draws are irreplaceable**. Simple backup:

```bash
docker exec wc_db pg_dump -U wc worldcup > backup_$(date +%F).sql
# restore:  cat backup.sql | docker exec -i wc_db psql -U wc worldcup
```

---

## 5. Updating

```bash
git pull
docker compose --profile full up -d --build      # rebuild changed images
```

Migrations apply automatically on backend start. The DB volume is untouched.

---

## 6. Hosting options

### A cheap VPS (recommended — easiest)

A small VPS (Hetzner CX22 ~€4/mo, DigitalOcean/Vultr ~$5/mo, or Oracle Cloud
Always-Free ARM for $0) runs this compose file unchanged:

```bash
# on the server, with Docker installed
git clone <repo> && cd 2026-world-cup-game
docker compose --profile full up -d --build
```

It's then reachable at `http://<server-ip>:8745`. `restart: always` keeps it
running across reboots.

### fly.io

fly.io can run a Compose file on a single Machine (Multi-container Machines), with
two caveats for this app:

- Exactly one service may use `build:` — pre-build the others and reference images.
- **Compose `volumes:` are ignored** — Postgres data must be on a Fly Volume
  mounted via `[mounts]` in `fly.toml`, or it's wiped on every deploy.

For a one-machine deploy a plain VPS is simpler; reach for fly.io if you want its
edge/scaling.

---

## 7. How sync works (ops note)

- The backend runs an **hourly** cron (`@hourly`) that does a **daily-scoped**
  sync: it pulls the schedule + results, keeps refreshing until every match that
  day is final, then no-ops until the next day.
- A full refresh = **2 API calls** (`/matches` + `/standings`), so daily usage is
  tiny vs. the free tier (100/day).
- Sync health (last success, last error, warnings) is visible in **Settings →
  Sync status**. You can also trigger a manual full sync there.

---

## Troubleshooting

| Symptom | Check |
|---------|-------|
| Settings/Wheel rejects the password | `SETTINGS_PASSWORD` in `.env`; restart backend after changing it |
| Bracket/teams empty | token set in Settings? hit **Sync now**; check **Sync status** for errors |
| `8745` already in use | change the `web` port mapping in `docker-compose.yml` |
| Want a clean slate | **Settings → Reset all draws** (keeps participants), or delete participants |
