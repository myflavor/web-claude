#!/bin/bash
set -euo pipefail

# Classic PUID/PGID: remap sandbox user, fix ownership, drop to sandbox.
# Then login-shell once so ~/.profile + ~/.bashrc enter web-claude's env.
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

if [ "$(id -u)" -eq 0 ]; then
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

  mkdir -p /home/sandbox /home/sandbox/.claude /home/sandbox/.local/bin /data
  [ -f /home/sandbox/.profile ] || touch /home/sandbox/.profile
  [ -f /home/sandbox/.bashrc ] || touch /home/sandbox/.bashrc

  # Only the paths that matter for NAS mounts + home.
  chown -R sandbox:sandbox /home/sandbox 2>/dev/null || true
  chown sandbox:sandbox /home/sandbox/.claude 2>/dev/null || true
  chown sandbox:sandbox /data 2>/dev/null || true

  echo 'sandbox ALL=(ALL) NOPASSWD:ALL' >/etc/sudoers.d/sandbox
  chmod 0440 /etc/sudoers.d/sandbox

  export HOME=/home/sandbox
  export USER=sandbox
  cd /data 2>/dev/null || cd /home/sandbox

  # Drop to sandbox, load profile/bashrc, exec app (usually web-claude).
  exec gosu sandbox env HOME=/home/sandbox USER=sandbox \
    bash -lc 'export HOME=/home/sandbox; cd /data 2>/dev/null || true; exec "$@"' -- "$@"
fi

export HOME="${HOME:-/home/sandbox}"
export USER="${USER:-sandbox}"
exec bash -lc 'export HOME="${HOME:-/home/sandbox}"; cd /data 2>/dev/null || true; exec "$@"' -- "$@"
