#!/usr/bin/env bash
# Pastikan CLI `claude` ada di Mac/Linux (native build/service).
# Non-interaktif: pakai installer resmi Anthropic ke ~/.local/bin.
set -eu

export PATH="${HOME}/.local/bin:/opt/homebrew/bin:/usr/local/bin:${PATH:-}"

claude_bin=""
find_claude() {
  claude_bin=""
  if command -v claude >/dev/null 2>&1; then
    claude_bin=$(command -v claude)
  else
    for p in "${HOME}/.local/bin/claude" /usr/local/bin/claude /opt/homebrew/bin/claude; do
      if [ -x "$p" ]; then
        claude_bin=$p
        break
      fi
    done
  fi
  [ -n "$claude_bin" ] || return 1
  # Binary ada tapi rusak / wrong arch → anggap belum siap.
  "$claude_bin" --version >/dev/null 2>&1
}

if find_claude; then
  echo "[ayumi-link] claude CLI OK: $claude_bin ($("$claude_bin" --version 2>/dev/null | head -1))"
  exit 0
fi

echo "[ayumi-link] claude CLI belum ada → menginstal (https://claude.ai/install.sh)…"

if ! command -v curl >/dev/null 2>&1 && ! command -v wget >/dev/null 2>&1; then
  echo "[ayumi-link] butuh curl atau wget untuk mengunduh installer Claude" >&2
  exit 1
fi

# Installer menolak sudo dari user biasa; LaunchAgent/systemd user biasanya tanpa sudo.
if command -v curl >/dev/null 2>&1; then
  curl -fsSL https://claude.ai/install.sh | bash
else
  wget -qO- https://claude.ai/install.sh | bash
fi

export PATH="${HOME}/.local/bin:${PATH}"

if ! find_claude; then
  echo "[ayumi-link] instalasi Claude CLI gagal — cek https://code.claude.com/docs/en/setup" >&2
  exit 1
fi

echo "[ayumi-link] claude CLI terpasang: $claude_bin ($("$claude_bin" --version 2>/dev/null | head -1))"
