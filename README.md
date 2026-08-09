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
mkdir -p data
# 必须是文件（缺文件时 Docker 会建成目录）
cat > settings.json <<'JSON'
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
    volumes:
      - ./data:/data
      - ./settings.json:/home/sandbox/.claude/settings.json
    # NAS：写共享目录属主（ls -ln / id 用户）。本机默认 1000 可省略。
    # user: "1002:10"
    restart: unless-stopped
```

启动：

```bash
# 私有包需先：docker login ghcr.io
docker compose pull
docker compose up -d
```

### 本机 / 非 NAS

一般**不用**写 `user:`。镜像默认以 uid **1000** 运行。

### NAS

1. 查属主：

```bash
ls -ln data settings.json
# 或
id 你的用户名
```

2. 在 compose 里打开一行，例如：

```yaml
user: "1002:10"
```

3. **不要**再设 `PUID`/`PGID`（已废弃）。  
4. `settings.json` 必须是文件；不要加 `:ro`。

```bash
docker compose pull
docker compose up -d
```

挂载：

| 宿主机 | 容器 | 用途 |
|--------|------|------|
| `./data` | `/data` | 项目目录 |
| `./settings.json` | `/home/sandbox/.claude/settings.json` | Claude 配置 |

镜像约定：

- **非 root**（默认 uid 1000；可用 compose `user:` 换成任意 uid）
- `HOME=/home/sandbox`（目录对任意 uid 可写，用于 transcript）
- Claude 配置：`/home/sandbox/.claude`
- 程序：`/usr/local/bin/claude`、`/usr/local/bin/web-claude`
- 预装：**git** + **Claude Code** only

改密码：改 `WEB_CLAUDE_TOKEN`；改端口：同时改 `ports` 与 `WEB_CLAUDE_PORT`。

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 网页登录密码 | 必填（Compose 示例为 `password`） |
| `WEB_CLAUDE_PORT` | 监听端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 可浏览的项目根 | 二进制：用户家目录；Docker：`/data` |

Claude 的 API / 模型等仍走 Claude 自己的配置（`settings.json` 或 `ANTHROPIC_*` 环境变量）。

---

## 发版

```bash
git tag v0.1.6
git push origin v0.1.6
```

自动发布：

1. GitHub Release：多平台 `web-claude` 二进制  
2. 镜像：`ghcr.io/myflavor/web-claude:latest` / `:0.1.6`
