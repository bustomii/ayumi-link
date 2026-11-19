#!/usr/bin/env sh
# Native development: backend :8081 + Vite :5173 (satu perintah).
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

cd frontend
if [ ! -d node_modules ]; then
  npm install
fi
cd "$ROOT"

BE_PID=
FE_PID=
cleanup() {
  if [ -n "${FE_PID:-}" ]; then kill "$FE_PID" 2>/dev/null || true; fi
  if [ -n "${BE_PID:-}" ]; then kill "$BE_PID" 2>/dev/null || true; fi
}
trap cleanup INT TERM EXIT

echo "Backend  → http://127.0.0.1:8081"
echo "Frontend → http://localhost:5173  (proxy /api → :8081)"
echo "Ctrl+C to stop both."
echo ""

(cd "$ROOT/backend" && go run .) &
BE_PID=$!
(cd "$ROOT/frontend" && npm run dev) &
FE_PID=$!

wait
