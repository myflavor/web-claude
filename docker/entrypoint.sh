#!/bin/bash
set -euo pipefail

PUID="${PUID:-1000}"
PGID="${PGID:-1000}"
case "$PUID" in *[!0-9]*) PUID=1000 ;; esac
case "$PGID" in *[!0-9]*) PGID=1000 ;; esac

HOME_DIR=/home/claude
CLAUDE_DIR="${HOME_DIR}/.claude"
SETTINGS_SRC="${WEB_CLAUDE_SETTINGS:-/settings.json}"
SETTINGS_DST="${CLAUDE_DIR}/settings.json"

apply_settings() {
  mkdir -p "${CLAUDE_DIR}"
  if [ -f "${SETTINGS_SRC}" ]; then
    cp -f "${SETTINGS_SRC}" "${SETTINGS_DST}"
    chmod 600 "${SETTINGS_DST}" 2>/dev/null || true
  elif [ -d "${SETTINGS_SRC}" ]; then
    echo "[web-claude] ${SETTINGS_SRC} is a directory — host settings.json missing (Docker created a dir)." >&2
  fi
}

if [ "$(id -u)" -eq 0 ]; then
  # Align image user "claude" to NAS PUID/PGID (linuxserver-style).
  if getent group "${PGID}" >/dev/null 2>&1; then
    :
  else
    groupmod -o -g "${PGID}" claude 2>/dev/null || groupadd -o -g "${PGID}" claude
  fi
  usermod -o -u "${PUID}" -g "${PGID}" -d "${HOME_DIR}" -s /bin/bash claude 2>/dev/null || true

  mkdir -p "${CLAUDE_DIR}" /data
  chown -R "${PUID}:${PGID}" "${HOME_DIR}" 2>/dev/null || true
  # /data may be a large NAS share; only ensure top-level writable when possible
  chown "${PUID}:${PGID}" /data 2>/dev/null || true

  apply_settings
  chown "${PUID}:${PGID}" "${SETTINGS_DST}" 2>/dev/null || true

  export HOME="${HOME_DIR}"
  export USER=claude
  cd /data 2>/dev/null || cd /

  if command -v setpriv >/dev/null 2>&1; then
    exec setpriv --reuid="${PUID}" --regid="${PGID}" --clear-groups --inh-caps=-all \
      web-claude "$@"
  fi
  exec su -s /bin/bash claude -c "export HOME=${HOME_DIR}; exec web-claude \"\$@\"" -- "$@"
fi

# Non-root start (compose user: ...). Cannot fix ownership; best-effort only.
export HOME="${HOME:-$HOME_DIR}"
apply_settings || true
exec web-claude "$@"
