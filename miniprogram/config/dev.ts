import { defineConfig } from '@tarojs/cli'

export default defineConfig({
  env: { NODE_ENV: '"development"' },
  defineConstants: {
    API_BASE: '"http://localhost:18080/api/v1"',
  },
})
