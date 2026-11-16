import Taro from '@tarojs/taro'
import type { PickedAsset, UploadAssetParams } from './types'

export type { PickedAsset }

export function chooseAssets(count = 9): Promise<PickedAsset[]> {
  return new Promise((resolve, reject) => {
    Taro.chooseMedia({
      count,
      mediaType: ['image', 'video'],
      sourceType: ['album', 'camera'],
      sizeType: ['compressed'],
      success: (res) => {
        const list: PickedAsset[] = res.tempFiles.map((f: any) => ({
          path: f.tempFilePath,
          size: f.size || 0,
          type: f.fileType === 'video' ? 'video' : 'image',
          mime: f.fileType === 'video' ? 'video/mp4' : 'image/jpeg',
          durationMs: f.duration ? Math.round(f.duration * 1000) : undefined,
          width: f.width,
          height: f.height,
        }))
        resolve(list)
      },
      fail: reject,
    })
  })
}

export function uploadAsset(params: UploadAssetParams): Promise<void> {
  const { filePath, uploadUrl, mime, apiBase, token, localMode, assetId, objectKey } = params
  return new Promise((resolve, reject) => {
    if (!localMode && uploadUrl) {
      // OSS 预签名 PUT 直传。
      // 注意：wx.uploadFile 仅支持 POST，无法直传预签名 PUT，
      // 改为读取文件内容后用 Taro.request PUT（适合 MVP 阶段的中等体积素材）。
      readFileBuffer(filePath).then((buf) => {
        Taro.request({
          url: uploadUrl,
          method: 'PUT',
          data: buf,
          header: { 'Content-Type': mime },
          success: (r) => {
            if (r.statusCode >= 200 && r.statusCode < 300) resolve()
            else reject(new Error('OSS直传失败:' + r.statusCode))
          },
          fail: reject,
        })
      }).catch(reject)
    } else {
      // 本地降级：POST multipart 到后端
      Taro.uploadFile({
        url: `${apiBase}/assets/upload`,
        filePath,
        name: 'file',
        header: { Authorization: `Bearer ${token}` },
        formData: { asset_id: String(assetId), object_key: objectKey },
        success: (r) => {
          try {
            const body = JSON.parse(r.data)
            body.code === 0 ? resolve() : reject(new Error(body.message))
          } catch (e) { reject(e) }
        },
        fail: reject,
      })
    }
  })
}

function readFileBuffer(filePath: string): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const fsm = Taro.getFileSystemManager()
    fsm.readFile({
      filePath,
      success: (res: any) => resolve(res.data as ArrayBuffer),
      fail: reject,
    })
  })
}

export function saveToAlbum(url: string): Promise<void> {
  return new Promise((resolve, reject) => {
    Taro.downloadFile({
      url,
      success: (res) => {
        const isVideo = /\.(mp4|mov|m4v)$/i.test(url.split('?')[0])
        if (isVideo) {
          Taro.saveVideoToPhotosAlbum({
            filePath: res.tempFilePath,
            success: () => resolve(),
            fail: reject,
          })
        } else {
          Taro.saveImageToPhotosAlbum({
            filePath: res.tempFilePath,
            success: () => resolve(),
            fail: reject,
          })
        }
      },
      fail: reject,
    })
  })
}

export function copyText(text: string): Promise<void> {
  return new Promise((resolve, reject) => {
    Taro.setClipboardData({ data: text, success: () => resolve(), fail: reject })
  })
}
