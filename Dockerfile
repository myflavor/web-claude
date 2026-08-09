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

# —— Runtime: Debian + git + Claude Code only ——
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates \
      curl \
      bash \
      git \
    && rm -rf /var/lib/apt/lists/*

# Official Claude Code installer → real binary on PATH.
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

RUN useradd -m -u 1000 -s /bin/bash claude \
    && mkdir -p /data /home/claude/.claude \
    && chown -R claude:claude /data /home/claude

ENV HOME=/home/claude \
    WEB_CLAUDE_ROOT=/data \
    WEB_CLAUDE_PORT=3080 \
    RUN_MODE=docker

USER claude
WORKDIR /data
EXPOSE 3080

ENTRYPOINT ["web-claude"]
