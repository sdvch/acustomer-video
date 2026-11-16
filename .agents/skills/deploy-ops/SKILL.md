---
name: deploy-ops
description: 部署与运维手册：push 即上生产的自动部署流水线、服务器结构、nginx 隔离红线、网络加速镜像、常见故障排查。凡是涉及发版、改 CI/CD、服务器排障、nginx 配置时必读。用中文写作与回答。
---

# 部署与运维手册（acustomer-video）

## 发版模型

- **push main = 自动部署到生产**（`.github/workflows/deploy.yml`）：构建 H5（注入 API_BASE）→ scp 到服务器 → git fetch + ff-only → `scripts/deploy.sh` → 解压 H5。推送未过本地校验的代码等于直接上生产，禁止。
- `v*` 标签 → Release（GHCR 镜像 + 小程序 zip）
- 每次推送跑 CI（`ci.yml`）：gofmt/vet/test-race、tsc+双端构建+白屏回归防护、镜像构建 ffmpeg 断言

## 服务器拓扑（admin@8.152.208.234）

- 代码：`/home/admin/acustomer-video`；H5 静态：`/home/admin/acv-h5`
- 容器：MySQL（13306，限内存参数已调）/ server（18080，healthcheck）
- 与 storyboard 项目**共用 nginx 和安全组**：只开 80/443
- 路由：`/acv/v1/` → `127.0.0.1:18080/api/`；`/storage/` → 静态；`/acv-h5/` → H5

## nginx 红线（违反会弄坏另一个生产项目）

- **绝对不动** `/etc/nginx/conf.d/popreels.conf`（storyboard 的配置，其脚本会覆盖）
- 本项目 nginx 片段唯一源：`deploy/nginx-acv.conf`；服务器实际生效位置是 `/etc/nginx/nginx.conf` 里的对应 server 块，改完 `nginx -t` 再 reload
- 安全组只开 80/443，任何新端口都到不了公网——新增服务一律走 nginx 路径前缀反代

## 网络环境（国内服务器，直连超时是常态）

| 坑 | 解法（已固化在配置里） |
|---|---|
| Docker Hub 超时 | 拉取用 `docker.m.daocloud.io` retag |
| `apk add` 卡死 | Dockerfile `APK_MIRROR` 默认阿里云镜像源 |
| Go 模块超时 | `GOPROXY=https://goproxy.cn` |
| npm 慢 | CI/构建机按需配镜像 |

## 手动操作

```bash
# 手动部署（绕过流水线时）
ssh admin@8.152.208.234 'cd ~/acustomer-video && git pull && cd deploy && docker compose up -d --build'

# 健康检查
curl http://8.152.208.234/acv/v1/health

# 看容器日志（部署失败时 deploy.sh 也会自动打出）
ssh admin@8.152.208.234 'docker logs --tail 100 acustomer-server'
```

## GitHub Secrets

`ECS_HOST` / `ECS_USER` / `ECS_SSH_KEY`（专用 ed25519，私钥只在 Secrets）。轮换：生成新密钥对 → 服务器追加公钥 → 更新 Secret → 删旧公钥。

## 排障速查

| 症状 | 排查顺序 |
|---|---|
| 部署后 health 非 200 | deploy.yml 日志（deploy.sh 已自动带容器日志）→ `docker ps` 看 restart → `docker logs` |
| H5 白屏 | `dist/h5/index.html` 是否含全部 initial chunk（gen-h5-html.mjs 产物）→ curl 逐资源 200 |
| 改了 nginx 不生效 | `nginx -t` 报错？conf.d 有没有把 popreels.conf 顶掉？ |
| CI 绿但线上行为不对 | 确认 deploy run 是否真的跑到了同一 commit；服务器 `git log -1` |
| 磁盘满（成片/素材堆积） | `du -sh storage/`，清理已删除项目的孤儿素材 |

## 数据与清理

- 测试数据清空：TRUNCATE `videos`/`compose_jobs`/`assets`/`compose_projects` + 删 `storage/acustomer-video/u1/*`
- MySQL 数据卷、storage 卷都在 compose 定义的挂载里，重建容器不丢；删卷=删数据

## 回答与沟通约定

本 skill 生效期间，向用户解释方案、撰写文档、生成代码注释时**一律使用中文**。
