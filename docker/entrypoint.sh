#!/bin/bash
set -euo pipefail

# Classic PUID/PGID: remap sandbox, chown mounts, drop privileges,
# then login-shell once so ~/.profile + ~/.bashrc enter web-claude env.
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

if [ "$(id -u)" -ne 0 ]; then
  echo "[web-claude] entrypoint must start as root (do not set compose user:)" >&2
  exit 1
fi

if getent group sandbox >/dev/null 2>&1; then
  cur_gid="$(getent group sandbox | cut -d: -f3)"
  if [ "$cur_gid" != "$PGID" ]; then
    groupmod -o -g "$PGID" sandbox
  fi
else
  groupadd -o -g "$PGID" sandbox
fi

if id sandbox >/dev/null 2>&1; then
  cur_uid="$(id -u sandbox)"
  cur_gid="$(id -g sandbox)"
  if [ "$cur_uid" != "$PUID" ] || [ "$cur_gid" != "$PGID" ]; then
    usermod -o -u "$PUID" -g "$PGID" -d /home/sandbox sandbox
  fi
else
  useradd -o -u "$PUID" -g "$PGID" -d /home/sandbox -s /bin/bash -m sandbox
fi

mkdir -p /home/sandbox/.claude /data
# Paths that matter: home (profile/local), mounted .claude, project root.
chown -R sandbox:sandbox /home/sandbox 2>/dev/null || true
chown sandbox:sandbox /data 2>/dev/null || true

# Drop to sandbox, load profile/bashrc once, exec app (web-claude).
exec gosu sandbox env HOME=/home/sandbox USER=sandbox \
  bash -lc 'cd /data 2>/dev/null || true; exec "$@"' -- "$@"
