# Tasks: 用户注册登录 + 管理后台

## 1. 数据库与模型

- [x] 1.1 新增 `migrations/002_admin_auth.sql`：admins 表（IF NOT EXISTS）
- [x] 1.2 `ensureSchemaUpgrades` 补 users 列：phone（可空+唯一索引）、password_hash、status；索引存在性用 information_schema.statistics 幂等检查
- [x] 1.3 `model.User` 增加 Phone/Status 字段；新增 `model.Admin`
- [x] 1.4 启动种子：admins 表为空时按 ADMIN_USERNAME/ADMIN_PASSWORD 创建超管（未配置则生成随机密码写日志）

## 2. 用户认证（后端）

- [x] 2.1 `POST /auth/register`：手机号格式 + 密码强度校验、bcrypt、唯一性冲突返回明确错误、签发 JWT
- [x] 2.2 `POST /auth/login-phone`：校验密码、status=0 拒绝、签发 JWT
- [x] 2.3 `PUT /auth/bind-phone`（JWT）：微信用户绑定手机号+设置密码
- [x] 2.4 `JWTAuth` 中间件增加用户状态拦截（status=0 → 401）；拒绝 scope=admin token
- [x] 2.5 单测：手机号/密码校验函数、token scope 区分

## 3. 管理 API（后端）

- [x] 3.1 `IssueAdminToken` + `AdminAuth` 中间件（Cookie 或 Bearer，scope=admin）
- [x] 3.2 `POST /admin/auth/login`（设 httpOnly Cookie）/ `POST /admin/auth/logout` / `GET /admin/auth/profile`
- [x] 3.3 用户管理：`GET /admin/users`（关键词+分页+项目/成片计数）、`GET /admin/users/:id`、`PATCH /admin/users/:id/status`、`POST /admin/users/:id/reset-password`、`GET /admin/users/:id/projects`
- [x] 3.4 数据总览：`GET /admin/stats/overview`（用户/项目/成片/存储用量/今日新增）
- [x] 3.5 内容审查：`DELETE /admin/videos/:id`（下架成片）
- [x] 3.6 管理员管理（仅超管）：`GET/POST /admin/admins`、`PATCH /admin/admins/:id`
- [x] 3.7 路由组装 + 超管权限在 handler 内校验

## 4. 管理后台前端（admin/）

- [x] 4.1 脚手架：Vite + React + TS + antd + react-router，base=/acv-admin/
- [x] 4.2 登录页（用户名+密码，401 提示，成功跳总览）
- [x] 4.3 布局（侧边菜单：数据总览/用户管理/内容审查/管理员）
- [x] 4.4 数据总览页（指标卡片 + 今日新增）
- [x] 4.5 用户管理页（搜索/分页表格/禁用启用/重置密码/详情抽屉含项目与成片、成片下架）
- [x] 4.6 管理员管理页（列表/新增/改角色/停用，超管专属）
- [x] 4.7 api 封装（credentials 同源 Cookie、401 统一跳登录）

## 5. 用户侧前端（Taro）

- [x] 5.1 登录/注册页（手机号+密码，双端样式适配）
- [x] 5.2 `request.ts`：H5 无 token 跳登录页；401 清 token 跳登录
- [x] 5.3 「我的」页展示登录态（手机号/昵称/平台）、登录注册入口、退出登录
- [x] 5.4 `api.ts` 增加 register/phoneLogin/profile/bindPhone

## 6. 部署与文档

- [x] 6.1 `deploy/nginx-acv.conf` 加 /acv-admin/ 段；服务器 nginx 同步 + reload
- [x] 6.2 `ci.yml` 加 admin 构建 job；`deploy.yml` 加 admin 构建发布
- [x] 6.3 `deploy/docker-compose.yml` 加 ADMIN_USERNAME/ADMIN_PASSWORD 环境变量
- [x] 6.4 `docs/api.md` 补新端点；README 目录结构补 admin/
- [x] 6.5 本地全量验证（编译+单测+冒烟）→ 中文 commit → push 部署 → 线上验证（注册登录/后台登录/用户管理全链路）
