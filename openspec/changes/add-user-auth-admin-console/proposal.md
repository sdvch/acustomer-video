# Proposal: 用户注册登录 + 管理后台

## Why

当前认证体系只有微信 code2session（小程序）+ dev 单用户降级（H5），没有真正的注册登录；且系统没有任何管理入口，无法管理用户、查看运营数据、审查内容。业务上线前需要：

1. 用户侧：手机号+密码注册登录（H5 必走；小程序保留微信登录，可绑手机号）
2. 管理侧：独立管理后台（React+AntD，挂 /acv-admin/），含用户管理、数据总览、内容审查、管理员账号管理

## What Changes

- **数据库**：users 表新增 `phone`（唯一、可空）、`password_hash`、`status`（1 正常 / 0 禁用）；新增 `admins` 表（独立于用户表，超管由环境变量种子创建）
- **用户认证 API**：`POST /auth/register`（手机号+密码注册）、`POST /auth/login-phone`（手机号+密码登录）、`PUT /auth/bind-phone`（微信用户绑定手机号）；微信登录保留不变
- **管理 API**：`/api/v1/admin/*` 路由组，独立 admin JWT（scope=admin + httpOnly Cookie），含管理员登录/登出/资料、用户列表搜索/禁用启用/重置密码/详情（含项目与成片）、运营数据总览、成片下架、管理员账号 CRUD（超管权限）
- **管理后台前端**：新增 `admin/` 独立工程（Vite + React + TypeScript + Ant Design），nginx 挂 `/acv-admin/`，CI 构建校验、deploy 自动发布
- **用户侧前端**：Taro 端新增登录/注册页；H5 无 token 时跳登录页（不再静默 dev 登录），小程序保留微信静默登录；「我的」页展示登录态与退出
- **禁用治理**：被禁用用户登录被拒，已持 token 的用户在 JWT 中间件处按库内状态拦截

## Impact

- 兼容性：已有微信登录链路、dev 降级链路（本地开发）不变；旧 token（无 scope 声明）仍视为用户 token
- 涉及规格：`auth`（新增手机号认证 Requirements）、新增 `admin-console` 能力域
- 部署：nginx 配置新增 /acv-admin/ 段；deploy.yml 新增 admin 构建发布；服务器新增环境变量 ADMIN_USERNAME/ADMIN_PASSWORD（首个超管）
