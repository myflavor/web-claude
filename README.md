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
# 容器用户 uid=1000，权限不对时可：
# chown 1000:1000 settings.json
# chmod 666 settings.json
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
    restart: unless-stopped
```

启动：

```bash
# 私有包需先：docker login ghcr.io
docker compose pull
docker compose up -d
```

挂载：

| 宿主机 | 容器 | 用途 |
|--------|------|------|
| `./data` | `/data` | 项目目录 |
| `./settings.json` | `/home/sandbox/.claude/settings.json` | Claude 配置 |

镜像约定：

- **非 root**：用户 `sandbox`（uid **1000**）
- `HOME=/home/sandbox`
- Claude 配置目录：**`/home/sandbox/.claude`**
- `claude` 程序：`/usr/local/bin/claude`
- 预装：**git** + **Claude Code**

`settings.json` 建议不要加 `:ro`（Claude 可能写入）。

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
git tag v0.1.4
git push origin v0.1.4
```

自动发布：

1. GitHub Release：多平台 `web-claude` 二进制  
2. 镜像：`ghcr.io/myflavor/web-claude:latest` / `:0.1.4`
