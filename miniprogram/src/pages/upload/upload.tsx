import { useState } from 'react'
import Taro from '@tarojs/taro'
import { View, Text, Button } from '@tarojs/components'
import { chooseAssets, uploadAsset, type PickedAsset } from '../../platform'
import { api } from '../../services/api'
import { getToken } from '../../services/request'
import { useStore } from '../../store'
import './upload.scss'

export default function Upload() {
  const [uploading, setUploading] = useState(false)
  const [progressText, setProgressText] = useState('')
  const [warnings, setWarnings] = useState<string[]>([])
  const [done, setDone] = useState(false)
  const setAssets = useStore((s) => s.setAssets)
  const addAssets = useStore((s) => s.addAssets)

  async function handleChoose() {
    try {
      const picked = await chooseAssets(9)
      if (!picked.length) return
      setUploading(true)
      setWarnings([])
      setDone(false)
      const created: any[] = []
      const allWarnings: string[] = []
      for (let i = 0; i < picked.length; i++) {
        const p: PickedAsset = picked[i]
        setProgressText(`上传中 ${i + 1}/${picked.length}`)
        const type = p.type === 'video' ? 2 : 1
        const ps = await api.presign({
          filename: `asset_${Date.now()}.${p.mime.includes('mp4') ? 'mp4' : 'jpg'}`,
          mime: p.mime, type, shot_no: i + 1,
        })
        await uploadAsset({
          filePath: p.path,
          uploadUrl: ps.upload_url,
          objectKey: ps.object_key,
          assetId: ps.asset_id,
          mime: p.mime,
          token: getToken(),
          apiBase: API_BASE,
          localMode: ps.local_mode,
          file: p.file, // h5 端：原生 File，避免二次读取
        })
        setProgressText(`质检中 ${i + 1}/${picked.length}`)
        // 服务端用 ffprobe 复核真实宽高/时长，并返回质检警告
        const qc = await api.confirm({
          asset_id: ps.asset_id,
          width: p.width || 0,
          height: p.height || 0,
          duration_ms: p.durationMs || 0,
          size_bytes: p.size,
          tags_json: '{}',
          authorized: true,
        })
        const probe = qc?.probe
        if (qc?.warnings?.length) {
          allWarnings.push(...qc.warnings.map((w: string) => `镜头 ${i + 1}：${w}`))
        }
        created.push({
          id: ps.asset_id, shot_no: i + 1, type, object_key: ps.object_key,
          preview_url: '', mime: p.mime,
          width: probe?.width || p.width || 1080,
          height: probe?.height || p.height || 1920,
          duration_ms: probe?.duration_ms || p.durationMs || 0,
          sort_order: i, authorized: true, qc_status: qc?.qc_status, created_at: '',
        })
      }
      addAssets(created)
      if (allWarnings.length) {
        setWarnings(allWarnings)
        setDone(true)
      } else {
        Taro.showToast({ title: `已上传 ${created.length} 个素材`, icon: 'success' })
        Taro.navigateTo({ url: '/pages/editor/editor' })
      }
    } catch (e: any) {
      Taro.showToast({ title: e.message || '上传失败', icon: 'none' })
    } finally {
      setUploading(false)
      setProgressText('')
    }
  }

  async function handleExisting() {
    try {
      const list = await api.listAssets()
      setAssets(list || [])
      Taro.navigateTo({ url: '/pages/editor/editor' })
    } catch (e: any) {
      Taro.showToast({ title: e.message || '加载失败', icon: 'none' })
    }
  }

  return (
    <View className='page upload'>
      <View className='section-title'>上传素材</View>
      <View className='card'>
        <Text className='muted'>支持图片与视频混传，建议竖屏 9:16 拍摄，单个镜头 3-5 秒。横屏素材将自动补黑边。</Text>
      </View>
      <Button className='btn-primary' onClick={handleChoose} disabled={uploading}>
        {uploading ? progressText : '选择图片 / 视频'}
      </Button>
      <Button className='btn-secondary' onClick={handleExisting}>使用已上传的素材</Button>

      {done && warnings.length > 0 && (
        <View className='card qc-card'>
          <Text className='qc-title'>素材质检提示（{warnings.length}）</Text>
          {warnings.map((w, i) => (
            <Text className='qc-item' key={i}>· {w}</Text>
          ))}
          <Button className='btn-primary' onClick={() => Taro.navigateTo({ url: '/pages/editor/editor' })}>
            知道了，继续编辑
          </Button>
        </View>
      )}
    </View>
  )
}
