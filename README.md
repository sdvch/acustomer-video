# AI 门店获客视频系统（acustomer-video）

[![CI](https://github.com/zhao-yi-fan/acustomer-video/actions/workflows/ci.yml/badge.svg)](https://github.com/zhao-yi-fan/acustomer-video/actions/workflows/ci.yml)
[![Release](https://github.com/zhao-yi-fan/acustomer-video/actions/workflows/release.yml/badge.svg)](https://github.com/zhao-yi-fan/acustomer-video/actions/workflows/release.yml)

用 **Taro 4.x 微信小程序 + Go(Gin) 后端** 实现的获客短视频拼接系统。把用户上传的图片/视频素材拼接成 1080×1920 竖屏成片。

## 与短剧项目（storyboard-system）的隔离

本项目为**全新独立项目**，仅借鉴其 ffmpeg/OSS/异步任务/LLM 调用思路，代码完全独立重写。

| 资源 | 本项目 | 说明 |
|---|---|---|
| 数据库 | 独立实例/独立库 `acustomer_video`/独立账号 | 端口 13306，与旧项目不共用 |
| 后端服务 | 独立进程/独立端口 18080 | 独立日志、独立临时目录 |
| 前端 | 全新 Taro 工程 | 独立 AppID |
| OSS | **新 Bucket** `acustomer-video-prod` | objectKey 前缀 `acustomer-video/` |
| RAM 子账号 | **复用旧项目子账号 AK/SK** | 需为该子账号追加新 bucket 读写权限 |
| 服务器 | 硬件可共用 | 其余全部隔离 |

## 目录

- `miniprogram/` — Taro 微信小程序（仅 weapp 端）
- `server/` — Go 后端（API + 异步 worker + ffmpeg 流水线）
- `deploy/` — docker-compose / nginx / CORS 配置
- `docs/` — 架构 / API / ffmpeg 规格文档

## 快速开始

### 后端
```bash
cd server
cp .env.example .env      # 按需修改（OSS 未配则自动走本地存储降级）
go mod tidy
go run ./cmd/server       # 监听 :18080，自动执行 migrations 建表
```
健康检查：`GET http://127.0.0.1:18080/api/v1/health`

### 前端
```bash
cd miniprogram
npm install
npm run dev        # 编译到 dist/weapp，用微信开发者工具打开
npm run build:h5   # 编译 H5 到 dist/h5（浏览器预览界面用）
```

> 接口地址由 `config/index.ts` 的 `API_BASE` 决定，构建时可用 `API_BASE=http://<host>/acv/v1` 环境变量覆盖。
> 小程序正式发布必须是已备案的 HTTPS 域名（占位符为 `https://your-domain.com/api/v1`）。
> H5 仅用于浏览器查看界面（`hash` 路由、`publicPath=/acv-h5/`），素材选择/保存相册等依赖微信 API 的能力在 H5 不可用。

### ffmpeg 流水线单独验证
```bash
cd server
go run ./cmd/composecli -out /tmp/final.mp4 img1.jpg video.mp4 img2.jpg
```

## OSS 配置要点

1. 新建 bucket（如 `acustomer-video-prod`），设置私有读写 + CORS（见 `deploy/oss-cors.json`）。
2. **复用现有 RAM 子账号**，在其授权策略中追加对新 bucket 的权限：
   ```json
   {"Version":"1","Statement":[{"Effect":"Allow","Action":["oss:PutObject","oss:GetObject","oss:DeleteObject","oss:ListObjects"],"Resource":["acs:oss:*:*:acustomer-video-prod","acs:oss:*:*:acustomer-video-prod/*"]}]}
   ```
3. 将 AK/SK 填入 `server/.env` 的 `ALIYUN_OSS_ACCESS_KEY_ID/SECRET`。**若不配 OSS，系统自动降级本地存储**（`server/storage/`），本地模式上传走 `POST /assets/upload`。

## 成片规格（SOP 表10）

1080×1920 / 9:16、30fps、H.264 High@4.1、**8-15Mbps（CBR，实测 ~11.9Mbps）**、AAC 192k/48kHz、`+faststart`、**时长 25-35s（强制校验）**。
MVP 阶段为纯拼接，字幕烧录/BGM 混音的接入点已在 `internal/ffmpeg/pipeline.go` 预留（下一轮启用）。

### 内置的质量闸门
| 闸门 | 位置 | 行为 |
|---|---|---|
| 素材质检 | `POST /assets/confirm` | 服务端 ffprobe 复核真实宽高/时长/编码，覆盖前端上报值，返回横屏/低分辨率/镜头过长等 warnings |
| 时长治理 | `POST /projects/:id/compose` | 逐段算时长求和，不在 25-35s 直接拒绝并给出"每段调到几秒/补几段"的建议 |
| 出口断言 | `ffmpeg.Validate` | 分辨率/帧率/编码/Profile/Level/码率/音频/时长 逐项校验，不符即任务失败 |
| 并发生效 | `ClaimJob` | `FOR UPDATE SKIP LOCKED`，多 worker 不互相死锁 |

## 微信登录与 AI 文案配置

```bash
# server/.env
WECHAT_APP_ID=wx...          # 填了才走真实 code2session，留空=本地开发单用户
WECHAT_APP_SECRET=...
DEEPSEEK_API_KEY=sk-...      # 复用短剧项目同一个 key
DEEPSEEK_BASE_URL=https://api.deepseek.com
DEEPSEEK_MODEL=deepseek-v4-flash
```

- 登录：`wx.login` → `POST /auth/login`，服务端 `jscode2session` 换 openid，按 `(open_id, platform)` 落 `users` 表并签发 JWT（7 天）。所有业务接口现在**必须**带 token。
- 文案：`POST /ai/copy` 生成 3 条钩子 + 正文 + CTA，编辑页有"AI 生成获客文案"按钮，点钩子可复制（详见 `docs/api.md`）。


## 持续集成与发布

流水线在 `.github/workflows/`，无需额外配置即可运行（使用内置 `GITHUB_TOKEN`）。

### CI（`ci.yml`）

PR 与推送到 `main`/`dev` 时触发，同一分支的新提交会自动取消上一次运行：

| Job | 内容 |
|---|---|
| 后端 · Go 编译与测试 | `gofmt` 检查 → `go vet` → `go build` → `go test -race` + 覆盖率 |
| 小程序 · 类型检查与构建 | `npm ci` → `tsc --noEmit` → `taro build`，产物 `weapp-dist` 可在 Actions 页下载，直接用微信开发者工具导入 |
| 部署 · 镜像可构建性 | 构建 `server/Dockerfile` 并断言镜像内 `ffmpeg`/`ffprobe` 可用 |

本地提交前对齐检查：

```bash
cd server && gofmt -l . && go vet ./... && go test ./...
cd miniprogram && npx tsc --noEmit
```

### 发布（`release.yml`）

打语义化标签即发布：

```bash
git tag v0.1.0 && git push origin v0.1.0
```

- 后端镜像推送至 `ghcr.io/zhao-yi-fan/acustomer-video`（同时打 `latest`）
- 小程序构建产物打包为 `weapp-v0.1.0.zip` 附到 Release，可直接导入微信开发者工具

服务器拉取镜像：`docker pull ghcr.io/zhao-yi-fan/acustomer-video:latest`（公开仓库的包默认可匿名拉取，若为私有需 `docker login ghcr.io`）。

> 说明：Taro 的原生 binding 走 `optionalDependencies`，本仓库 lockfile 在 macOS 生成，只锁了 darwin 包，因此流水线里显式补装 Linux 版本；本地 macOS 无需处理。

## 部署

- `deploy/docker-compose.yml` — 独立 MySQL（宿主机端口 13306）+ 后端容器（18080），数据卷/网络/容器名均带 `acustomer-` 前缀，与旧项目完全隔离
- `deploy/nginx-acv.conf` — **对外入口片段**（见下）
- `server/Dockerfile` — 多阶段构建，镜像内自带 ffmpeg
- `deploy/oss-cors.json` — 新 bucket 的 CORS 规则

```bash
cd deploy && cp .env.example .env   # 填入 OSS AK/SK 等真实值
docker compose up -d --build
```

### 线上实例（阿里云 ECS）

| 项 | 值 |
|---|---|
| **H5 预览（浏览器）** | **`http://8.152.208.234/acv-h5/`** |
| API 入口 | `http://8.152.208.234/acv/v1` |
| 健康检查 | `http://8.152.208.234/acv/v1/health` |
| 静态文件 | `http://8.152.208.234/storage/<objectKey>`（仅本地存储模式） |
| 代码目录 | `/home/admin/acustomer-video`（git clone，`deploy/.env` 未入库） |
| MySQL | 容器内 13306 映射，库 `acustomer_video`，与 storyboard 完全隔离 |

> 只有微信小程序是正式投放端。H5 是同一份 Taro 代码编出来的浏览器预览版，用于快速看界面，
> 素材选择（`chooseMedia`/`FileSystemManager`）在浏览器里不可用。

安全组只放行 80/443，18080 不直接对外，因此通过现有 nginx 加路径前缀反代（片段见 `deploy/nginx-acv.conf`，需并入 `/etc/nginx/nginx.conf` 的 80 `default_server`）。

> **不要改** `/etc/nginx/conf.d/popreels.conf`，该文件由 storyboard 的部署脚本覆盖。

更新发布：

```bash
ssh admin@<host> 'cd ~/acustomer-video && git pull && cd deploy && docker compose up -d --build'
```

国内网络注意事项（已在仓库里默认处理好）：

- Docker Hub 直连超时 → 用 `docker.m.daocloud.io` 拉基础镜像后 retag
- `apk add ffmpeg` 卡死 → `APK_MIRROR=mirrors.aliyun.com`
- Go 模块下载慢 → `GOPROXY=https://goproxy.cn,direct`

## 目标平台

**仅微信小程序**（Taro 编译到 weapp）。用微信开发者工具导入 `miniprogram/`（需填入本项目独立 AppID），接口域名加入小程序后台 request/uploadFile/downloadFile 合法域名白名单。
