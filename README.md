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

本机 PATH 上需有 `claude`、`git`。会话经 login shell 启动，会读你的 `~/.profile` / `~/.bashrc`。

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

### 设计（稳）

| 目标 | 做法 |
|------|------|
| Claude 读得到 profile/bashrc | 会话用 **`bash -lc`**（login shell） |
| NAS 方便 | **`PUID`/`PGID`**：直接以该数字 uid 运行，**不 usermod** |
| Go 找得到 claude/git | 固定在 **`/usr/local/bin`** 与系统 PATH |
| 会话不丢 | 只挂 **`./claude` → `/home/sandbox/.claude`**（不挂整个 home） |

镜像构建时：

1. 在 `/home/sandbox` 先建好 `.profile` / `.bashrc`
2. 用 sandbox 装 Claude（可写进 profile）
3. 再把 `claude` 拷到 `/usr/local/bin`

启动时 entrypoint：

1. 保证 `.profile` / `.bashrc` 存在  
2. `chown PUID:PGID` 到 home / data  
3. `gosu PUID:PGID web-claude`（**不改** `/etc/passwd`）

### NAS

```bash
ls -ln data claude
# 例如 1002:10
```

```yaml
environment:
  PUID: 1002
  PGID: 10
```

**不要**写 compose `user:`。

### 装软件

会话里：

```bash
sudo apt-get update
sudo apt-get install -y python3
```

（按当前 `PUID` 写了免密 sudo。）

| 宿主机 | 容器 |
|--------|------|
| `./data` | `/data` 项目 |
| `./claude` | `/home/sandbox/.claude` 配置+会话 |

---

## 环境变量

| 变量 | 说明 | 默认 |
|------|------|------|
| `WEB_CLAUDE_TOKEN` | 登录密码 | 必填 |
| `WEB_CLAUDE_PORT` | 端口 | `3080` |
| `WEB_CLAUDE_ROOT` | 项目根 | Docker：`/data` |
| `PUID` / `PGID` | 进程 uid/gid（NAS） | `1000` |

---

## 发版

```bash
git tag v0.1.12
git push origin v0.1.12
```
