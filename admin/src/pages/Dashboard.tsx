import { useEffect, useState } from 'react'
import { Card, Col, Row, Statistic, Spin } from 'antd'
import {
  UserOutlined,
  FolderOpenOutlined,
  VideoCameraOutlined,
  HddOutlined,
  RiseOutlined,
} from '@ant-design/icons'
import { api, formatBytes } from '../api'

interface Overview {
  total_users: number
  today_users: number
  total_projects: number
  total_videos: number
  total_jobs: number
  storage_bytes: number
}

export default function Dashboard() {
  const [data, setData] = useState<Overview | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    api<Overview>('/admin/stats/overview')
      .then(setData)
      .finally(() => setLoading(false))
  }, [])

  if (loading) return <Spin size="large" style={{ display: 'block', margin: '80px auto' }} />
  if (!data) return <Card>暂无数据</Card>

  return (
    <Row gutter={[16, 16]}>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="总用户数（正常）"
            value={data.total_users}
            prefix={<UserOutlined />}
          />
        </Card>
      </Col>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="今日新增用户"
            value={data.today_users}
            prefix={<RiseOutlined />}
            valueStyle={{ color: '#3f8600' }}
          />
        </Card>
      </Col>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="合成项目总数"
            value={data.total_projects}
            prefix={<FolderOpenOutlined />}
          />
        </Card>
      </Col>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="成片总数"
            value={data.total_videos}
            prefix={<VideoCameraOutlined />}
          />
        </Card>
      </Col>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="合成任务总数"
            value={data.total_jobs}
            prefix={<VideoCameraOutlined />}
          />
        </Card>
      </Col>
      <Col xs={12} md={8} lg={6}>
        <Card>
          <Statistic
            title="素材存储占用"
            value={formatBytes(data.storage_bytes)}
            prefix={<HddOutlined />}
          />
        </Card>
      </Col>
    </Row>
  )
}
