#!/usr/bin/env bash
# Entry point launchd / systemd: setiap start/restart = make build (FE+BE) lalu serve.
set -eu
ROOT=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
cd "$ROOT"

export PATH="/opt/homebrew/bin:/usr/local/go/bin:/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin:/snap/bin:${PATH:-}"

# Node via nvm / fnm / volta bila ada (service tidak memuat shell profile).
if [ -d "${HOME}/.nvm/versions/node" ]; then
  # Hanya rilis genap (LTS) yang stabil; versi alpha/nightly dan ganjil dilewati.
  NVM_NODE=$(ls -1d "${HOME}/.nvm/versions/node"/v* 2>/dev/null | grep -E '/v(20|22|24|26)\.[0-9]+\.[0-9]+$' | sort -V | tail -1 || true)
  if [ -n "${NVM_NODE:-}" ] && [ -x "${NVM_NODE}/bin/node" ]; then
    export PATH="${NVM_NODE}/bin:${PATH}"
  fi
fi
if [ -d "${HOME}/.fnm/node-versions" ]; then
  FNM_NODE=$(ls -1d "${HOME}/.fnm/node-versions"/v*/installation/bin 2>/dev/null | sort -V | tail -1 || true)
  if [ -n "${FNM_NODE:-}" ]; then
    export PATH="${FNM_NODE}:${PATH}"
  fi
fi
if [ -d "${HOME}/.volta/bin" ]; then
  export PATH="${HOME}/.volta/bin:${PATH}"
fi
if [ -d "${HOME}/go/bin" ]; then
  export PATH="${HOME}/go/bin:${PATH}"
fi
if [ -d "${HOME}/.local/bin" ]; then
  export PATH="${HOME}/.local/bin:${PATH}"
fi

command -v make >/dev/null || { echo "make tidak ada di PATH" >&2; exit 1; }
command -v go >/dev/null || { echo "go tidak ada di PATH (Homebrew / /usr/local/go?)" >&2; exit 1; }
command -v node >/dev/null || { echo "node tidak ada di PATH (nvm/fnm?)" >&2; exit 1; }
command -v npm >/dev/null || { echo "npm tidak ada di PATH" >&2; exit 1; }

echo "[ayumi-link] service start → make build ($(date -u +%Y-%m-%dT%H:%M:%SZ))"
echo "[ayumi-link] ROOT=$ROOT go=$(command -v go) node=$(command -v node)"
exec make build
