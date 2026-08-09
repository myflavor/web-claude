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

本机需已安装 `claude` 与 `git`（从 **PATH** 查找）。

---

## 方式二：Docker Compose

准备：

```bash
mkdir -p data claude
# 可选配置
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
      PUID: 1000
      PGID: 1000
    volumes:
      - ./data:/data
      - ./claude:/home/sandbox/.claude
    restart: unless-stopped
```

```bash
docker compose pull
docker compose up -d
```

### 挂载（不要挂整个 home）

| 宿主机 | 容器 | 用途 |
|--------|------|------|
| `./data` | `/data` | 项目目录 |
| `./claude` | `/home/sandbox/.claude` | settings、会话 transcript |

镜像内已有（**不挂载**）：

- `/home/sandbox/.profile`、`.bashrc`（装 Claude **之前**创建，install.sh 会往里写 PATH）
- `/home/sandbox/.local`（Claude 安装位置）
- `/usr/local/bin/claude`（固定一份，供 Go 从 PATH 启动）
- 系统 `git`

### 进程怎么找命令

- **Go / web-claude**：`PATH` 里找 `claude`、`git`
- **会话**：`bash -lc`，会加载 `~/.profile` / `~/.bashrc`（镜像里那份）

### 装软件

```bash
sudo apt-get update && sudo apt-get install -y python3
```

装到系统 PATH 的工具会话里都能用。  
（容器重建会丢 apt 包，需要的话写进自己的 Dockerfile。）

**不要**写 compose `user:`。

### NAS

```bash
ls -ln data claude
# PUID/PGID 对齐属主
```

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 网页登录密码 | 必填 |
| `WEB_CLAUDE_PORT` | 监听端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 项目根 | 二进制：家目录；Docker：`/data` |
| `PUID` / `PGID` | 业务进程 uid/gid | `1000` |

---

## 发版

```bash
git tag v0.1.11
git push origin v0.1.11
```
