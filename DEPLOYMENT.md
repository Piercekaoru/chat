# 部署指南

本文档记录 DEEIX-Chat / OpenAchieve 的完整部署流程：本地构建 amd64 镜像 → 上传服务器 → 服务器端更新容器。

适用场景：本地（Apple Silicon Mac）改完代码，构建镜像推到远程 Linux（amd64）服务器上线。

---

## 架构速览

- **单一应用镜像**：`Dockerfile` 是多阶段构建，前端（Next.js）+ 后端（Go）打进**同一个镜像**，容器内后端在 `:8080` 同时托管 API 和前端静态资源。
- **compose 引用的镜像 tag 由环境变量决定**：`docker-compose.yml` 里是 `image: ${DEEIX_CHAT_IMAGE:-ghcr.io/deeix-ai/deeix-chat:latest}`。服务器上通过 `.env` 把 `DEEIX_CHAT_IMAGE` 指到本地导入的 tag（当前为 **`openachieve:amd64`**）。
- **SearXNG 是独立容器**：与 app 同在 `deeix-chat-network` 网络，app 通过容器名 `http://deeix-chat-searxng:8080` 访问它（在管理后台「工具 → 联网搜索」里配置）。

> ⚠️ 关键坑：`docker load` 进来的镜像 tag 必须和 compose 实际引用的 tag 一致，否则 `docker compose up` 会认为「没变化」而不重建，代码不会更新。确认服务器实际用的 tag：
> ```bash
> docker inspect deeix-chat-app --format '{{.Config.Image}}'
> ```

---

## 一、本地构建 amd64 镜像

在项目根目录（`~/desktop/chat`）执行。

Apple Silicon Mac 必须指定 `--platform linux/amd64`，否则构建出 arm64 镜像，服务器无法运行。

```bash
# 1. 确认在正确分支、代码已提交
git status
git log --oneline -3

# 2. 构建 amd64 镜像（首次会创建 buildx builder）
docker buildx build \
  --platform linux/amd64 \
  --build-arg GIT_COMMIT=$(git rev-parse --short HEAD) \
  -t openachieve:amd64 \
  --load \
  .

# 3. 导出为 tar 包（gzip 压缩，约 75MB）
docker save openachieve:amd64 | gzip > openachieve-amd64.tar.gz
```

说明：
- `-t openachieve:amd64`：直接打成服务器 compose 引用的 tag，省去服务器上再 retag。
- `--load`：把构建结果加载进本地 docker（buildx 默认不 load）。
- `GIT_COMMIT` 会写进镜像的 buildinfo，方便线上核对版本。

> 低内存机器构建：项目里出现过名为 `openachieve-lowmem` 的 buildx builder，如需限制资源可先
> `docker buildx create --name openachieve-lowmem --driver docker-container --use` 再构建。普通机器忽略即可。

---

## 二、上传到服务器

在**本地** Mac 终端执行（IP 换成你 SSH 用的服务器地址）：

```bash
scp openachieve-amd64.tar.gz root@<服务器IP>:~/DEEIX-Chat/
```

补充：
- 用密钥 / 非默认端口：`scp -i <私钥路径> -P <端口> openachieve-amd64.tar.gz root@<IP>:~/DEEIX-Chat/`
- 包大、网络不稳想要进度条和断点续传：
  ```bash
  rsync -avz --progress openachieve-amd64.tar.gz root@<服务器IP>:~/DEEIX-Chat/
  ```

---

## 三、服务器端更新

SSH 登录服务器后，在 `~/DEEIX-Chat` 目录执行：

```bash
cd ~/DEEIX-Chat

# 1. 导入镜像
docker load -i openachieve-amd64.tar.gz

# 2. 强制重建 app 容器（--force-recreate 确保用上新镜像）
docker compose up -d --force-recreate

# 3. 核对容器确实用了新镜像
docker inspect deeix-chat-app --format '{{.Image}}'
```

第 3 步输出的 sha256 应与本地新镜像一致。本地查新镜像 ID：
```bash
docker image inspect openachieve:amd64 --format '{{.Id}}'
```

> 如果 tag 名字对不上（`docker load` 的 tag ≠ compose 引用的 tag），先补一步 retag 再重建：
> ```bash
> docker tag <load进来的tag> openachieve:amd64
> docker compose up -d --force-recreate
> ```

---

## 四、部署后验证

```bash
# 容器状态与端口
docker ps --format 'table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Ports}}'

# 后端日志（看是否正常启动、有无 panic）
docker compose logs -f app
```

浏览器侧：
1. 打开 `chat.openachieve.asia`，硬刷新（Cmd+Shift+R）。
2. 若改动涉及联网搜索：确认输入框有地球按钮，发一句「今天有什么新闻」，观察工具调用是否成功。

---

## 五、联网搜索（SearXNG）相关

如果重装了 SearXNG 或换了服务器，需要保证：

1. **SearXNG 与 app 同网络**：
   ```bash
   docker inspect deeix-chat-searxng --format '{{range $k,$v := .NetworkSettings.Networks}}{{$k}} {{end}}'
   # 应包含 deeix-chat-network；若无：
   docker network connect deeix-chat-network deeix-chat-searxng
   ```
2. **开启 JSON 输出**（否则搜索返回 403）——`/etc/searxng/settings.yml` 的 `search.formats` 须含 `json`：
   ```bash
   docker exec deeix-chat-searxng sh -c "grep -A3 'formats' /etc/searxng/settings.yml"
   ```
3. **管理后台配置**：「工具 → 联网搜索」→ 启用 → 搜索源选 SearXNG → 地址填 `http://deeix-chat-searxng:8080` → 保存。

> 注意：`deeix-chat-app` 是精简镜像，容器内**没有 `wget`/`curl`**，无法在 app 容器里直接测连通。要测 SearXNG，进 SearXNG 容器自己测：
> ```bash
> docker exec deeix-chat-searxng wget -qO- "http://localhost:8080/search?q=test&format=json" | head -c 200
> ```

---

## 六、回滚

保留上一版 tar 包即可快速回滚：

```bash
docker load -i openachieve-amd64.<上一版>.tar.gz   # 若之前按版本命名保存
docker tag <上一版镜像ID> openachieve:amd64
docker compose up -d --force-recreate
```

建议每次上线前把当前镜像另存一份带版本号的 tar，例如 `openachieve-amd64.v0.3.0.tar.gz`，便于回退。
