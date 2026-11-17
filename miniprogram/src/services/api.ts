import { request } from './request'
import type { Asset, ComposeJob, Video, ComposeProject, ConfirmResult, CopyResult, AuthResult, UserProfile } from '../types'

export const api = {
  // 认证
  phoneLogin(phone: string, password: string) {
    return request<AuthResult>('/auth/login-phone', { method: 'POST', data: { phone, password } })
  },
  register(phone: string, password: string, nickname?: string) {
    return request<AuthResult>('/auth/register', { method: 'POST', data: { phone, password, nickname } })
  },
  profile() {
    return request<UserProfile>('/auth/profile')
  },
  bindPhone(phone: string, password: string) {
    return request('/auth/bind-phone', { method: 'PUT', data: { phone, password } })
  },

  // 素材
  presign(data: { filename: string; mime: string; type: number; shot_no: number }) {
    return request<{ asset_id: number; object_key: string; upload_url: string; local_mode: boolean }>(
      '/assets/presign', { method: 'POST', data })
  },
  confirm(data: {
    asset_id: number; width: number; height: number; duration_ms: number
    size_bytes: number; tags_json: string; authorized: boolean
  }) {
    return request<ConfirmResult>('/assets/confirm', { method: 'POST', data })
  },
  listAssets() {
    return request<Asset[]>('/assets')
  },
  sortAssets(ids: number[]) {
    return request('/assets/sort', { method: 'PATCH', data: { ids } })
  },
  deleteAsset(id: number) {
    return request(`/assets/${id}`, { method: 'DELETE' })
  },

  // 项目 / 合成
  createProject(title: string, industryId = 0) {
    return request<{ project_id: number }>('/projects', { method: 'POST', data: { title, industry_id: industryId } })
  },
  listProjects() {
    return request<any[]>('/projects')
  },
  getProject(id: number) {
    return request<{ project: ComposeProject; assets: Asset[]; latest_video: Video }>(`/projects/${id}`)
  },
  compose(projectId: number, assetIds: number[], clipSeconds = 4) {
    return request<{ job_id: number }>(`/projects/${projectId}/compose`, {
      method: 'POST',
      data: { asset_ids: assetIds, clip_seconds: clipSeconds, fit_mode: 'pad' },
    })
  },
  getJob(id: number) {
    return request<ComposeJob>(`/jobs/${id}`)
  },

  // 成片
  listVideos() {
    return request<Video[]>('/videos')
  },
  downloadUrl(id: number) {
    return request<{ url: string; filename: string }>(`/videos/${id}/download-url`)
  },
  deleteVideo(id: number) {
    return request(`/videos/${id}`, { method: 'DELETE' })
  },

  // AI 文案（SOP 阶段五）
  generateCopy(data: {
    project_id?: number; title?: string; industry?: string
    store_name?: string; selling_points?: string[]; asset_count?: number
  }) {
    return request<CopyResult>('/ai/copy', { method: 'POST', data })
  },
}
