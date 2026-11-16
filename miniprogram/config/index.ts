import { defineConfig } from '@tarojs/cli'

// 接口地址集中在此定义：dev/prod.ts 里的 defineConstants 会被本文件合并规则吃掉，
// 所以不要分散写，否则编译出来的 API_BASE 是空串（请求会打到相对路径）。
const isProd = process.env.NODE_ENV === 'production'
const API_BASE =
  process.env.API_BASE ||
  (isProd ? 'https://your-domain.com/api/v1' : 'http://localhost:18080/api/v1')

export default defineConfig({
  projectName: 'acustomer-video-miniprogram',
  date: '2026-11-15',
  designWidth: 750,
  deviceRatio: { 640: 2.34 / 2, 750: 1, 828: 1.81 / 2, 375: 2 / 1 },
  sourceRoot: 'src',
  // 小程序与 H5 分别输出，避免互相覆盖（h5 仅用于浏览器预览/演示）
  outputRoot: process.env.TARO_ENV === 'h5' ? 'dist/h5' : 'dist/weapp',
  plugins: ['@tarojs/plugin-framework-react'],
  defineConstants: {
    API_BASE: JSON.stringify(API_BASE),
  },
  copy: { patterns: [], options: {} },
  framework: 'react',
  compiler: 'webpack5',
  cache: { enable: false },
  mini: {
    postcss: {
      pxtransform: { enable: true, config: {} },
      cssModules: { enable: false },
    },
  },
  h5: {
    // 部署在 http://<host>/acv-h5/ 子路径下
    publicPath: '/acv-h5/',
    staticDirectory: 'static',
    router: {
      mode: 'hash', // hash 模式便于子路径静态托管，无需 nginx rewrite
    },
  },
})
