import Taro from '@tarojs/taro'
import { View, Text, Button } from '@tarojs/components'
import { getPlatformName } from '../../platform'
import './mine.scss'

export default function Mine() {
  return (
    <View className='page mine'>
      <View className='card profile'>
        <View className='avatar'>客</View>
        <View>
          <Text className='nickname'>门店用户</Text>
          <Text className='muted'>平台：{getPlatformName()}</Text>
        </View>
      </View>
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
