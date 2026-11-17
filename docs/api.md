# API 文档

Base: `/api/v1`　统一响应：`{code, data, message}`（code=0 成功）。除 `login`/`health` 外**均需** `Authorization: Bearer <token>`，缺失返回 401。

## 认证
| Method | Path | 请求 | 返回 |
|---|---|---|---|
| POST | `/auth/login` | `{platform, code, nickname?}` | `{token, dev_mode, user{id,platform,open_id}}` |
| POST | `/auth/register` | `{phone, password, nickname?}` | `{token, user}` |
| POST | `/auth/login-phone` | `{phone, password}` | `{token, user}` |
| PUT | `/auth/bind-phone` | `{phone, password}`（JWT） | `{phone}` |
| GET | `/auth/profile` | - | 当前用户 `{id,open_id,masked,month_quota,used_quota,...}` |
| GET | `/health` | - | ffmpeg 状态、db、bucket |

登录说明：
- 配置了 `WECHAT_APP_ID/WECHAT_APP_SECRET` → 调微信 `jscode2session` 换 openid，按 `(open_id, platform)` upsert `users`。
- 未配置 → 本地开发模式（`dev_mode=true`），固定开发用户，便于本机联调。
- `code` 失效/重复使用返回 `code=4001`，前端应重新 `wx.login()`。
- 手机号注册：11 位大陆手机号 + ≥8 位密码，bcrypt 存储；手机号已注册返回 409；被禁用用户登录返回 403，已持有 token 的业务请求返回 401（JWT 中间件按库内状态实时拦截）。

## 管理后台（/admin/*）
管理端独立令牌（scope=admin + httpOnly Cookie `admin_token`，也兼容 Authorization 头）。与用户 token 完全隔离：用户 token 访问管理接口 401，管理 token 访问用户接口 401。

| Method | Path | 请求 | 返回 |
|---|---|---|---|
| POST | `/admin/auth/login` | `{username, password}` | `{id, username, role}`（设 Cookie） |
| POST | `/admin/auth/logout` | - | 清除 Cookie |
| GET | `/admin/auth/profile` | - | 当前管理员 |
| GET | `/admin/users` | `?keyword=&page=&size=` | `{list, total, page, size}`（含项目/成片计数） |
| GET | `/admin/users/:id` | - | 用户详情 |
| PATCH | `/admin/users/:id/status` | `{status: 0\|1}` | 禁用/启用 |
| POST | `/admin/users/:id/reset-password` | - | `{new_password}`（仅显示一次） |
| GET | `/admin/users/:id/projects` | - | `{projects[], videos[]}` |
| DELETE | `/admin/videos/:id` | - | 下架成片 |
| GET | `/admin/stats/overview` | - | `{total_users, today_users, total_projects, total_videos, total_jobs, storage_bytes}` |
| GET | `/admin/admins` | -（仅超管） | 管理员列表 |
| POST | `/admin/admins` | `{username, password, role}`（仅超管） | 新管理员 |
| PATCH | `/admin/admins/:id` | `{role?, status?, password?}`（仅超管） | 更新 |

- 首个超管由 `ADMIN_USERNAME`/`ADMIN_PASSWORD` 环境变量在启动时种子创建（admins 表为空时）；未配置密码则生成随机密码输出到启动日志。
- 管理后台前端：`http://<IP>/acv-admin/`（React+AntD，源码在 `admin/`）。

## 素材
| Method | Path | 请求 | 返回 |
|---|---|---|---|
| POST | `/assets/presign` | `{filename,mime,type(1图2视频),shot_no}` | `{asset_id,object_key,upload_url,local_mode,preview_url}` |
| POST | `/assets/upload` | multipart: asset_id,object_key,file | 本地降级上传 |
| POST | `/assets/confirm` | `{asset_id,width,height,duration_ms,size_bytes,tags_json,authorized}` | `{asset_id,qc_status,warnings[],probe{...}}` |
| GET | `/assets` | - | 素材列表（含 `qc_status/qc_json/probe_verified`） |
| PATCH | `/assets/sort` | `{ids:[]}` | 批量排序 |
| DELETE | `/assets/:id` | - | 软删除 |

**confirm 会做服务端质检**（不再信任前端上报的宽高/时长）：
1. 把素材落到本地临时文件，用 `ffprobe` 读真实 `width/height/duration/api 编码/fps/音轨`；
2. 真实值覆盖入库，`probe_verified=1`；
3. 返回 `warnings[]`（不阻断上传，除无法解析外）：
   - 分辨率短边 < 720p（成片会发虚）
   - 横屏/方图（将补黑边）
   - 画幅偏离 9:16 超 15%
   - 视频镜头 < 2s 或 > `MAX_CLIP_SECONDS`
   - 非 h264/hevc 编码（将转码）
4. 无法解析 → `code=4003`，前端应让用户换文件。

## 项目与合成
| Method | Path | 请求 | 返回 |
|---|---|---|---|
| POST | `/projects` | `{title,industry_id}` | `{project_id}` |
| GET | `/projects` | - | 项目列表 |
| GET | `/projects/:id` | - | `{project,assets,latest_video}` |
| POST | `/projects/:id/compose` | `{asset_ids[],clip_seconds,fit_mode}` | `{job_id}`（立即返回） |
| GET | `/jobs/:id` | - | `{status,progress,stage,error_message,result_video_id,result}` |

**compose 校验（按 SOP）**
- 素材必须属于当前用户、未删除、已勾选授权；段数 `3-12`；
- `clip_seconds` 夹到 `[MIN_CLIP_SECONDS, MAX_CLIP_SECONDS]`（默认 2-8s）；
- 逐段时长：图片 = `clip_seconds`；视频 = `min(素材时长, clip_seconds)`；
- 总时长必须落在 `MIN_DURATION_SECONDS..MAX_DURATION_SECONDS`（默认 25-35s），否则返回：

```json
{"code":4002,"message":"预计成片 10.0 秒，短于 SOP 要求的 25 秒：建议每段调到 6.0 秒（当前共 5 段），或再补 8 段素材",
 "data":{"total_ms":10000,"min_ms":25000,"max_ms":35000,"asset_count":5,"suggest_clip_ms":6000}}
```

校验得到的逐段时长会随任务快照落库（`input_json.segment_ms`），worker 严格按快照裁剪，保证"校验的时长 = 实际成片时长"。

## AI 文案（SOP 阶段五）
| Method | Path | 请求 | 返回 |
|---|---|---|---|
| POST | `/ai/copy` | `{project_id?, title?, industry?, store_name?, selling_points[]?, asset_count?}` | `{hooks[{text,angle}], body, cta, tips[]}` |

- 走 DeepSeek（`DEEPSEEK_*` 配置），system prompt 内置 SOP 约束：前 3 秒钩子、单行字幕 ≤12 字、正文 100-140 字、禁用词规避、不编造门店信息。
- 未配置 key → `code=5001`；模型调用失败 → `code=5002`。
- 传了 `project_id` 会把结果写入 `copy_versions` 表，便于版本对比。

## 成片
| Method | Path | 返回 |
|---|---|---|
| GET | `/videos` | 成片列表（`file_url` 在 OSS 模式为 1 小时签名 URL） |
| GET | `/videos/:id/download-url` | `{url,filename}` |
| DELETE | `/videos/:id` | 删除 |

## 任务状态机
`queued → claimed → generating → uploading → succeeded`，异常 → `failed`，重新发起 → 旧任务 `superseded`。前端每 1.5s 轮询 `/jobs/:id`，进度 0→100。

并发领取用 `SELECT ... FOR UPDATE SKIP LOCKED`，多 worker 不会互相死锁。

## 出口规格断言（ffmpeg Validate）
1080×1920 / 30fps / h264 / High / level 4.1 / **码率 8-15Mbps（CBR，nal-hrd）** / aac 48kHz / 时长 25-35s（±1.5s 容差）。任一不符 → 任务失败并回写原因。
