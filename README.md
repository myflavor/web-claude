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
# export WEB_CLAUDE_PORT=3080
# export WEB_CLAUDE_ROOT=$HOME
./web-claude_vX.Y.Z_linux_amd64
```

浏览器打开 `http://127.0.0.1:3080`，用 `WEB_CLAUDE_TOKEN` 登录。

本机需已安装 `claude` 与 `git`（Go 只从 **PATH** 查找）。

---

## 方式二：Docker Compose

准备：

```bash
mkdir -p data home
# 可选：Claude 配置
mkdir -p home/.claude
cat > home/.claude/settings.json <<'JSON'
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
      PUID: 1000
      PGID: 1000
    volumes:
      - ./data:/data
      # 整个 home：会话、.bashrc/.profile、用户软件
      - ./home:/home/sandbox
    restart: unless-stopped
```

```bash
docker compose pull
docker compose up -d
```

### 路径与会话启动

| 宿主机 | 容器 | 用途 |
|--------|------|------|
| `./data` | `/data` | 项目目录 |
| `./home` | `/home/sandbox` | `$HOME`：`.claude`、`.bashrc`、`.profile`、`~/.local` 等 |

- **Go / web-claude**：从环境 **PATH** 找 `claude`、`git`（镜像里 `claude` 在 `/usr/local/bin`，`git` 在系统路径）。
- **会话进程**：`bash -lc …` **login shell**，会读 `~/.profile` / `~/.bashrc`，你在里面加的 PATH、工具对 Claude 可见。
- **`claude` 程序**装在镜像的 `/usr/local/bin`，不依赖 home 卷，避免挂载空 home 后找不到命令。
- **会话记录**在 `home/.claude/`，重建容器不丢。

### 装软件

```bash
# 系统包
sudo apt-get update && sudo apt-get install -y python3

# 或写进 home 里持久化的 shell 配置
echo 'export PATH="$HOME/tools/bin:$PATH"' >> home/.bashrc
```

**不要**写 compose `user:`（会跳过 PUID 入口）。

### NAS

```bash
ls -ln data home
# 设置 PUID/PGID 与属主一致
```

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 网页登录密码 | 必填 |
| `WEB_CLAUDE_PORT` | 监听端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 项目根 | 二进制：家目录；Docker：`/data` |
| `PUID` / `PGID` | 业务进程 uid/gid | `1000` / `1000` |

---

## 发版

```bash
git tag v0.1.10
git push origin v0.1.10
```
