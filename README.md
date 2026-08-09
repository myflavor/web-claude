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

| 宿主机 | 容器 |
|--------|------|
| `./data` | `/data` |
| `./claude` | `/home/sandbox/.claude` |

NAS：`PUID`/`PGID` 填 `ls -ln` 的属主。

装软件：`sudo apt-get install -y ...`

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 登录密码 | 必填 |
| `WEB_CLAUDE_PORT` | 端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 项目根 | Docker：`/data` |
| `PUID` / `PGID` | sandbox uid/gid | `1000` |

---

## 发版

```bash
docker build -t web-claude:local .
git tag v0.1.16 && git push origin v0.1.16
```
