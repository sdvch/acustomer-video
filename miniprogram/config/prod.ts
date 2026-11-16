import { defineConfig } from '@tarojs/cli'

export default defineConfig({
  env: { NODE_ENV: '"production"' },
  // 接口地址统一在 config/index.ts 定义，构建时用 API_BASE 环境变量注入
})
