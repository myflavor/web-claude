#!/bin/bash
set -euo pipefail

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

groupmod -o -g "$PGID" sandbox
usermod -o -u "$PUID" -g "$PGID" -d /home/sandbox sandbox

mkdir -p /home/sandbox/.claude /data
chown -R sandbox:sandbox /home/sandbox
chown sandbox:sandbox /data

exec gosu sandbox env HOME=/home/sandbox \
  bash -lc 'cd /data; exec "$@"' -- "$@"
