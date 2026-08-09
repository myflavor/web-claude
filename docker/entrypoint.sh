#!/bin/bash
set -euo pipefail

# NAS-friendly: map sandbox → host share owner via PUID/PGID.
# Defaults keep image uid 1000 / primary group of sandbox.
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

if [ "$(id -u)" -eq 0 ]; then
  # Ensure sandbox user/group match requested ids (ignore if already set).
  if getent group sandbox >/dev/null 2>&1; then
    groupmod -o -g "$PGID" sandbox 2>/dev/null || true
  else
    groupadd -o -g "$PGID" sandbox
  fi

  if id sandbox >/dev/null 2>&1; then
    usermod -o -u "$PUID" -g "$PGID" -d /home/sandbox sandbox 2>/dev/null || true
  else
    useradd -o -u "$PUID" -g "$PGID" -d /home/sandbox -s /bin/bash -m sandbox
  fi

  mkdir -p /home/sandbox/.claude /data
  # Own home always (transcripts / settings live here).
  chown -R sandbox:sandbox /home/sandbox || true
  # Best-effort on project tree; don't fail if host FS is stubborn.
  chown sandbox:sandbox /data 2>/dev/null || true

  export HOME=/home/sandbox
  export USER=sandbox
  cd /data 2>/dev/null || cd /home/sandbox

  # Drop privileges; keep env for child.
  exec gosu sandbox "$@"
fi

export HOME="${HOME:-/home/sandbox}"
exec "$@"
