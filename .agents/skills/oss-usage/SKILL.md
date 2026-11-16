---
name: oss-usage
description: 阿里云 OSS 的配置、使用与安全规范，含本地降级模式、CORS、URL 出口约定、AK/SK 安全红线。凡是涉及文件存储、上传下载、切换 OSS/本地模式、排查资源访问 404 时必读。用中文写作与回答。
---

# 阿里云 OSS 使用规范（acustomer-video）

本项目存储层是**双模式**：优先阿里云 OSS，未配置时自动降级本地存储。所有代码经 `OSSService`（`server/internal/service/oss_service.go`）统一访问，**禁止在业务代码里直接 import OSS SDK 或直接拼存储路径**。

## 模式判定

`config.OSSConfigured()`：以下 4 项 env **全部非空**才走 OSS，否则本地降级：

| 环境变量 | 说明 |
|---|---|
| `ALIYUN_OSS_ENDPOINT` | 内网上传端点（如 `oss-cn-beijing-internal.aliyuncs.com`） |
| `ALIYUN_OSS_PUBLIC_ENDPOINT` | 公网/对外端点 |
| `ALIYUN_OSS_ACCESS_KEY_ID` / `ALIYUN_OSS_ACCESS_KEY_SECRET` | RAM 子账号凭证 |
| `ALIYUN_OSS_BUCKET` | 独立 bucket（与 storyboard 项目**隔离**） |
| `ALIYUN_OSS_OBJECT_PREFIX` | 默认 `acustomer-video/` |
| `ALIYUN_OSS_REGION` | 地域 |

本地降级时必须配 `PUBLIC_BASE_URL`（对外基地址，含协议端口），否则前端拿到相对路径拿不到资源——这是历史上真实炸过的事故。

## 关键函数与用途

- `BuildObjectKey(userID, filename)`：生成对象 key，**不信任客户端文件名**（清洗+时间戳）
- `SignPutURL` / `SignGetURL`：前端直传/受限下载的预签名 URL
- `PublicURL` / `localURL`：对外访问 URL 的**统一出口**，业务代码只准用这两个
- `PutLocal` / `LocalPath` / `DownloadToLocal` / `UploadLocalFile`：本地模式与 worker 下载素材用
- `cleanKey`：防路径穿越（`../`），改 key 逻辑必须保住这层清洗

## 切换到真实 OSS 的步骤

1. 建独立 bucket + RAM 子账号（只授该 bucket 的读写，**不要用主账号 AK**）
2. 凭证写入服务器 `.env` / 环境变量，**绝不进 git、不进代码、不进 issue**
3. 配置 CORS：模板在 `deploy/oss-cors.json`（AllowedOrigin 已含 `https://servicewechat.com` 小程序域），用 `ossutil` 或控制台导入
4. 小程序后台把 OSS 公网域名加入 request/uploadFile/downloadFile 合法域名（需已备案）
5. 重启服务后用 `/acv/v1/health` + 一次完整上传链路验证

## 安全红线

- AK/SK 只能存在于：服务器环境变量、GitHub Secrets（如需要 CI 用）。出现在代码/日志/提交里立即视为事故，须轮换凭证
- 所有对 OSS 的 URL 生成走 `PublicURL`/`localURL`，禁止手拼
- bucket 私有读，下载用 `SignGetURL`；成片如需公开分享，另设过期策略，不放开整个 bucket
- 本地模式 `LocalPath` 返回的路径必须落在 `STORAGE_DIR` 挂载卷内（`cleanKey` 保证），任何新代码不得绕过

## 排障速查

| 症状 | 先查什么 |
|---|---|
| 前端拿到的资源 URL 打不开 | `PUBLIC_BASE_URL` 是否配置；nginx `/storage/` 反代是否在 |
| 小程序上传报域名校验失败 | OSS CORS（oss-cors.json）+ 小程序合法域名白名单 |
| 上传成功但质检找不到文件 | OSS 未配置却用了 OSS 路径；或 `LocalPath` 落在卷外 |
| 切 OSS 后旧本地素材 404 | 本地素材不会自动迁移，需要脚本搬移并改库里 key |

## 回答与沟通约定

本 skill 生效期间，向用户解释方案、撰写文档、生成代码注释时**一律使用中文**。
