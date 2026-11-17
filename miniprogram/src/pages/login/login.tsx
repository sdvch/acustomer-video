import Taro from '@tarojs/taro'
import { useState } from 'react'
import { View, Text, Input, Button } from '@tarojs/components'
import { api } from '../../services/api'
import { setToken } from '../../services/request'
import './login.scss'

// 登录/注册页：手机号+密码（H5 主登录方式；小程序也可用手机号登录）
export default function Login() {
  const [mode, setMode] = useState<'login' | 'register'>('login')
  const [phone, setPhone] = useState('')
  const [password, setPassword] = useState('')
  const [nickname, setNickname] = useState('')
  const [loading, setLoading] = useState(false)

  const validPhone = (p: string) => /^1[3-9]\d{9}$/.test(p)

  const submit = async () => {
    if (!validPhone(phone)) {
      Taro.showToast({ title: '手机号格式不正确', icon: 'none' })
      return
    }
    if (password.length < 8) {
      Taro.showToast({ title: '密码至少 8 位', icon: 'none' })
      return
    }
    setLoading(true)
    try {
      const data =
        mode === 'login'
          ? await api.phoneLogin(phone, password)
          : await api.register(phone, password, nickname)
      setToken(data.token)
      Taro.showToast({ title: mode === 'login' ? '登录成功' : '注册成功', icon: 'success' })
      setTimeout(() => {
        Taro.reLaunch({ url: '/pages/index/index' })
      }, 600)
    } catch (e: any) {
      Taro.showToast({ title: e.message || '操作失败', icon: 'none' })
    } finally {
      setLoading(false)
    }
  }

  return (
    <View className='page login'>
      <View className='title'>获客视频工坊</View>
      <View className='subtitle'>{mode === 'login' ? '手机号登录' : '注册新账号'}</View>

      <View className='form'>
        <Input
          className='input'
          type='number'
          maxlength={11}
          placeholder='请输入手机号'
          value={phone}
          onInput={(e) => setPhone(e.detail.value)}
        />
        <Input
          className='input'
          password
          placeholder='密码（至少 8 位）'
          value={password}
          onInput={(e) => setPassword(e.detail.value)}
        />
        {mode === 'register' && (
          <Input
            className='input'
            placeholder='昵称（选填）'
            value={nickname}
            onInput={(e) => setNickname(e.detail.value)}
          />
        )}
        <Button
          className='submit'
          loading={loading}
          disabled={loading}
          onClick={submit}
        >
          {mode === 'login' ? '登录' : '注册'}
        </Button>
        <View className='switch' onClick={() => setMode(mode === 'login' ? 'register' : 'login')}>
          <Text>
            {mode === 'login' ? '还没有账号？去注册' : '已有账号？去登录'}
          </Text>
        </View>
      </View>

      <View className='tip'>
        <Text className='muted'>小程序端也可使用微信一键登录；H5 请使用手机号注册</Text>
      </View>
    </View>
  )
}
