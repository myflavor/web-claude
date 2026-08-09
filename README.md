# Web Claude

用浏览器远程使用 Claude Code（PC / 手机）。  
两种用法：**单二进制**、**Docker Compose**。

默认地址：`http://<主机>:3080`  
Compose 默认登录密码：`password`

---

## 方式一：单二进制

从 [Releases](https://github.com/myflavor/web-claude/releases) 下载对应平台文件，例如：

- Linux amd64：`web-claude_vX.Y.Z_linux_amd64`
- macOS arm64：`web-claude_vX.Y.Z_darwin_arm64`
- Windows：`web-claude_vX.Y.Z_windows_amd64.exe`

```bash
chmod +x web-claude_vX.Y.Z_linux_amd64
export WEB_CLAUDE_TOKEN=password
# 可选：
# export WEB_CLAUDE_PORT=3080          # 默认 3080
# export WEB_CLAUDE_ROOT=$HOME         # 默认可浏览用户家目录
./web-claude_vX.Y.Z_linux_amd64
```

浏览器打开 `http://127.0.0.1:3080`，用 `WEB_CLAUDE_TOKEN` 登录。

本机需已安装并可直接运行 `claude`（使用真实用户的 `~/.claude` 配置）。

---

## 方式二：Docker Compose

准备：

```bash
mkdir -p data claude
# 可选：Claude 配置（放在挂载目录里，不要单独挂单文件）
cat > claude/settings.json <<'JSON'
{
  "permissions": {
    "deny": ["WebSearch", "WebFetch"]
  }
}
JSON
```

`docker-compose.yml`：

```yaml
services:
  web-claude:
    image: ghcr.io/myflavor/web-claude:latest
    ports:
      - 3080:3080
    environment:
      WEB_CLAUDE_TOKEN: password
      WEB_CLAUDE_PORT: 3080
      WEB_CLAUDE_ROOT: /data
      # NAS：对齐共享属主（ls -ln）。本机默认 1000 即可。
      PUID: 1000
      PGID: 1000
    volumes:
      - ./data:/data
      # 整个 ~/.claude：settings、会话 transcript、历史
      - ./claude:/home/sandbox/.claude
    restart: unless-stopped
```

启动：

```bash
# 私有包需先：docker login ghcr.io
docker compose pull
docker compose up -d
```

### 为什么要挂整个 `.claude`

会话记录、resume、项目历史都在 Claude 的配置目录里（`$HOME/.claude`）。  
只挂 `settings.json` 时，重建容器会丢掉 transcript。  
挂 **`./claude → /home/sandbox/.claude`** 后，配置和历史都在宿主机上，容器重建不丢。

| 宿主机 | 容器 | 用途 |
|--------|------|------|
| `./data` | `/data` | 项目目录 |
| `./claude` | `/home/sandbox/.claude` | settings、会话、历史等 |

### 身份与装软件

- 业务进程：**非 root** 用户 `sandbox`（可用 `PUID`/`PGID` 对齐 NAS）
- `HOME=/home/sandbox`
- **免密 sudo**：

```bash
sudo apt-get update
sudo apt-get install -y python3 build-essential
```

**不要**写 compose `user:`（会跳过入口）。

### NAS

```bash
ls -ln data claude
# compose 里设置 PUID/PGID 与属主一致
docker compose pull && docker compose up -d
```

改密码：改 `WEB_CLAUDE_TOKEN`；改端口：同时改 `ports` 与 `WEB_CLAUDE_PORT`。

镜像预装：**git** + **Claude Code**；其余用 `sudo apt` 按需装。

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 网页登录密码 | 必填（Compose 示例为 `password`） |
| `WEB_CLAUDE_PORT` | 监听端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 可浏览的项目根 | 二进制：用户家目录；Docker：`/data` |
| `PUID` | 业务进程 uid（NAS 对齐共享属主） | `1000` |
| `PGID` | 业务进程 gid | `1000` |

Claude 的 API / 模型等写在 `./claude/settings.json`，或通过 `ANTHROPIC_*` 环境变量传入。

---

## 发版

```bash
git tag v0.1.8
git push origin v0.1.8
```

自动发布：

1. GitHub Release：多平台 `web-claude` 二进制  
2. 镜像：`ghcr.io/myflavor/web-claude:latest` / `:0.1.8`
