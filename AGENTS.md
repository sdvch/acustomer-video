# AGENTS.md — acustomer-video AI 开发规范（唯一规则源）

> 本文件是本仓库所有 AI 编码工具（Codex / Cursor / Qoder / VS Code Copilot 等）的**唯一规则源**。
> `.cursor/rules/`、`.qoder/rules/`、`.github/copilot-instructions.md` 均为指向本文件的指针，请勿在其中直接写规则。
> Skills 唯一物理源：`.agents/skills/`，其他工具目录（`.cursor/skills`、`.qoder/skills`、`.github/skills`）为软链引用。

## 项目概述

获客短视频拼接系统：Taro 4.x（React）小程序 + H5 双端前端 + Go(Gin) 后端。
用户上传素材 → 质检 → ffmpeg 拼接合成 → 产出竖版成片（1080×1920 / 30fps / H.264 1080High@4.1 / 8-15Mbps CBR / 25-35s）。

## 目录结构

- `server/`：Go 后端。`cmd/`（server、composecli）、`internal/`（handler、service、repository、model、ffmpeg、worker、middleware）、`migrations/`
- `miniprogram/`：Taro 前端。`src/pages`、`src/platform`（双端适配层）、`src/services`、`src/components`、`config/`
- `deploy/`：docker-compose（MySQL 13306 / server 18080）、nginx-acv.conf、oss-cors.json
- `docs/`：api.md、ffmpeg-presets.md
- `scripts/`：deploy.sh、sync-agent-links.sh
- `storage/`：本地降级存储（内容不入库）

## 提交前必须通过（与 CI 对齐）

```bash
cd server && gofmt -l . && go vet ./... && go test ./...
cd miniprogram && npx tsc --noEmit
```

## 后端约定

- gofmt 强制：CI 对未格式化文件直接失败
- 错误包装用 `%w`；ffmpeg 失败必须透出 stderr 尾部（惯例 tail 800 字符，见 `internal/ffmpeg/pipeline.go`）
- 出口断言 `ffmpeg.Validate` 逐项校验规格，不得绕过；成片时长 25-35s 强制
- worker 认领任务用 `FOR UPDATE SKIP LOCKED`（ClaimJob），勿改
- 路由前缀代码内固定 `/api/v1`（`handler/router.go`）；对外 nginx 反代为 `/acv/v1`。新接口挂 `v1 := r.Group("/api/v1")` 下并走 authed 中间件（业务接口必须带 JWT）

## 前端约定

- 平台适配层 `src/platform/`（wechat.ts / h5.ts，构建期 TARO_ENV 分流）：页面代码严禁出现 `if (h5)` 分支，平台能力（选素材/上传/保存相册/复制）一律 `import` 自 `src/platform`（index.ts 统一出口）
- `API_BASE` 只在 `miniprogram/config/index.ts` 定义，勿散落；H5 构建时注入 `API_BASE=http://<host>/acv/v1`
- 详细可维护性规范见 skill：`frontend-code-style`（`.agents/skills/frontend-code-style/`）

## 存储与部署

- storage 路径约定：`STORAGE_DIR=../storage`（仓库根 `storage/`）；OSS 未配置时走本地降级，静态访问前缀 `/storage`
- **push main 即自动部署到生产**（`.github/workflows/deploy.yml`）：禁止推送未过本地校验的代码；打 `v*` 标签触发 Release
- 服务器 nginx 与 storyboard 项目共用：只改/并 `deploy/nginx-acv.conf` 片段，**绝不动** `/etc/nginx/conf.d/popreels.conf`
- 与 storyboard-system 完全隔离（端口 18080/13306、独立库 acustomer_video、独立 bucket），禁止交叉引用其代码/配置

## 行为规格（OpenSpec）

- 行为规格唯一权威源：`openspec/specs/`（video-output 成片规格与出口断言、asset-quality 质检、compose-jobs 任务、dual-platform 双端、storage 存储、auth 认证）
- README 与 `docs/` 的规格表仅作概览，冲突时以 `openspec/specs/` 为准；数值边界改动必须先改 spec 再改代码
- 大功能走 OpenSpec 流程：`/openspec-propose "想法"` → design/tasks → 实现 → 验证记录 → archive 同步规格；小 bugfix 无需走此流程
- openspec CLI 用 `npx -y @fission-ai/openspec`（未全局安装）；工作流 skill 已在 `.agents/skills/openspec-*`

## Git 提交规范（强制）

- AI 生成的 commit message **必须使用中文**：`类型(范围): 中文摘要`，正文也用中文
- 类型保留英文单词：feat / fix / chore / docs / test / refactor / style / perf / ci / build
- 示例：`feat(合成): 新增字幕烧录与 BGM 混音`；`fix(存储): 修复本地降级 URL 缺少 PUBLIC_BASE_URL 导致资源 404`
- 正文写清动机与影响范围，禁止无意义的 "update" / "修改" 式摘要
- **push main 即上生产**，提交前必须过本地校验（见上节），未验证的改动先开分支

## AI 工具配置

- rules 唯一源 = 本文件；skills 唯一源 = `.agents/skills/`
- **业务类 skill 一律用中文撰写**（正文与 description），通用工程 skill 可沿用上游英文原文
- skill 分工：业务全流程见 `business-compose-flow`；OSS/存储见 `oss-usage`；发版/服务器/nginx 见 `deploy-ops`；前端规范见 `frontend-code-style`
- 新增 skill：在 `.agents/skills/<name>/SKILL.md`（frontmatter `name` 须与目录名一致、全小写字母数字连字符），然后运行 `./scripts/sync-agent-links.sh`
- fresh clone 后运行一次 `./scripts/sync-agent-links.sh` 修复软链
