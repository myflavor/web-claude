#!/bin/bash
set -euo pipefail

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

groupmod -o -g "$PGID" sandbox
usermod -o -u "$PUID" -g "$PGID" -d /home/sandbox sandbox

mkdir -p /home/sandbox/.claude /data
chown -R sandbox:sandbox /home/sandbox
chown sandbox:sandbox /data

# Fix ~/.ssh perms so git-over-SSH works. Only touch if it exists
# (mounted volume may not have it yet).
if [ -d /home/sandbox/.ssh ]; then
  chown -R sandbox:sandbox /home/sandbox/.ssh
  chmod 700 /home/sandbox/.ssh
  # private keys 600; keep 644 for .pub / config
  find /home/sandbox/.ssh -maxdepth 1 -type f ! -name '*.pub' -exec chmod 600 {} + 2>/dev/null || true
  find /home/sandbox/.ssh -maxdepth 1 -type f -name '*.pub' -exec chmod 644 {} + 2>/dev/null || true
fi

exec gosu sandbox bash -lc 'cd /data; exec "$@"' -- "$@"
