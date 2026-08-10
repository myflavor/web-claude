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
# Order matters: globs would also match .pub, so set .pub to 644 first,
# then everything else to 600 (find excludes *.pub).
if [ -d /home/sandbox/.ssh ]; then
  chmod 700 /home/sandbox/.ssh
  chmod 644 /home/sandbox/.ssh/*.pub 2>/dev/null || true
  find /home/sandbox/.ssh -maxdepth 1 -type f ! -name '*.pub' -exec chmod 600 {} + 2>/dev/null || true
fi

exec gosu sandbox bash -lc 'cd /data; exec "$@"' -- "$@"
