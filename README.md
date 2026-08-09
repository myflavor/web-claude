# Web Claude

用浏览器远程使用 Claude Code（PC / 手机）。  
两种用法：**单二进制**、**Docker Compose**。

默认地址：`http://<主机>:3080`  
Compose 默认登录密码：`password`

---

## 方式一：单二进制

从 [Releases](https://github.com/myflavor/web-claude/releases) 下载对应平台文件。

```bash
chmod +x web-claude_vX.Y.Z_linux_amd64
export WEB_CLAUDE_TOKEN=password
./web-claude_vX.Y.Z_linux_amd64
```

本机 PATH 上需有 `claude`、`git`。

---

## 方式二：Docker Compose

```bash
mkdir -p data claude
cat > claude/settings.json <<'JSON'
{
  "permissions": {
    "deny": ["WebSearch", "WebFetch"]
  }
}
JSON
```

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

### 启动时做了什么

```text
entrypoint (root)
  → usermod/groupmod sandbox = PUID:PGID
  → chown /home/sandbox、/home/sandbox/.claude、/data
  → gosu sandbox
  → bash -lc          # 读 ~/.profile ~/.bashrc
  → exec web-claude
  → exec claude       # 继承环境
```

| 宿主机 | 容器 |
|--------|------|
| `./data` | `/data` 项目 |
| `./claude` | `/home/sandbox/.claude` 配置+会话 |

**不要**写 compose `user:`（会跳过入口）。

### NAS

```bash
ls -ln data claude
```

```yaml
environment:
  PUID: 1002
  PGID: 10
```

### 装软件

```bash
sudo apt-get update
sudo apt-get install -y python3
```

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 登录密码 | 必填 |
| `WEB_CLAUDE_PORT` | 端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 项目根 | Docker：`/data` |
| `PUID` / `PGID` | sandbox 的 uid/gid | `1000` |

---

## 发版

```bash
git tag v0.1.14
git push origin v0.1.14
```
