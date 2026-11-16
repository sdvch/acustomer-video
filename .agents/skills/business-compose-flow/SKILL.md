---
name: business-compose-flow
description: 获客短视频拼接业务的完整链路与业务规则。涉及素材上传、质检、合成任务、成片规格、项目与资产模型等任何业务功能开发时必读。用中文写作与回答。
---

# 获客短视频合成业务流程（acustomer-video）

本 skill 定义本项目的业务全链路、硬性业务规则和改动边界。开发任何业务功能前先通读本文，确保不偏离业务定位。

## 业务定位

面向获客场景的竖版短视频拼接系统：用户（商家/运营）把已有素材（视频片段为主）上传，系统自动质检、转码归一化、拼接成一条符合投放规格的竖版成片（类似朋友圈/信息流获客素材）。**不是**剪辑器、不是模板视频生成器、不做实时预览。

## 核心链路（按用户视角）

1. **登录**：小程序走微信 `code2session`；H5/未配置 AppID 时走本地开发单用户降级（`dev` 模式），两种模式不阻断核心链路。
2. **创建项目**：一个 `compose_project` 对应一次合成意图。
3. **选择素材**：从相册/文件选取，**3~12 个**（`MinAssetCount` / `MaxAssetCount`），提交前校验。
4. **上传**：前端经 `src/platform` 适配层上传（小程序 Taro.uploadFile，H5 fetch FormData），后端落 OSS 或本地存储。
5. **服务端质检**：`ffprobe` 实测宽高/时长/编码，**覆盖前端上报值**（前端值仅参考）。产出质检结论：`passed` 或带警告（横屏素材、低分辨率、单镜头过长）——警告必须明示给用户，不能静默吞掉。
6. **提交合成**：后端强制校验**总时长 25~35 秒**（素材时长求和，不够/超了都拒绝），不通过返回明确业务错误。
7. **合成执行**（worker 异步）：
   - 逐段转码归一化（1080×1920 / 30fps / H.264 High@4.1 / 12Mbps 真 CBR / AAC 192k 48kHz）
   - 无损 concat 拼接
   - **出口断言** `ffmpeg.Validate` 逐项复核规格，任何一项不达标即任务失败，**不上传不成片**
   - 进度按阶段加权上报（前端可见百分比），失败原因持久化可查
8. **成片交付**：用户浏览成片、保存到相册（小程序）/下载（H5）。

## 硬性业务规则（改任何数值前先改 openspec/specs）

| 规则 | 数值/约定 | 权威源 |
|---|---|---|
| 成片分辨率 | 1080×1920 竖版 | `openspec/specs/video-output` |
| 帧率/编码/码率 | 30fps / H.264 High@4.1 / 12Mbps 真 CBR | 同上 |
| 音频 | AAC 192k / 48kHz，`+faststart` | 同上 |
| 成片时长 | 25~35 秒，提交时强制校验 | `openspec/specs/compose-jobs` |
| 素材数量 | 3~12 个 | `config.go` Min/MaxAssetCount |
| 质检 | 服务端 ffprobe 实测覆盖前端上报 | `openspec/specs/asset-quality` |

数值边界属于产品规格：**必须先改 `openspec/specs/` 再改代码**，且要在 proposal 里说明业务理由。

## 数据模型速查

- `compose_projects`：合成项目（用户、标题、状态）
- `assets`：素材（key、时长、宽高、质检结果、警告）
- `compose_jobs`：合成任务（状态机、进度、失败原因、认领时间）
- `videos`：成片（输出 key、规格断言结果）

状态流转：project → pending job → claimed（worker 认领，`FOR UPDATE SKIP LOCKED`）→ transcoding → concat → validating → done / failed。

## 开发边界

- 涉及时长分配、镜头节奏、素材顺序的业务逻辑放在 service 层（可测纯函数），不要散进 handler 或 worker 循环
- 新业务字段先想清楚：是否要进质检？是否影响时长校验？是否要展示警告？
- API 变更同步更新 `docs/api.md`
- 大功能（如字幕烧录、BGM 混音、封面生成）先走 `/openspec-propose`，预留接入点见 `video-output` spec

## 回答与沟通约定

本 skill 生效期间，向用户解释业务方案、撰写文档、生成代码注释时**一律使用中文**。
