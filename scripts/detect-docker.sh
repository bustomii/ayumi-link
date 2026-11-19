#!/bin/sh
# Finds the Docker socket and its group on THIS machine and stores them in .env
# (DOCKER_SOCK, DOCKER_GID) for docker-compose.host-docker.yml.
#   sh scripts/detect-docker.sh            # writes .env
#   sh scripts/detect-docker.sh --print    # only prints, writes nothing
set -eu

ENV_FILE="${ENV_FILE:-.env}"
PRINT_ONLY=0
[ "${1:-}" = "--print" ] && PRINT_ONLY=1

sock=""
# 1) an explicit DOCKER_HOST=unix:///path wins
case "${DOCKER_HOST:-}" in
  unix://*) sock="${DOCKER_HOST#unix://}" ;;
esac
# 2) otherwise the usual places (Linux, Docker Desktop, OrbStack, Colima, Rancher Desktop, rootless)
if [ -z "$sock" ] || [ ! -S "$sock" ]; then
  sock=""
  for c in /var/run/docker.sock \
           "$HOME/.docker/run/docker.sock" \
           "$HOME/.orbstack/run/docker.sock" \
           "$HOME/.colima/default/docker.sock" \
           "$HOME/.rd/docker.sock" \
           "${XDG_RUNTIME_DIR:-/nonexistent}/docker.sock"; do
    if [ -S "$c" ]; then sock="$c"; break; fi
  done
fi
if [ -z "$sock" ]; then
  echo "No Docker socket found on this machine. Is Docker running? (set DOCKER_HOST=unix:///path to override)" >&2
  exit 1
fi

# Group that owns the socket. On macOS the socket lives in a VM, where root (0) is the right group.
gid=0
if [ "$(uname -s)" = "Linux" ]; then
  gid="$(stat -c %g "$sock" 2>/dev/null || echo 0)"
fi

echo "DOCKER_SOCK=$sock"
echo "DOCKER_GID=$gid"
[ "$PRINT_ONLY" = 1 ] && exit 0

touch "$ENV_FILE"
for kv in "DOCKER_SOCK=$sock" "DOCKER_GID=$gid"; do
  key="${kv%%=*}"
  grep -v "^${key}=" "$ENV_FILE" > "$ENV_FILE.tmp" || true
  echo "$kv" >> "$ENV_FILE.tmp"
  mv "$ENV_FILE.tmp" "$ENV_FILE"
done
chmod 600 "$ENV_FILE" 2>/dev/null || true
echo "Saved to $ENV_FILE"
