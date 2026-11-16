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
  const res = await Taro.login()
  const resp = await request<any>('/auth/login', {
    method: 'POST',
    data: { platform: getPlatform(), code: res.code },
  }, false)
  token = resp.token
  setToken(token)
  return token
}

function getPlatform(): string {
  // 仅支持微信小程序
  return 'wechat'
}

export async function request<T = any>(path: string, options: {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
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
  }
  throw new Error(body?.message || '请求失败')
}
