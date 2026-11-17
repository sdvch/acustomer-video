import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Form,
  Input,
  message,
  Modal,
  Popconfirm,
  Select,
  Space,
  Table,
  Tag,
} from 'antd'
import { api } from '../api'

interface AdminRow {
  id: number
  username: string
  role: string
  status: number
  last_login_at: string
  created_at: string
}

export default function Admins() {
  const [rows, setRows] = useState<AdminRow[]>([])
  const [loading, setLoading] = useState(false)
  const [form] = Form.useForm()
  const [creating, setCreating] = useState(false)

  const load = useCallback(async () => {
    setLoading(true)
    try {
      const data = await api<AdminRow[]>('/admin/admins')
      setRows(data || [])
    } catch (e: any) {
      message.error(e.message)
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    load()
  }, [load])

  const create = async () => {
    const values = await form.validateFields()
    try {
      await api('/admin/admins', { method: 'POST', data: values })
      message.success('已创建')
      setCreating(false)
      form.resetFields()
      load()
    } catch (e: any) {
      message.error(e.message)
    }
  }

  const update = async (row: AdminRow, patch: any) => {
    try {
      await api(`/admin/admins/${row.id}`, { method: 'PATCH', data: patch })
      message.success('已更新')
      load()
    } catch (e: any) {
      message.error(e.message)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '用户名', dataIndex: 'username' },
    {
      title: '角色',
      dataIndex: 'role',
      width: 130,
      render: (r: string) =>
        r === 'super' ? <Tag color="purple">超级管理员</Tag> : <Tag color="blue">运营</Tag>,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (s: number) =>
        s === 1 ? <Tag color="green">正常</Tag> : <Tag color="red">已停用</Tag>,
    },
    { title: '最后登录', dataIndex: 'last_login_at', width: 170 },
    { title: '创建时间', dataIndex: 'created_at', width: 170 },
    {
      title: '操作',
      width: 260,
      render: (_: any, r: AdminRow) => (
        <Space>
          <Popconfirm
            title={r.status === 1 ? '确认停用？' : '确认启用？'}
            onConfirm={() => update(r, { status: r.status === 1 ? 0 : 1 })}
          >
            <Button size="small" danger={r.status === 1}>
              {r.status === 1 ? '停用' : '启用'}
            </Button>
          </Popconfirm>
          <Popconfirm
            title={r.role === 'super' ? '降为运营？' : '升为超管？'}
            onConfirm={() => update(r, { role: r.role === 'super' ? 'operator' : 'super' })}
          >
            <Button size="small">{r.role === 'super' ? '降为运营' : '升为超管'}</Button>
          </Popconfirm>
          <Popconfirm
            title="重置为输入框中的新密码？"
            description={
              <Input
                id={`pwd-${r.id}`}
                placeholder="新密码（≥8位）"
                style={{ marginTop: 8 }}
              />
            }
            onConfirm={() => {
              const el = document.getElementById(`pwd-${r.id}`) as HTMLInputElement | null
              const pwd = el?.value || ''
              if (pwd.length < 8) {
                message.error('密码至少 8 位')
                return
              }
              update(r, { password: pwd })
            }}
          >
            <Button size="small">重置密码</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Button type="primary" style={{ marginBottom: 16 }} onClick={() => setCreating(true)}>
        新增管理员
      </Button>
      <Table rowKey="id" loading={loading} columns={columns as any} dataSource={rows} pagination={false} />

      <Modal
        title="新增管理员"
        open={creating}
        onCancel={() => setCreating(false)}
        onOk={create}
        okText="创建"
        cancelText="取消"
      >
        <Form form={form} layout="vertical">
          <Form.Item name="username" label="用户名" rules={[{ required: true }]}>
            <Input placeholder="用户名" />
          </Form.Item>
          <Form.Item name="password" label="密码" rules={[{ required: true }, { min: 8, message: '密码至少 8 位' }]}>
            <Input.Password placeholder="至少 8 位" />
          </Form.Item>
          <Form.Item name="role" label="角色" initialValue="operator" rules={[{ required: true }]}>
            <Select
              options={[
                { value: 'operator', label: '运营（用户管理与内容审查）' },
                { value: 'super', label: '超级管理员（全部权限）' },
              ]}
            />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  )
}
