# auth 规格变更：新增手机号注册登录

## ADDED Requirements

### Requirement: 手机号注册

系统 SHALL 提供 `POST /auth/register`：入参 phone（11 位大陆手机号格式）+ password（≥8 位）+ 可选 nickname；校验失败返回 400 与中文原因；phone 已被占用返回 409；成功后创建用户（platform=phone，bcrypt 哈希存储密码）并签发与微信登录同构的 JWT。

#### Scenario: 正常注册

- WHEN 提交合法手机号与 8 位以上密码
- THEN 返回 code=0，data 含 token 与 user（手机号脱敏展示）

#### Scenario: 手机号已注册

- WHEN 提交的手机号已存在
- THEN 返回 409「该手机号已注册，请直接登录」

### Requirement: 手机号密码登录

系统 SHALL 提供 `POST /auth/login-phone`：校验手机号+密码；密码错误返回 401；用户被禁用（status=0）返回 403「账号已被禁用」且不签发 token。

#### Scenario: 登录成功

- WHEN 手机号与密码正确且用户正常
- THEN 返回 token，前端写入本地存储

#### Scenario: 被禁用用户登录

- WHEN 用户 status=0
- THEN 返回 403，不签发 token

### Requirement: 微信用户绑定手机号

已登录微信用户 SHALL 可通过 `PUT /auth/bind-phone` 绑定手机号并设置密码；绑定后可用手机号+密码登录同一账号；手机号被其他账号占用时返回 409。

### Requirement: 禁用用户全链路拦截

用户被管理员禁用后，其后续业务请求 SHALL 在 JWT 中间件处被拒绝（401「账号已被禁用」），无论 token 是否仍在有效期内。

#### Scenario: 禁用后携带有效 token 访问

- WHEN status=0 的用户携带有效 JWT 调用任一业务接口
- THEN 返回 401，前端清理 token 并跳转登录页

### Requirement: token 作用域隔离

管理端 token（scope=admin）SHALL NOT 能访问用户侧业务接口；用户 token SHALL NOT 能访问 /admin/* 路由。旧版无 scope 声明的用户 token SHALL 继续被视为用户 token（向后兼容）。
