#!/usr/bin/env bash
# Tournament simulator driver. Usage:
#   ./sim.sh list            # show the 35 match days with their index
#   ./sim.sh through 5       # advance the DB to day 5 (cumulative)
#   ./sim.sh reset           # back to pre-tournament
#   ./sim.sh full            # jump to the final / champion
# After each command, reload http://localhost:8745/bracket
set -e
cd "$(dirname "$0")"
export DATABASE_URL="${DATABASE_URL:-postgres://wc:change_me_pg@localhost:6573/worldcup?sslmode=disable}"
BIN=/tmp/wc_sim
if [ ! -x "$BIN" ]; then
  echo "building simulator…"
  (cd backend && go build -o "$BIN" ./cmd/simulate)
fi
exec "$BIN" "$@"
