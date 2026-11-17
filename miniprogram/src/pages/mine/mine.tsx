import Taro from '@tarojs/taro'
import { useState } from 'react'
import { View, Text, Button } from '@tarojs/components'
import { useDidShow } from '@tarojs/taro'
import { api } from '../../services/api'
import { getToken, setToken } from '../../services/request'
import type { UserProfile } from '../../types'
import './mine.scss'

export default function Mine() {
  const [profile, setProfile] = useState<UserProfile | null>(null)

  useDidShow(() => {
    if (!getToken()) {
      setProfile(null)
      return
    }
    api.profile()
      .then(setProfile)
      .catch(() => setProfile(null))
  })

  const logout = () => {
    setToken('')
    Taro.removeStorageSync('acustomer_token')
    setProfile(null)
    Taro.showToast({ title: '已退出登录', icon: 'none' })
  }

  return (
    <View className='page mine'>
      {profile ? (
        <View className='card profile'>
          <View className='avatar'>{(profile.nickname || '客')[0]}</View>
          <View>
            <Text className='nickname'>{profile.nickname || '门店用户'}</Text>
            <Text className='muted'>
              {profile.phone ? `手机：${profile.phone}` : `平台：${profile.platform}`}
            </Text>
            <Text className='muted'>本月配额：{profile.used_quota}/{profile.month_quota}</Text>
          </View>
        </View>
      ) : (
        <View className='card profile'>
          <View className='avatar'>客</View>
          <View style={{ flex: 1 }}>
            <Text className='nickname'>未登录</Text>
            <Text className='muted'>登录后可保存作品与进度</Text>
          </View>
          <Button
            size='mini'
            className='login-btn'
            onClick={() => Taro.navigateTo({ url: '/pages/login/login' })}
          >
            登录 / 注册
          </Button>
        </View>
      )}

      {profile && (
        <View className='card'>
          <Button className='logout-btn' onClick={logout}>
            退出登录
          </Button>
        </View>
      )}

      <View className='card'>
        <View className='section-title'>使用说明</View>
        <Text className='muted tip-line'>· 上传图片/视频素材，勾选授权后即可合成</Text>
        <Text className='muted tip-line'>· 成片规格 1080×1920 竖屏，可保存到相册</Text>
        <Text className='muted tip-line'>· 字幕与 BGM 功能即将上线</Text>
      </View>
      <View className='card'>
        <View className='section-title'>合规提示</View>
        <Text className='muted tip-line'>本工具仅辅助生成视频素材，不代发布、不留存联系方式。请遵守各平台内容规范，勿使用未授权素材。</Text>
      </View>
    </View>
  )
}
