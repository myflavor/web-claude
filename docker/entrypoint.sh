#!/bin/bash
set -euo pipefail

# Map sandbox → host share owner (NAS). Defaults 1000/1000.
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

  mkdir -p /home/sandbox/.claude /data
  chown -R sandbox:sandbox /home/sandbox || true
  chown sandbox:sandbox /data 2>/dev/null || true

  # Ensure passwordless sudo still works after uid remap.
  echo 'sandbox ALL=(ALL) NOPASSWD:ALL' >/etc/sudoers.d/sandbox
  chmod 0440 /etc/sudoers.d/sandbox

  export HOME=/home/sandbox
  export USER=sandbox
  cd /data 2>/dev/null || cd /home/sandbox

  exec gosu sandbox "$@"
fi

export HOME="${HOME:-/home/sandbox}"
exec "$@"
