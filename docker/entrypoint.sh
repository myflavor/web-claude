#!/bin/bash
set -euo pipefail

# NAS: run as numeric PUID:PGID (no usermod).
# Load ~/.profile + ~/.bashrc once via login shell, then exec web-claude so
# Go children inherit that environment (PATH, exports). No bash -lc in Go.
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

if [ "$(id -u)" -eq 0 ]; then
  mkdir -p /home/sandbox /home/sandbox/.claude /home/sandbox/.local/bin /data

  [ -f /home/sandbox/.profile ] || printf '%s\n' \
    'export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"' \
    > /home/sandbox/.profile
  [ -f /home/sandbox/.bashrc ] || printf '%s\n' \
    'export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"' \
    > /home/sandbox/.bashrc

  chown -R "${PUID}:${PGID}" /home/sandbox 2>/dev/null || true
  chown "${PUID}:${PGID}" /data 2>/dev/null || true

  echo "# web-claude runtime" >/etc/sudoers.d/web-claude-puid
  echo "User_Alias WEBCLAUDE = #${PUID}" >>/etc/sudoers.d/web-claude-puid
  echo "WEBCLAUDE ALL=(ALL) NOPASSWD:ALL" >>/etc/sudoers.d/web-claude-puid
  chmod 0440 /etc/sudoers.d/web-claude-puid

  cd /data 2>/dev/null || cd /home/sandbox

  # Drop privileges, then login-shell to load profile/bashrc, then exec app.
  # "$@" is normally: web-claude
  exec gosu "${PUID}:${PGID}" env HOME=/home/sandbox \
    bash -lc 'export HOME=/home/sandbox; cd /data 2>/dev/null || true; exec "$@"' -- "$@"
fi

# Non-root re-entry: still load profile then exec.
export HOME="${HOME:-/home/sandbox}"
exec bash -lc 'export HOME="${HOME:-/home/sandbox}"; cd /data 2>/dev/null || true; exec "$@"' -- "$@"
