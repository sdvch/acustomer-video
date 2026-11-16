// H5（浏览器）平台实现：与 wechat.ts 接口完全一致，用原生 Web API 替代微信 API
// - chooseAssets: <input type="file"> 选文件，读宽高/时长做客户端预估
// - uploadAsset:   OSS 预签名 PUT 直传（fetch），本地模式 POST multipart
// - saveToAlbum:   <a download> 触发浏览器下载
// - copyText:      navigator.clipboard（老浏览器降级 execCommand）
import type { PickedAsset, UploadAssetParams } from './types'

export function chooseAssets(count = 9): Promise<PickedAsset[]> {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input')
    input.type = 'file'
    input.accept = 'image/*,video/*'
    input.multiple = count > 1
    input.style.display = 'none'
    document.body.appendChild(input)

    let settled = false
    const cleanup = () => { input.remove() }

    input.addEventListener('change', async () => {
      const files = Array.from(input.files || []).slice(0, count)
      cleanup()
      if (settled) return
      settled = true
      if (!files.length) { resolve([]); return }
      const list: PickedAsset[] = files.map((f) => {
        const isVideo = f.type.startsWith('video/')
        return {
          path: URL.createObjectURL(f),
          size: f.size,
          type: isVideo ? 'video' : 'image',
          mime: f.type || (isVideo ? 'video/mp4' : 'image/jpeg'),
          file: f,
        }
      })
      // 客户端 best-effort 元数据：读不到也不影响（服务端 ffprobe 会复核纠正）
      await Promise.all(list.map((p) => probeMeta(p).catch(() => {})))
      resolve(list)
    })

    // 用户取消选择（现代浏览器触发 cancel 事件）
    input.addEventListener('cancel', () => {
      cleanup()
      if (settled) return
      settled = true
      resolve([])
    })

    try {
      input.click()
    } catch (e) {
      cleanup()
      reject(e)
    }
  })
}

async function probeMeta(p: PickedAsset): Promise<void> {
  if (p.type === 'image') {
    await new Promise<void>((res) => {
      const img = new Image()
      img.onload = () => {
        p.width = img.naturalWidth
        p.height = img.naturalHeight
        res()
      }
      img.onerror = () => res()
      img.src = p.path
    })
  } else {
    await new Promise<void>((res) => {
      const v = document.createElement('video')
      v.preload = 'metadata'
      v.muted = true
      v.onloadedmetadata = () => {
        p.width = v.videoWidth
        p.height = v.videoHeight
        if (Number.isFinite(v.duration) && v.duration > 0) {
          p.durationMs = Math.round(v.duration * 1000)
        }
        res()
      }
      v.onerror = () => res()
      v.src = p.path
    })
  }
}

export async function uploadAsset(params: UploadAssetParams): Promise<void> {
  const { filePath, uploadUrl, mime, apiBase, token, localMode, assetId, objectKey, file } = params

  if (!localMode && uploadUrl) {
    // OSS 预签名 PUT 直传。fetch 直接以 File/Blob 为 body，无 10MB 限制
    const body: Blob = file || await fetchBlob(filePath)
    const resp = await fetch(uploadUrl, {
      method: 'PUT',
      headers: { 'Content-Type': mime },
      body,
    })
    if (!(resp.status >= 200 && resp.status < 300)) {
      throw new Error('OSS直传失败: HTTP ' + resp.status)
    }
    return
  }

  // 本地降级：POST multipart 到后端（与 Taro.uploadFile 的 formData 对齐）
  const fd = new FormData()
  const blob: Blob = file || await fetchBlob(filePath)
  fd.append('file', blob, `asset_${assetId}.${mime.includes('mp4') ? 'mp4' : 'jpg'}`)
  fd.append('asset_id', String(assetId))
  fd.append('object_key', objectKey)

  const resp = await fetch(`${apiBase}/assets/upload`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: fd,
  })
  let body: any = null
  try { body = await resp.json() } catch { /* 非 JSON 响应 */ }
  if (resp.ok && body && body.code === 0) return
  throw new Error(body?.message || `上传失败: HTTP ${resp.status}`)
}

async function fetchBlob(url: string): Promise<Blob> {
  const r = await fetch(url)
  if (!r.ok) throw new Error('读取本地文件失败: ' + r.status)
  return r.blob()
}

export function saveToAlbum(url: string): Promise<void> {
  // 浏览器没有"相册"概念，用 <a download> 触发下载到本机
  return new Promise((resolve, reject) => {
    try {
      const a = document.createElement('a')
      a.href = url
      a.download = url.split('/').pop()?.split('?')[0] || 'video.mp4'
      a.target = '_blank'
      a.rel = 'noopener'
      document.body.appendChild(a)
      a.click()
      a.remove()
      resolve()
    } catch (e) {
      reject(e)
    }
  })
}

export async function copyText(text: string): Promise<void> {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(text)
    return
  }
  // http 环境或老浏览器降级
  const ta = document.createElement('textarea')
  ta.value = text
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  try {
    if (!document.execCommand('copy')) throw new Error('复制失败')
  } finally {
    ta.remove()
  }
}
