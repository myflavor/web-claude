# syntax=docker/dockerfile:1

# —— Frontend ——
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

# —— Runtime ——
FROM debian:bookworm-slim

# Keep sbin on PATH: useradd/groupmod/usermod live in /usr/sbin.
ENV DEBIAN_FRONTEND=noninteractive \
    HOME=/home/sandbox \
    WEB_CLAUDE_ROOT=/data \
    WEB_CLAUDE_PORT=3080 \
    RUN_MODE=docker \
    PUID=1000 \
    PGID=1000 \
    PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

RUN apt-get update && apt-get install -y --no-install-recommends \
      ca-certificates curl bash git gosu sudo passwd \
    && useradd -m -u 1000 -d /home/sandbox -s /bin/bash sandbox \
    && mkdir -p /data /home/sandbox/.claude /home/sandbox/.local/bin \
    && touch /home/sandbox/.profile /home/sandbox/.bashrc \
    && chown -R sandbox:sandbox /home/sandbox /data \
    && echo 'sandbox ALL=(ALL) NOPASSWD:ALL' >/etc/sudoers.d/sandbox \
    && chmod 0440 /etc/sudoers.d/sandbox \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/web-claude /usr/local/bin/web-claude
COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

# Install Claude as sandbox (may write ~/.local and patch profile/bashrc).
USER sandbox
WORKDIR /home/sandbox
ENV PATH=/home/sandbox/.local/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin
RUN curl -fsSL https://claude.ai/install.sh | bash

# Pin claude on system PATH so Go always finds it even if home layout changes.
USER root
RUN set -eux; \
    CLAUDE_SRC="$(su -s /bin/bash sandbox -c 'command -v claude')"; \
    test -n "$CLAUDE_SRC"; \
    install -m 0755 "$(readlink -f "$CLAUDE_SRC")" /usr/local/bin/claude; \
    command -v claude; \
    claude --version; \
    command -v git

# Runtime default PATH: system tools first; user-local tools still available.
ENV PATH=/usr/local/bin:/home/sandbox/.local/bin:/usr/local/sbin:/usr/sbin:/usr/bin:/sbin:/bin

WORKDIR /data
EXPOSE 3080
ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
CMD ["web-claude"]
