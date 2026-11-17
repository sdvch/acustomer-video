// 管理后台 API 封装：同源 Cookie 鉴权（httpOnly），401 统一跳登录
const API_BASE = (import.meta as any).env?.VITE_API_BASE || '/acv/v1'

export class ApiError extends Error {
  code: number
  constructor(code: number, message: string) {
    super(message)
    this.code = code
  }
}

export async function api<T = any>(path: string, options: {
  method?: 'GET' | 'POST' | 'PATCH' | 'DELETE'
  data?: any
} = {}): Promise<T> {
  const resp = await fetch(API_BASE + path, {
    method: options.method || 'GET',
    headers: { 'Content-Type': 'application/json' },
    credentials: 'same-origin',
    body: options.data !== undefined ? JSON.stringify(options.data) : undefined,
  })
  const body = await resp.json().catch(() => ({ code: -1, message: '响应解析失败' }))
  if (body.code === 0) return body.data as T
  if (resp.status === 401) {
    window.location.hash = '#/login'
    window.location.reload()
    throw new ApiError(401, body.message || '请先登录')
  }
  throw new ApiError(body.code ?? resp.status, body.message || '请求失败')
}

export function formatBytes(n: number): string {
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i === 0 ? 0 : 1)} ${units[i]}`
}
