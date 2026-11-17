export default defineAppConfig({
  pages: [
    'pages/index/index',
    'pages/upload/upload',
    'pages/editor/editor',
    'pages/progress/progress',
    'pages/works/works',
    'pages/mine/mine',
    'pages/login/login',
  ],
  window: {
    backgroundTextStyle: 'dark',
    navigationBarBackgroundColor: '#1a1a2e',
    navigationBarTitleText: '获客视频工坊',
    navigationBarTextStyle: 'white',
    backgroundColor: '#f5f6fa',
  },
  tabBar: {
    color: '#8a8a99',
    selectedColor: '#6c5ce7',
    backgroundColor: '#ffffff',
    borderStyle: 'black',
    list: [
      { pagePath: 'pages/index/index', text: '首页' },
      { pagePath: 'pages/works/works', text: '作品' },
      { pagePath: 'pages/mine/mine', text: '我的' },
    ],
  },
})
