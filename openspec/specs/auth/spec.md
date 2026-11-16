# auth

## Purpose

认证与接口保护：业务接口一律 JWT 鉴权，支持微信 code2session 真实登录与未配置 AppID 时的本地开发单用户降级，两种模式均不阻断核心链路。

## Requirements

### Requirement: 业务接口必须鉴权

系统 SHALL 对全部业务接口启用 JWT 鉴权（authed 中间件），未携带或无效 token 的请求 MUST 被拒绝；健康检查等公开接口除外。

#### Scenario: 无 token 访问业务接口

- **WHEN** 客户端未带 Authorization 头调用 /api/v1/projects
- **THEN** 返回 401，不返回业务数据。

### Requirement: 微信登录与开发降级

系统 SHALL 支持 wx.login → POST /auth/login → 服务端 jscode2session 换 openid，按 (open_id, platform) 落 users 表并签发 7 天 JWT；WECHAT_APP_ID 未配置时 SHALL 降级为本地开发单用户，不阻断开发流程。

#### Scenario: 未配置 AppID 的开发登录

- **WHEN** 服务未配置 WECHAT_APP_ID 且客户端请求登录
- **THEN** 签发 dev 用户 JWT，全链路可用。
