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
  手机浏览器 → :3080 → claude-mobile(二进制)
                           → spawn 系统 claude
                           → HOME=~  →  ~/.claude/settings.json

NAS Docker:
  手机浏览器 → :3080 → container
                         → claude-mobile
                         → spawn 镜像内 claude
                         → HOME=/data/home  (volume)
                              └── .claude/settings.json   ← 你挂进去的配置
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
go build -o claude-mobile ./cmd/server && ./claude-mobile
```

也支持自动读当前目录 `.env`（不会覆盖已有环境变量）：

```bash
cp .env.example .env
# 编辑 WEB_CLAUDE_TOKEN / WEB_CLAUDE_PORT / WEB_CLAUDE_ROOT；API 相关若已在 shell/settings 里可省略
./claude-mobile
```

打开 `http://<wsl-ip>:3080`（手机同一局域网 / Tailscale）。

**此时 settings：** 直接用 `~/.claude/settings.json`，无需复制。  
env 里的 `ANTHROPIC_*` 若设置了，会覆盖进子进程（便于临时改网关）。

---

## 方式 B：Docker（NAS 图方便）

镜像内已安装 Claude Code。你要准备的是：

1. **项目代码目录** → `/data/projects`
2. **Claude 家目录** → `/data/home`（里面放 `.claude/`）

```bash
cp .env.example .env
# 必填: WEB_CLAUDE_TOKEN
# 填 API: ANTHROPIC_BASE_URL / ANTHROPIC_AUTH_TOKEN / ANTHROPIC_MODEL ...

# 准备 Claude 配置（二选一）

# 1) 专用目录
mkdir -p data/home/.claude data/projects
cat > data/home/.claude/settings.json <<'EOF'
{
  "permissions": {
    "deny": ["WebSearch", "WebFetch"]
  },
  "model": "your-model-id"
}
EOF

# 2) 或直接挂真实 home / 真实 .claude 父目录
# CLAUDE_HOME_HOST_PATH=/volume1/docker/claude-home

docker compose up -d --build
```

`docker-compose.yml` 关键挂载：

```yaml
volumes:
  - ${PROJECTS_HOST_PATH}:/data/projects
  - ${CLAUDE_HOME_HOST_PATH}:/data/home   # 容器 HOME
```

容器内 `HOME=/data/home`，因此 Claude 读的是：

```text
/data/home/.claude/settings.json
/data/home/.claude/projects/...
```

**等价于「把 `.claude` 挂进去」**——更准确说是挂 **包含 `.claude` 的 HOME**，这样会话记录、plugins、settings 都在一处。

若你只想挂 settings 文件：

```yaml
# 进阶：只覆盖配置文件（一般不推荐，会话数据仍在 home 里）
- ./settings.json:/data/home/.claude/settings.json:ro
```

API Key 建议用 compose `environment` / `.env` 注入，不必写进 `settings.json`（你现在的用法也是 env + settings 分工）。

---

## 配置对照

| 变量 | 独立运行 | Docker |
|------|----------|--------|
| `WEB_CLAUDE_TOKEN` | 网页密码（首选） | 网页密码 |
| `WEB_CLAUDE_ROOT` | 可浏览的代码根（默认 `~`） | 容器内 `/data/projects` |
| `WEB_CLAUDE_PORT` | 监听端口（默认 `3080`） | compose 宿主映射端口 |
| `CLAUDE_HOME` / `HOME_DIR` | **通常留空**（用真实 HOME） | `/data/home` |
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
- 上传文件 / 粘贴图片 → `{cwd}/.claude-mobile-uploads/`

---

## 安全

- 强 `WEB_CLAUDE_TOKEN`
- 优先局域网 / Tailscale，勿裸奔公网
- `.env` 勿提交 git

---

## 开发

```bash
go test ./internal/...
go build -o claude-mobile ./cmd/server
```

---

## 和「挂载 settings」的关系

| 你的想法 | 实现方式 |
|----------|----------|
| 挂载整个 `.claude` | Docker：挂载父目录为 `CLAUDE_HOME`（`/data/home`），内含 `.claude` |
| 只用本机已有配置 | 独立运行：不设 `CLAUDE_HOME`，直接用 `~/.claude` |
| settings 里 permissions / model | 写在 `settings.json`；API 网关可用 env 覆盖 |

程序 **不会** 在容器里凭空生成你的 API 配置逻辑；Docker 只负责带上 CLI，配置仍是你的文件 + 环境变量。
