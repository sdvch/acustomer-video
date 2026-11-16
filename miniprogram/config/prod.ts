import { defineConfig } from '@tarojs/cli'

export default defineConfig({
  env: { NODE_ENV: '"production"' },
  defineConstants: {
    // 替换为已备案的 HTTPS 域名（小程序后台配置 request/uploadFile 合法域名）
    API_BASE: '"https://your-domain.com/api/v1"',
  },
})
