# dual-platform

## Purpose

小程序与 H5 双端功能对等：平台能力全部收敛在 src/platform/ 适配层（TARO_ENV 构建期分流），页面禁止平台分支，API_BASE 单点定义、构建时注入。

## Requirements

### Requirement: 平台能力走适配层

系统 SHALL 将平台能力（选素材、上传、保存相册/下载、复制）统一封装在 miniprogram/src/platform/（wechat.ts/h5.ts，构建期按 TARO_ENV 分流，index.ts 统一出口）；页面与组件代码 MUST NOT 出现 `if (process.env.TARO_ENV === 'h5')` 式分支。

#### Scenario: 新增平台能力

- **WHEN** 需要新增一个平台相关能力
- **THEN** 在 src/platform/ 的 types.ts 定义契约、双端各自实现，页面统一 import 自 src/platform。

### Requirement: 双端功能对等

系统 SHALL 保证小程序与 H5 的核心链路对等：选素材 → 上传 → 质检 → 编辑 → 合成 → 成片获取。已知允许的差异：H5"保存到相册"为浏览器下载；H5 登录走 dev 单用户（配置 WECHAT_APP_ID 后小程序走真实微信登录）。

#### Scenario: H5 全链路可用

- **WHEN** 用户在浏览器打开 H5 端
- **THEN** 可完成选素材（input[type=file]）、上传、质检、合成、下载全链路，无白屏或功能缺失。

### Requirement: API_BASE 单点定义

API 基地址 SHALL 仅在 miniprogram/config/index.ts 定义；H5 构建时通过 `API_BASE=http://<host>/acv/v1` 环境变量注入，不得在代码中散落。

#### Scenario: 构建注入生效

- **WHEN** 以 API_BASE 环境变量执行 build:h5
- **THEN** 产物中的接口请求地址全部指向该值。
