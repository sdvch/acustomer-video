import { defineConfig } from '@tarojs/cli'

export default defineConfig({
  env: { NODE_ENV: '"development"' },
  // 接口地址统一在 config/index.ts 定义（此处的 defineConstants 会被覆盖）
})
