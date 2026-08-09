#!/bin/bash
set -euo pipefail

# 仅在 root 启动时：按 PUID/PGID 对齐 sandbox，再降权运行。
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
  chown -R sandbox:sandbox /home/sandbox 2>/dev/null || true
  chown sandbox:sandbox /home/sandbox/.claude 2>/dev/null || true
  chown sandbox:sandbox /data 2>/dev/null || true

  echo 'sandbox ALL=(ALL) NOPASSWD:ALL' >/etc/sudoers.d/sandbox
  chmod 0440 /etc/sudoers.d/sandbox

  export HOME=/home/sandbox
  export USER=sandbox
  # 保证 sandbox 安装的 claude 在 PATH 里
  export PATH="/home/sandbox/.local/bin:${PATH}"
  cd /data 2>/dev/null || cd /home/sandbox
  exec gosu sandbox env HOME=/home/sandbox USER=sandbox PATH="/home/sandbox/.local/bin:${PATH}" "$@"
fi

export HOME="${HOME:-/home/sandbox}"
export PATH="/home/sandbox/.local/bin:${PATH}"
exec "$@"
