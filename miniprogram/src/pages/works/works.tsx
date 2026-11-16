import { useState } from 'react'
import Taro, { useDidShow } from '@tarojs/taro'
import { View, Text, Button, Video } from '@tarojs/components'
import { api } from '../../services/api'
import { saveToAlbum, copyText, getPlatformName } from '../../platform'
import type { Video as VideoType } from '../../types'
import './works.scss'

export default function Works() {
  const [videos, setVideos] = useState<VideoType[]>([])
  const [loading, setLoading] = useState(true)

  useDidShow(() => {
    api.listVideos().then((list) => setVideos(list || [])).catch(() => {}).finally(() => setLoading(false))
  })

  async function handleSave(v: VideoType) {
    const isH5 = getPlatformName() === 'h5'
    try {
      Taro.showLoading({ title: isH5 ? '下载中' : '保存中' })
      await saveToAlbum(v.file_url)
      Taro.hideLoading()
      Taro.showToast({ title: isH5 ? '已开始下载到本机' : '已保存到相册', icon: 'success' })
    } catch (e) {
      Taro.hideLoading()
      Taro.showToast({ title: isH5 ? '下载失败' : '保存失败，请检查相册权限', icon: 'none' })
    }
  }

  return (
    <View className='page works'>
      <View className='section-title'>我的作品</View>
      {loading && <Text className='muted'>加载中...</Text>}
      {!loading && videos.length === 0 && (
        <View className='card empty'>
          <Text className='muted'>还没有成片，去首页制作第一个获客视频吧</Text>
          <Button className='btn-primary' onClick={() => Taro.switchTab({ url: '/pages/index/index' })}>去制作</Button>
        </View>
      )}
      {videos.map((v) => (
        <View className='card work-item' key={v.id}>
          <Video className='work-video' src={v.file_url} poster={v.poster_url} controls />
          <Text className='work-meta'>{v.width}×{v.height} · {Math.round(v.duration_ms / 1000)}s · {new Date(v.created_at).toLocaleString()}</Text>
          <View className='work-actions'>
            <Button className='btn-primary' onClick={() => handleSave(v)}>保存到相册</Button>
            <Button className='btn-secondary' onClick={() => { copyText(v.file_url).then(() => Taro.showToast({ title: '链接已复制', icon: 'none' })).catch(() => {}) }}>复制链接</Button>
          </View>
        </View>
      ))}
    </View>
  )
}
