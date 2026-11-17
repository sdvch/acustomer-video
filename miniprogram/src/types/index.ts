export interface Asset {
  id: number
  shot_no: number
  type: number // 1图片 2视频
  object_key: string
  preview_url: string
  mime: string
  width: number
  height: number
  duration_ms: number
  sort_order: number
  authorized: boolean
  qc_status?: string
  qc_json?: string
  probe_verified?: boolean
  created_at: string
}

// 服务端质检结果（confirm 返回）
export interface ConfirmResult {
  asset_id: number
  qc_status: 'passed' | 'warning' | 'failed'
  warnings: string[]
  probe: {
    kind: string
    width: number
    height: number
    duration_ms: number
    codec: string
    fps: number
    has_audio: boolean
    size_bytes: number
  }
}

// AI 文案
export interface CopyResult {
  hooks: { text: string; angle: string }[]
  body: string
  cta: string
  tips: string[]
}

export interface ComposeProject {
  id: number
  title: string
  aspect_ratio: string
  target_width: number
  target_height: number
  target_fps: number
  status: string
}

export interface Video {
  id: number
  project_id: number
  job_id: number
  object_key: string
  file_url: string
  poster_url: string
  duration_ms: number
  width: number
  height: number
  fps: number
  created_at: string
}

export interface ComposeJob {
  job_id: number
  status: 'queued' | 'claimed' | 'generating' | 'uploading' | 'succeeded' | 'failed' | 'superseded'
  progress: number
  stage: string
  error_message: string
  result_video_id: number
  result?: Video
}

export interface AuthResult {
  token: string
  dev_mode?: boolean
  user: { id: number; nickname?: string; phone?: string; platform: string; open_id?: string }
}

export interface UserProfile {
  id: number
  nickname: string
  phone: string
  platform: string
  member_tier: string
  month_quota: number
  used_quota: number
  created_at: string
}
