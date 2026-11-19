# syntax=docker/dockerfile:1
ARG WITH_DOCKER_CLI=off
#
# Ayumi Link — single image: Go (Fiber) API + built React frontend + Claude Code CLI.
#   docker compose up -d --build

# ---- 1) Frontend build -------------------------------------------------------
FROM node:22-alpine AS web
WORKDIR /src
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# ---- 2) Backend build --------------------------------------------------------
FROM golang:1.24-alpine AS api
ENV CGO_ENABLED=0 GOTOOLCHAIN=local
WORKDIR /src
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN go build -trimpath -ldflags="-s -w" -o /out/ayumi-link .

# ---- docker CLI + compose plugin (only used when WITH_DOCKER_CLI=on, see docker-compose.host-docker.yml)
FROM scratch AS cli-off
FROM docker:27-cli AS cli-on
FROM cli-${WITH_DOCKER_CLI} AS dockercli

# ---- 3) Runtime --------------------------------------------------------------
# Debian-based Node image: the Claude Code CLI needs Node and glibc.
FROM node:22-slim

# Pin for reproducible builds, e.g. --build-arg CLAUDE_CODE_VERSION=2.1.325
ARG CLAUDE_CODE_VERSION=latest

# Chromium (headless) powers the "Browser" tab. Build without it: --build-arg WITH_BROWSER=false
ARG WITH_BROWSER=true
ARG WITH_DOCKER_CLI
# Extra tools for the Terminal tab / Claude, e.g. --build-arg EXTRA_APT_PACKAGES="jq curl python3 make"
ARG EXTRA_APT_PACKAGES=""
RUN apt-get update \
 && apt-get install -y --no-install-recommends git ca-certificates \
 && if [ "$WITH_BROWSER" = "true" ]; then apt-get install -y --no-install-recommends chromium fonts-liberation; fi \
 && if [ -n "$EXTRA_APT_PACKAGES" ]; then apt-get install -y --no-install-recommends $EXTRA_APT_PACKAGES; fi \
 && apt-get clean

# docker CLI + compose plugin talk to the host daemon through the mounted socket (opt-in).
RUN --mount=type=bind,from=dockercli,target=/cli \
    if [ -x /cli/usr/local/bin/docker ]; then \
      cp /cli/usr/local/bin/docker /usr/local/bin/docker \
      && mkdir -p /usr/local/libexec/docker/cli-plugins \
      && cp /cli/usr/local/libexec/docker/cli-plugins/docker-compose /usr/local/libexec/docker/cli-plugins/ \
      && docker --version && docker compose version; \
    fi

# The CLI ships its binary as an *optional* platform dependency, and npm skips optional
# dependencies silently when a download fails. So: retry, and fail the build loudly if
# `claude` does not actually run, instead of shipping a broken image.
RUN set -eu; \
    for i in 1 2 3; do \
      if npm install -g "@anthropic-ai/claude-code@${CLAUDE_CODE_VERSION}" && claude --version; then \
        break; \
      fi; \
      if [ "$i" = 3 ]; then echo "ERROR: Claude Code CLI could not be installed" >&2; exit 1; fi; \
      echo "Claude Code CLI install attempt $i failed; retrying..." >&2; \
      npm uninstall -g @anthropic-ai/claude-code || true; \
      sleep 5; \
    done; \
    npm cache clean --force

COPY --from=api /out/ayumi-link /usr/local/bin/ayumi-link
COPY --from=web /src/dist /app/web

# Runs as the unprivileged "node" user (uid 1000) that ships with the base image.
RUN mkdir -p /data /workspace /home/node/.claude \
 && chown -R node:node /data /workspace /home/node/.claude
USER node
WORKDIR /home/node

ENV HOST=0.0.0.0 \
    PORT=8081 \
    AYUMI_DATA_DIR=/data \
    AYUMI_ROOTS=/workspace \
    AYUMI_STATIC_DIR=/app/web \
    CLAUDE_HOME=/home/node/.claude

VOLUME ["/data", "/home/node/.claude"]
EXPOSE 8081

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD ["ayumi-link", "healthcheck"]

ENTRYPOINT ["ayumi-link"]
