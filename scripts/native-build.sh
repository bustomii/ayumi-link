#!/usr/bin/env sh
# Native production-style: build FE + BE binary, serve on :8081.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

if [ ! -f .env ]; then
  cp .env.example .env
  echo "Created .env from .env.example"
fi

echo "==> Frontend build"
cd frontend
if [ ! -d node_modules ]; then
  npm install
else
  npm install --prefer-offline --no-audit --no-fund >/dev/null
fi
npm run build
cd "$ROOT"

echo "==> Backend build"
cd backend
go build -trimpath -ldflags="-s -w" -o ayumi-link .
cd "$ROOT"

echo ""
echo "Ayumi Link (native) → http://127.0.0.1:8081"
echo "Ctrl+C to stop."
echo ""

cd backend
export AYUMI_STATIC_DIR="${AYUMI_STATIC_DIR:-../frontend/dist}"
exec ./ayumi-link
