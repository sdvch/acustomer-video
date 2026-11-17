# Design: 用户注册登录 + 管理后台

## Context

- 现有认证：微信 code2session（小程序）+ dev 单用户（H5/本地）；JWT 由 `middleware.IssueToken` 签发（uid + exp，无角色概念）
- 迁移机制：`migrations/*.sql` 幂等执行（CREATE TABLE IF NOT EXISTS）+ `ensureSchemaUpgrades` 幂等补列（MySQL 无 ADD COLUMN IF NOT EXISTS）
- 部署形态：单服务器，nginx 路径前缀分流（/acv/v1、/storage、/acv-h5），push main 自动部署

## Goals / Non-Goals

- Goals：手机号+密码注册登录；独立管理员体系与后台前端；用户禁用治理；运营数据总览；内容审查（看用户项目/成片、下架成片）
- Non-Goals：短信验证码（后续接阿里云短信再评估）；RBAC 细粒度权限（仅 super/operator 两级）；审计日志表（后续）；微信-手机号双向打通合并账号（仅单向绑定）

## Decisions

### D1: 密码认证用手机号，bcrypt 哈希

无短信成本、运营可用性好。密码 bcrypt cost 10；手机号 11 位数字校验；密码 ≥8 位。`users.phone` 可空 + 唯一索引（微信用户未绑定时为 NULL，避开唯一键冲突）。

### D2: 用户与管理员账号体系完全隔离

独立 `admins` 表（username/password_hash/role/status），不与 users 表混用。理由：安全边界清晰——用户 token 无法触达 admin 路由（scope 声明不同），管理员泄露不影响用户体系。首个超管由环境变量 `ADMIN_USERNAME`/`ADMIN_PASSWORD` 在启动时种子创建（表空才创建）；未配置时自动生成随机密码打一条启动日志。

### D3: admin JWT 加 scope 声明 + httpOnly Cookie

- 用户 token：`{uid, exp}`（现状不变，向后兼容旧 token）
- 管理 token：`{admin_id, role, scope:"admin", exp}`
- `JWTAuth` 拒绝 scope=admin 的 token；`AdminAuth` 只接受 scope=admin
- 管理后台登录成功后 token 写入 httpOnly Cookie（SameSite=Lax，Path=/），前端同源请求自动携带，不落 localStorage；同时兼容 Authorization 头（方便 curl 调试）

### D4: 管理后台前端独立工程（Vite + React + antd）

`admin/` 目录，与 miniprogram 完全解耦：技术栈不同（antd 桌面组件 vs Taro 移动端）、发布路径不同（/acv-admin/ vs /acv-h5/）、受众不同（内部运营 vs C 端用户）。CI 加构建校验 job，deploy.yml 同步发布到 /home/admin/acv-admin。

### D5: 禁用用户在 JWT 中间件拦截

`JWTAuth` 解析 uid 后查一次用户状态（PK 点查，代价可接受），status=0 返回 401「账号已被禁用」。避免禁用后旧 token 继续使用 7 天。

### D6: H5 不再静默 dev 登录

H5 无 token 时跳转登录页（注册/登录）；小程序保留微信静默登录。后端 dev 单用户降级保留（本地开发用），生产上 H5 前端不再触发它。

## Risks / Trade-offs

- 每请求多一次 PK 查询（用户状态）——当前规模可忽略；量大后可加进程内 TTL 缓存
- httpOnly Cookie 在 http（无 TLS）下传输——服务器当前只有 IP + 80 端口，Cookie 未开 Secure；接入域名 + HTTPS 后补
- 手机号唯一索引在已有数据上建索引，users 表目前数据量极小，无风险
