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
# (mounted volume may not have it yet). The recursive chown above already
# covers ownership; only the modes are new here.
# id_* = private keys 600; id_*.pub = public 644; known_hosts(+old) = 644.
if [ -d /home/sandbox/.ssh ]; then
  chmod 700 /home/sandbox/.ssh
  chmod 600 /home/sandbox/.ssh/id_* 2>/dev/null || true
  chmod 644 /home/sandbox/.ssh/id_*.pub /home/sandbox/.ssh/known_hosts /home/sandbox/.ssh/known_hosts.old 2>/dev/null || true
fi

exec gosu sandbox bash -lc 'cd /data; exec "$@"' -- "$@"
