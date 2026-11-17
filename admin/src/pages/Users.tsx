import { useCallback, useEffect, useState } from 'react'
import {
  Button,
  Drawer,
  Form,
  Input,
  InputNumber,
  message,
  Modal,
  Popconfirm,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd'
import { api } from '../api'

interface UserRow {
  id: number
  open_id: string
  platform: string
  nickname: string
  phone: string | null
  status: number
  member_tier: string
  month_quota: number
  used_quota: number
  created_at: string
  project_count: number
  video_count: number
}

interface VideoRow {
  id: number
  project_id: number
  file_url: string
  duration_ms: number
  width: number
  height: number
  created_at: string
}

interface ProjectRow {
  id: number
  title: string
  status: string
  created_at: string
}

export default function Users() {
  const [rows, setRows] = useState<UserRow[]>([])
  const [total, setTotal] = useState(0)
  const [page, setPage] = useState(1)
  const [keyword, setKeyword] = useState('')
  const [loading, setLoading] = useState(false)

  // 详情抽屉（内容审查）
  const [detail, setDetail] = useState<UserRow | null>(null)
  const [detailData, setDetailData] = useState<{ projects: ProjectRow[]; videos: VideoRow[] } | null>(null)
  const [detailLoading, setDetailLoading] = useState(false)
  // 重置密码弹窗
  const [resetForm] = Form.useForm()
  const [resetTarget, setResetTarget] = useState<UserRow | null>(null)
  const [resetPwdResult, setResetPwdResult] = useState<string | null>(null)

  const load = useCallback(async (p = page, kw = keyword) => {
    setLoading(true)
    try {
      const data = await api<{ list: UserRow[]; total: number }>(
        `/admin/users?page=${p}&size=20&keyword=${encodeURIComponent(kw)}`,
      )
      setRows(data.list || [])
      setTotal(data.total || 0)
    } catch (e: any) {
      message.error(e.message)
    } finally {
      setLoading(false)
    }
  }, [page, keyword])

  useEffect(() => {
    load()
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [page])

  const toggleStatus = async (row: UserRow) => {
    try {
      await api(`/admin/users/${row.id}/status`, {
        method: 'PATCH',
        data: { status: row.status === 1 ? 0 : 1 },
      })
      message.success(row.status === 1 ? '已禁用' : '已启用')
      load()
    } catch (e: any) {
      message.error(e.message)
    }
  }

  const openDetail = async (row: UserRow) => {
    setDetail(row)
    setDetailLoading(true)
    setDetailData(null)
    try {
      const data = await api<{ projects: ProjectRow[]; videos: VideoRow[] }>(
        `/admin/users/${row.id}/projects`,
      )
      setDetailData(data)
    } catch (e: any) {
      message.error(e.message)
    } finally {
      setDetailLoading(false)
    }
  }

  const takedown = async (videoId: number) => {
    try {
      await api(`/admin/videos/${videoId}`, { method: 'DELETE' })
      message.success('已下架')
      if (detail) openDetail(detail)
    } catch (e: any) {
      message.error(e.message)
    }
  }

  const doReset = async () => {
    if (!resetTarget) return
    try {
      const data = await api<{ new_password: string }>(
        `/admin/users/${resetTarget.id}/reset-password`,
        { method: 'POST' },
      )
      setResetPwdResult(data.new_password)
      load()
    } catch (e: any) {
      message.error(e.message)
    }
  }

  const columns = [
    { title: 'ID', dataIndex: 'id', width: 70 },
    { title: '昵称', dataIndex: 'nickname' },
    {
      title: '手机号',
      dataIndex: 'phone',
      render: (v: string | null) => v || <Tag>未绑定</Tag>,
    },
    { title: '平台', dataIndex: 'platform', width: 90 },
    { title: '项目', dataIndex: 'project_count', width: 70 },
    { title: '成片', dataIndex: 'video_count', width: 70 },
    {
      title: '配额',
      width: 100,
      render: (_: any, r: UserRow) => `${r.used_quota}/${r.month_quota}`,
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 90,
      render: (s: number) =>
        s === 1 ? <Tag color="green">正常</Tag> : <Tag color="red">已禁用</Tag>,
    },
    { title: '注册时间', dataIndex: 'created_at', width: 170 },
    {
      title: '操作',
      width: 240,
      render: (_: any, r: UserRow) => (
        <Space>
          <Button size="small" onClick={() => openDetail(r)}>
            详情
          </Button>
          <Popconfirm
            title={r.status === 1 ? '确认禁用该用户？' : '确认启用该用户？'}
            onConfirm={() => toggleStatus(r)}
          >
            <Button size="small" danger={r.status === 1}>
              {r.status === 1 ? '禁用' : '启用'}
            </Button>
          </Popconfirm>
          <Button size="small" onClick={() => { setResetTarget(r); setResetPwdResult(null); resetForm.resetFields() }}>
            重置密码
          </Button>
        </Space>
      ),
    },
  ]

  return (
    <div>
      <Space style={{ marginBottom: 16 }}>
        <Input.Search
          placeholder="手机号 / 昵称搜索"
          allowClear
          style={{ width: 260 }}
          onSearch={(v) => { setPage(1); setKeyword(v); load(1, v) }}
        />
      </Space>
      <Table
        rowKey="id"
        loading={loading}
        columns={columns as any}
        dataSource={rows}
        pagination={{
          current: page,
          pageSize: 20,
          total,
          showTotal: (t) => `共 ${t} 人`,
          onChange: setPage,
        }}
      />

      <Drawer
        title={detail ? `用户详情 · ${detail.nickname}（${detail.phone || '未绑定手机'}）` : ''}
        open={!!detail}
        onClose={() => setDetail(null)}
        width={680}
      >
        {detailLoading && <p>加载中...</p>}
        {detailData && (
          <>
            <Typography.Title level={5}>项目（{detailData.projects.length}）</Typography.Title>
            <Table
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={detailData.projects}
              columns={[
                { title: 'ID', dataIndex: 'id', width: 70 },
                { title: '标题', dataIndex: 'title' },
                { title: '状态', dataIndex: 'status', width: 100 },
                { title: '创建时间', dataIndex: 'created_at' },
              ]}
              style={{ marginBottom: 24 }}
            />
            <Typography.Title level={5}>成片（{detailData.videos.length}）</Typography.Title>
            <Table
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={detailData.videos}
              columns={[
                { title: 'ID', dataIndex: 'id', width: 70 },
                {
                  title: '规格',
                  width: 150,
                  render: (_: any, v: VideoRow) =>
                    `${v.width}×${v.height} · ${(v.duration_ms / 1000).toFixed(1)}s`,
                },
                { title: '生成时间', dataIndex: 'created_at' },
                {
                  title: '操作',
                  width: 140,
                  render: (_: any, v: VideoRow) => (
                    <Space>
                      <a href={v.file_url} target="_blank" rel="noreferrer">
                        预览
                      </a>
                      <Popconfirm title="确认下架该成片？" onConfirm={() => takedown(v.id)}>
                        <a style={{ color: '#cf1322' }}>下架</a>
                      </Popconfirm>
                    </Space>
                  ),
                },
              ]}
            />
          </>
        )}
      </Drawer>

      <Modal
        title={resetTarget ? `重置密码 · ${resetTarget.nickname}` : ''}
        open={!!resetTarget}
        onCancel={() => setResetTarget(null)}
        footer={resetPwdResult
          ? [<Button key="close" type="primary" onClick={() => setResetTarget(null)}>完成</Button>]
          : [
              <Button key="cancel" onClick={() => setResetTarget(null)}>取消</Button>,
              <Button key="ok" type="primary" onClick={doReset}>确认重置</Button>,
            ]}
      >
        <Form form={resetForm} layout="vertical">
          <Form.Item label="新密码将由服务端随机生成，仅显示一次">
            <InputNumber style={{ display: 'none' }} />
          </Form.Item>
        </Form>
        {resetPwdResult && (
          <Typography.Paragraph copyable={{ text: resetPwdResult }} style={{ fontSize: 18, marginTop: 8 }}>
            新密码：{resetPwdResult}
          </Typography.Paragraph>
        )}
      </Modal>
    </div>
  )
}
