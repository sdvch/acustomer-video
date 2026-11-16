import { useEffect } from 'react'
import Taro from '@tarojs/taro'
import { View, Text, Button } from '@tarojs/components'
import { ensureLogin } from '../../services/request'
import './index.scss'

export default function Index() {
  useEffect(() => {
    ensureLogin().catch(() => {})
  }, [])

  return (
    <View className='page home'>
      <View className='hero'>
        <Text className='hero-title'>AI 获客视频工坊</Text>
        <Text className='hero-sub'>图片视频一键拼接，批量生成门店获客短视频</Text>
      </View>

      <View className='card'>
        <View className='section-title'>三步生成你的获客视频</View>
        <View className='step'><Text className='step-no'>1</Text><View><Text className='step-t'>上传素材</Text><Text className='step-d'>选择门店图片/视频，竖屏效果更佳</Text></View></View>
        <View className='step'><Text className='step-no'>2</Text><View><Text className='step-t'>排序编辑</Text><Text className='step-d'>拖动调整顺序，设置片段时长</Text></View></View>
        <View className='step'><Text className='step-no'>3</Text><View><Text className='step-t'>合成下载</Text><Text className='step-d'>一键生成 1080×1920 竖屏成片</Text></View></View>
      </View>

      <Button className='btn-primary' onClick={() => Taro.navigateTo({ url: '/pages/upload/upload' })}>
        开始制作
      </Button>
      <View className='muted home-tip'>提示：素材需勾选授权后方可合成，系统不代发布、不留联系方式</View>
    </View>
  )
}
