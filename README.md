# Claude Mobile / Web Claude

单二进制 **Go** 服务 + **Vue 3 / Vite** 前端：用浏览器（PC / 手机 H5）管理多路 Claude Code 会话。前端构建产物通过 `go:embed` 打进二进制。

两种运行方式同一套程序：

| 方式 | 场景 | Claude 从哪来 |
|------|------|----------------|
| **独立运行** | WSL / 本机 | 系统已安装的 `claude`，默认用真实 `~/.claude` |
| **Docker** | NAS 一键部署 | 镜像内置 Claude Code；配置挂在 volume |

---


## 前端开发

源码在 `web/ui`（Vue 3 + Vite + Vue Router）。生产构建输出到 `web/static`，由 `web/embed.go` 嵌入。

```bash
cd web/ui
npm install
npm run dev      # http://127.0.0.1:5173 ，/api 代理到 :3080
npm run build    # → web/static
```

本地联调：先 `go run ./cmd/server`，再 `npm run dev`。

改 UI 后需 `npm run build` 再 `go build`（Docker 构建会自动跑前端）。

---
## 设计原则

1. **程序本身不内置 Claude 配置逻辑**——尽量复用你本机 / 挂载进去的 `.claude`。
2. **Docker 只是打包**：镜像 = Go 服务 + `claude` CLI + 基础工具。
3. **Web 登录密码**（`WEB_CLAUDE_TOKEN`）和 **API Key** 分开。

```text
WSL 独立运行:
  手机浏览器 → :3080 → web-claude(二进制)
                           → spawn 系统 claude
                           → HOME=~  →  ~/.claude/settings.json

NAS Docker:
  手机浏览器 → :3080 → container
                         → web-claude
                         → spawn 镜像内 claude
                         → HOME=/home/claude
                              └── .claude/settings.json  ← 挂载 ./settings.json
                         → WEB_CLAUDE_ROOT=/data         ← 挂载 ./data
```

---

## 方式 A：WSL / 本机独立运行

前提：本机已能正常使用 `claude`（含自定义 API、settings）。

```bash
cd claude-mobile

# 构建前端（首次 / 改 UI 后）
(cd web/ui && npm install && npm run build)

# 最小配置：只要 Web 登录密码
export WEB_CLAUDE_TOKEN='你的网页密码'
# 可选：监听端口（默认 3080）
# export WEB_CLAUDE_PORT=3080
# 可选：可浏览的项目根（默认用户家目录 ~）
# export WEB_CLAUDE_ROOT="$HOME"
# 不要设置 CLAUDE_HOME / HOME_DIR → 自动用真实用户 HOME 和 ~/.claude

go run ./cmd/server
# 或
go build -o web-claude ./cmd/server && ./web-claude
```

也支持自动读当前目录 `.env`（不会覆盖已有环境变量）：

```bash
cp .env.example .env
# 编辑 WEB_CLAUDE_TOKEN / WEB_CLAUDE_PORT / WEB_CLAUDE_ROOT；API 相关若已在 shell/settings 里可省略
./web-claude
```

打开 `http://<wsl-ip>:3080`（手机同一局域网 / Tailscale）。

**此时 settings：** 直接用 `~/.claude/settings.json`，无需复制。  
env 里的 `ANTHROPIC_*` 若设置了，会覆盖进子进程（便于临时改网关）。

---

## 方式 B：Docker

```bash
cp settings.json.example settings.json   # 按需改 Claude 配置
mkdir -p data
# 如仓库/包为私有：先 docker login ghcr.io
docker compose pull
docker compose up -d
```

默认使用 CI 发布的镜像 `ghcr.io/myflavor/web-claude:latest`（见 `docker-compose.yml`）。


挂载约定：

| 宿主机 | 容器 |
|--------|------|
| `./data` | `/data`（项目根，`WEB_CLAUDE_ROOT`） |
| `./settings.json` | `/home/claude/.claude/settings.json` |

只需 `WEB_CLAUDE_TOKEN`；可选 `WEB_CLAUDE_PORT` 改宿主端口。


## 配置对照

| 变量 | 独立运行 | Docker |
|------|----------|--------|
| `WEB_CLAUDE_TOKEN` | 网页密码（首选） | 网页密码 |
| `WEB_CLAUDE_ROOT` | 可浏览的代码根（默认 `~`） | 容器内 `/data` |
| `WEB_CLAUDE_PORT` | 监听端口（默认 `3080`） | compose 宿主映射端口 |
| `ANTHROPIC_*` | 可选（已有 settings/env 可省略） | 常用，注入自定义网关 |
| `CLAUDE_BIN` | 默认 PATH 里的 `claude` | 镜像内 `claude` |
| `RUN_MODE` | `native` / `auto` | compose 设 `docker` |

---

## 功能（MVP）

- Token 登录
- 多会话并行；单会话多端连接
- 断网不杀进程；重连回放最近输出
- 选目录启动 `claude`
- xterm.js（PC + H5）
- 上传文件 / 粘贴图片 → `{cwd}/.web-claude/uploads/`

---

## 安全

- 强 `WEB_CLAUDE_TOKEN`
- 优先局域网 / Tailscale，勿裸奔公网
- `.env` 勿提交 git

---

## 开发

```bash
go test ./internal/...
go build -o web-claude ./cmd/server
```

---

## 和「挂载 settings」的关系

| 你的想法 | 实现方式 |
|----------|----------|
| 挂载整个 `.claude` | Docker：挂载父目录为 `CLAUDE_HOME`（`/data/home`），内含 `.claude` |
| 只用本机已有配置 | 独立运行：不设 `CLAUDE_HOME`，直接用 `~/.claude` |
| settings 里 permissions / model | 写在 `settings.json`；API 网关可用 env 覆盖 |

程序 **不会** 在容器里凭空生成你的 API 配置逻辑；Docker 只负责带上 CLI，配置仍是你的文件 + 环境变量。


## 发版（GitHub Actions）

打 tag 即发版：

```bash
git tag v0.1.0
git push origin v0.1.0
```

会自动：

1. **GitHub Release**：多平台独立二进制  
   - `linux/amd64` `linux/arm64`  
   - `darwin/amd64` `darwin/arm64`  
   - `windows/amd64`  
2. **Docker 镜像**（`debian:bookworm-slim` + `git` + Claude Code）：  
   `ghcr.io/myflavor/web-claude:latest` / `:0.1.0`

本机二进制：

```bash
export WEB_CLAUDE_TOKEN='你的密码'
./web-claude_v0.1.0_linux_amd64
```

Docker：

```bash
docker run --rm -p 3080:3080 \
  -e WEB_CLAUDE_TOKEN='你的密码' \
  -v "$PWD/data:/data" \
  -v "$PWD/settings.json:/home/claude/.claude/settings.json:ro" \
  ghcr.io/myflavor/web-claude:latest
```

镜像内只预装 **git** 与 **Claude Code**，其他工具请自行装进容器或挂载。

