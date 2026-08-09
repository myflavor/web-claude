#!/bin/bash
set -euo pipefail

HOME_DIR="${HOME:-/home/claude}"
CLAUDE_DIR="${HOME_DIR}/.claude"
SETTINGS_DST="${CLAUDE_DIR}/settings.json"
SETTINGS_SRC="${WEB_CLAUDE_SETTINGS:-/settings.json}"

mkdir -p "${CLAUDE_DIR}"

# Prefer an explicitly mounted settings file. Copy into place so Claude can
# read/write without fighting a bind-mounted file (atomic rename / chmod).
if [ -f "${SETTINGS_SRC}" ]; then
  cp -f "${SETTINGS_SRC}" "${SETTINGS_DST}"
  chmod 600 "${SETTINGS_DST}" 2>/dev/null || true
elif [ -d "${SETTINGS_SRC}" ]; then
  echo "[web-claude] ${SETTINGS_SRC} is a directory — host path missing? create a settings.json file" >&2
fi

exec web-claude "$@"
