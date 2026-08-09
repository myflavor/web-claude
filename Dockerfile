# syntax=docker/dockerfile:1

# —— Frontend (Vue3 + Vite) ——
FROM node:22-bookworm-slim AS frontend
WORKDIR /src/web/ui
COPY web/ui/package.json web/ui/package-lock.json* ./
RUN npm install
COPY web/ui/ ./
RUN npm run build

# —— Backend (static binary) ——
FROM golang:1.22-bookworm AS builder
WORKDIR /src
ENV GOTOOLCHAIN=local
ARG VERSION=dev
ARG COMMIT=unknown
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web/embed.go ./web/embed.go
COPY --from=frontend /src/web/static ./web/static
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath \
    -ldflags="-s -w -X main.version=${VERSION} -X main.commit=${COMMIT}" \
    -o /out/web-claude ./cmd/server

# —— Runtime: Debian + git + Claude Code（默认非 root；入口可用 PUID/PGID）——
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      bash \
      git \
      gosu \
    && rm -rf /var/lib/apt/lists/*

# Build as root: install Claude Code binary onto system PATH.
# Runtime config lives under /home/sandbox/.claude (HOME).
RUN set -eux; \
    curl -fsSL https://claude.ai/install.sh | bash; \
    CLAUDE_SRC=""; \
    for c in /root/.local/bin/claude /root/.claude/local/bin/claude /root/.claude/local/claude; do \
      if [ -e "$c" ]; then CLAUDE_SRC="$c"; break; fi; \
    done; \
    if [ -z "$CLAUDE_SRC" ]; then CLAUDE_SRC="$(command -v claude)"; fi; \
    CLAUDE_REAL="$(readlink -f "$CLAUDE_SRC")"; \
    install -m 0755 "$CLAUDE_REAL" /usr/local/bin/claude; \
    command -v claude; \
    claude --version

COPY --from=builder /out/web-claude /usr/local/bin/web-claude
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

# Default identity (overridden at start by PUID/PGID when set).
RUN useradd -m -u 1000 -d /home/sandbox -s /bin/bash sandbox \
    && mkdir -p /data /home/sandbox/.claude \
    && chown -R sandbox:sandbox /data /home/sandbox

ENV HOME=/home/sandbox \
    WEB_CLAUDE_ROOT=/data \
    WEB_CLAUDE_PORT=3080 \
    RUN_MODE=docker \
    PUID=1000 \
    PGID=1000

# Entrypoint starts as root only long enough to map uid/gid, then gosu → sandbox.
USER root
WORKDIR /data
EXPOSE 3080

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["web-claude"]
