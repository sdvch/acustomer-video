# admin-console 规格变更：新增管理后台能力域

## ADDED Requirements

### Requirement: 管理员独立账号体系

系统 SHALL 使用独立 `admins` 表管理管理员账号（username 唯一、bcrypt 密码、role=super|operator、status），与用户表完全隔离；首个超管账号 SHALL 在启动时由 ADMIN_USERNAME/ADMIN_PASSWORD 环境变量种子创建（表为空时），未配置时生成随机密码并输出到启动日志。

#### Scenario: 首次启动种子

- WHEN admins 表为空且配置了 ADMIN_USERNAME/ADMIN_PASSWORD
- THEN 创建 role=super 的管理员，密码 bcrypt 存储

### Requirement: 管理员登录与令牌

系统 SHALL 提供 `POST /admin/auth/login`（用户名+密码）；成功后签发含 scope=admin 与 role 声明的 JWT，并通过 httpOnly Cookie（SameSite=Lax）下发，同时兼容 Authorization 头；连续鉴权失败 SHALL 返回 401。`POST /admin/auth/logout` SHALL 清除 Cookie。

#### Scenario: 非 scope=admin 的 token 访问管理接口

- WHEN 携带用户 token 或无 token 访问 /admin/*
- THEN 返回 401

### Requirement: 用户管理

管理员 SHALL 可通过 `GET /admin/users` 按手机号/昵称关键词搜索并分页浏览用户（含项目数、成片数）；SHALL 可禁用（status=0）/启用（status=1）用户；SHALL 可重置用户密码（服务端生成随机密码返回一次）；被禁用用户立即失去业务接口访问能力。

#### Scenario: 禁用用户

- WHEN 管理员对某用户执行禁用
- THEN 该用户后续业务请求返回 401，登录被拒绝

### Requirement: 运营数据总览

系统 SHALL 提供 `GET /admin/stats/overview`：返回总用户数、今日新增用户、项目总数、成片总数、合成任务总数、存储占用字节数。

### Requirement: 内容审查

管理员 SHALL 可查看任一用户的项目列表与成片（`GET /admin/users/:id/projects`），SHALL 可下架成片（`DELETE /admin/videos/:id`）；下架仅删除记录与可访问性，不阻塞该用户后续合成。

### Requirement: 管理员账号管理（仅超管）

role=super 的管理员 SHALL 可增删管理员账号、分配角色（super/operator）、停用/启用管理员；operator SHALL 仅能使用用户管理与内容审查功能，访问管理员管理接口返回 403。

#### Scenario: operator 越权

- WHEN operator 调用 POST /admin/admins
- THEN 返回 403
