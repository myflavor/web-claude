# syntax=docker/dockerfile:1

# —— Frontend (Vue3 + Vite) ——
FROM node:22-bookworm-slim AS frontend
WORKDIR /src/web/ui
COPY web/ui/package.json web/ui/package-lock.json* ./
RUN npm install
COPY web/ui/ ./
RUN npm run build

# —— Backend ——
FROM golang:1.22-bookworm AS builder
WORKDIR /src
ENV GOTOOLCHAIN=local
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
COPY web/embed.go ./web/embed.go
COPY --from=frontend /src/web/static ./web/static
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/claude-mobile ./cmd/server

# —— Runtime (Claude CLI + binary) ——
FROM node:22-bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl ca-certificates bash openssh-client python3 \
    && rm -rf /var/lib/apt/lists/*

RUN npm install -g @anthropic-ai/claude-code

COPY --from=builder /out/claude-mobile /usr/local/bin/claude-mobile

RUN mkdir -p /data/projects /data/home \
    && chown -R node:node /data

ENV HOME=/data/home \
    WEB_CLAUDE_ROOT=/data/projects \
    CLAUDE_HOME=/data/home \
    HOME_DIR=/data/home \
    WEB_CLAUDE_PORT=3080 \
    RUN_MODE=docker

USER node
WORKDIR /data/projects
EXPOSE 3080

ENTRYPOINT ["claude-mobile"]
