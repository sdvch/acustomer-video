import Taro from '@tarojs/taro'

const TOKEN_KEY = 'acustomer_token'

export function getToken(): string {
  return Taro.getStorageSync(TOKEN_KEY) || ''
}

export function setToken(token: string) {
  Taro.setStorageSync(TOKEN_KEY, token)
}

export async function ensureLogin(): Promise<string> {
  let token = getToken()
  if (token) return token
  // H5：不再静默 dev 登录，引导到注册/登录页
  if (process.env.TARO_ENV === 'h5') {
    Taro.navigateTo({ url: '/pages/login/login' })
    throw new Error('请先登录')
  }
  let code = ''
  try {
    const res = await Taro.login()
    code = res.code
  } catch {
    code = ''
  }
  const resp = await request<any>('/auth/login', {
    method: 'POST',
    data: { platform: getPlatform(), code: code || 'h5-preview' },
  }, false)
  token = resp.token
  setToken(token)
  return token
}

function getPlatform(): string {
  // weapp 走微信登录；h5 无微信登录能力，后端会落到 dev 用户（配置 WECHAT_APP_ID 后仅 weapp 走真实登录）
  return process.env.TARO_ENV === 'h5' ? 'h5' : 'wechat'
}

export async function request<T = any>(path: string, options: {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
  data?: any
} = {}, needAuth = true): Promise<T> {
  const header: any = { 'Content-Type': 'application/json' }
  if (needAuth) {
    // 首次进入或 token 缺失时先静默登录
    let token = getToken()
    if (!token) {
      try { token = await ensureLogin() } catch (e) { /* 交给后续 401 处理 */ }
    }
    if (token) header.Authorization = `Bearer ${token}`
  }
  const resp = await Taro.request({
    url: API_BASE + path,
    method: options.method || 'GET',
    data: options.data,
    header,
  })
  const body = resp.data as any
  if (body && body.code === 0) return body.data as T
  if (body && body.code === 401) {
    Taro.removeStorageSync(TOKEN_KEY)
    // H5 统一跳登录页；小程序下次请求会重新静默登录
    if (process.env.TARO_ENV === 'h5') {
      Taro.navigateTo({ url: '/pages/login/login' })
    }
  }
  throw new Error(body?.message || '请求失败')
}
