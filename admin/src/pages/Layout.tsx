import { useEffect, useState } from 'react'
import { Layout as AntLayout, Menu, Button, Space, Tag, message } from 'antd'
import { Outlet, useNavigate, useLocation } from 'react-router-dom'
import { api } from '../api'

const { Sider, Header, Content } = AntLayout

export default function Layout() {
  const nav = useNavigate()
  const loc = useLocation()
  const [me, setMe] = useState<{ username: string; role: string } | null>(null)

  useEffect(() => {
    api<{ username: string; role: string }>('/admin/auth/profile')
      .then(setMe)
      .catch(() => {})
  }, [])

  const logout = async () => {
    try {
      await api('/admin/auth/logout', { method: 'POST' })
    } catch { /* 忽略登出失败 */ }
    message.success('已退出')
    nav('/login')
  }

  return (
    <AntLayout style={{ minHeight: '100vh' }}>
      <Sider theme="dark">
        <div style={{ color: '#fff', padding: 20, fontWeight: 600, fontSize: 15 }}>获客视频后台</div>
        <Menu
          theme="dark"
          mode="inline"
          selectedKeys={[loc.pathname]}
          onClick={({ key }) => nav(key)}
          items={[
            { key: '/dashboard', label: '数据总览' },
            { key: '/users', label: '用户管理' },
            ...(me?.role === 'super' ? [{ key: '/admins', label: '管理员' }] : []),
          ]}
        />
      </Sider>
      <AntLayout>
        <Header
          style={{
            background: '#fff',
            display: 'flex',
            justifyContent: 'flex-end',
            alignItems: 'center',
            paddingInline: 24,
          }}
        >
          <Space>
            {me && (
              <>
                <span>{me.username}</span>
                <Tag color={me.role === 'super' ? 'purple' : 'blue'}>
                  {me.role === 'super' ? '超级管理员' : '运营'}
                </Tag>
              </>
            )}
            <Button size="small" onClick={logout}>
              退出
            </Button>
          </Space>
        </Header>
        <Content style={{ margin: 16 }}>
          <Outlet />
        </Content>
      </AntLayout>
    </AntLayout>
  )
}
