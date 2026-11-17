import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// 生产挂在 nginx /acv-admin/ 路径下；开发时代理到本地 Go 服务
export default defineConfig({
  plugins: [react()],
  base: '/acv-admin/',
  server: {
    port: 5174,
    proxy: {
      '/api': {
        target: 'http://127.0.0.1:18080',
        changeOrigin: true,
      },
    },
  },
  build: {
    outDir: 'dist',
    chunkSizeWarningLimit: 1200,
  },
})
