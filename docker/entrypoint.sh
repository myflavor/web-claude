#!/bin/bash
set -euo pipefail

# NAS-friendly: run as numeric PUID:PGID without usermod/groupmod.
# HOME stays /home/sandbox so .profile/.bashrc and ~/.claude paths stay stable.
PUID="${PUID:-1000}"
PGID="${PGID:-1000}"

if [ "$(id -u)" -eq 0 ]; then
  mkdir -p /home/sandbox /home/sandbox/.claude /home/sandbox/.local/bin /data

  # Shell rc must exist for bash -lc (sessions load these).
  [ -f /home/sandbox/.profile ] || printf '%s\n' \
    'export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"' \
    > /home/sandbox/.profile
  [ -f /home/sandbox/.bashrc ] || printf '%s\n' \
    'export PATH="$HOME/.local/bin:/usr/local/bin:/usr/bin:/bin:$PATH"' \
    > /home/sandbox/.bashrc

  # Fix ownership for this uid/gid (no /etc/passwd rewrite).
  chown -R "${PUID}:${PGID}" /home/sandbox 2>/dev/null || true
  chown "${PUID}:${PGID}" /data 2>/dev/null || true

  # Passwordless sudo for this numeric uid (apt install in session).
  echo "Defaults:#!requiretty" >/etc/sudoers.d/web-claude-notty 2>/dev/null || true
  echo "# web-claude runtime" >/etc/sudoers.d/web-claude-puid
  echo "User_Alias WEBCLAUDE = #${PUID}" >>/etc/sudoers.d/web-claude-puid
  echo "WEBCLAUDE ALL=(ALL) NOPASSWD:ALL" >>/etc/sudoers.d/web-claude-puid
  chmod 0440 /etc/sudoers.d/web-claude-puid

  export HOME=/home/sandbox
  export USER="${USER:-sandbox}"
  # Base PATH for Go (web-claude); sessions still re-load profile via bash -lc.
  export PATH="/usr/local/bin:/usr/bin:/bin${PATH:+:$PATH}"
  cd /data 2>/dev/null || cd /home/sandbox

  exec gosu "${PUID}:${PGID}" env HOME=/home/sandbox \
    PATH="/usr/local/bin:/usr/bin:/bin" \
    "$@"
fi

export HOME="${HOME:-/home/sandbox}"
export PATH="/usr/local/bin:/usr/bin:/bin${PATH:+:$PATH}"
exec "$@"
