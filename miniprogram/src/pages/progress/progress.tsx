import { useEffect, useRef, useState } from 'react'
import Taro, { useRouter } from '@tarojs/taro'
import { View, Text, Progress, Button } from '@tarojs/components'
import { api } from '../../services/api'
import type { ComposeJob } from '../../types'
import './progress.scss'

const STAGE_LABEL: Record<string, string> = {
  queued: '排队中', claimed: '准备中', generating: '合成中',
  uploading: '上传中', done: '已完成',
}

export default function ProgressPage() {
  const router = useRouter()
  const jobId = Number(router.params.jobId)
  const [job, setJob] = useState<ComposeJob | null>(null)
  const timer = useRef<any>(null)
  const done = useRef(false)

  useEffect(() => {
    if (!jobId) return
    async function poll() {
      try {
        const j = await api.getJob(jobId)
        setJob(j)
        if (j.status === 'succeeded') {
          done.current = true
          clearInterval(timer.current)
          Taro.showToast({ title: '合成完成', icon: 'success' })
        } else if (j.status === 'failed') {
          done.current = true
          clearInterval(timer.current)
        }
      } catch (e: any) {
        // 网络抖动容忍
      }
    }
    poll()
    timer.current = setInterval(poll, 1500)
    return () => clearInterval(timer.current)
  }, [jobId])

  const isFailed = job?.status === 'failed'
  const isDone = job?.status === 'succeeded'

  return (
    <View className='page progress'>
      <View className='card progress-card'>
        <Text className='progress-title'>{isFailed ? '合成失败' : isDone ? '合成完成' : '正在生成获客视频'}</Text>
        {!isFailed && (
          <>
            <View className='progress-bar-wrap'>
              <Progress percent={job?.progress || 0} strokeWidth={10} activeColor='#6c5ce7' backgroundColor='#efeaff' />
            </View>
            <Text className='progress-pct'>{job?.progress || 0}%</Text>
            <Text className='muted'>当前阶段：{STAGE_LABEL[job?.stage || 'queued'] || job?.stage || '排队中'}</Text>
          </>
        )}
        {isFailed && <Text className='error-msg'>{job?.error_message || '未知错误'}</Text>}
      </View>

      {isDone && (
        <>
          <Button className='btn-primary' onClick={() => Taro.redirectTo({ url: '/pages/works/works' })}>查看作品</Button>
          {job?.result && (
            <Button className='btn-secondary' onClick={() => {
              Taro.setClipboardData({ data: job.result!.file_url })
              Taro.showToast({ title: '链接已复制', icon: 'none' })
            }}>复制下载链接</Button>
          )}
        </>
      )}
      {isFailed && (
        <Button className='btn-secondary' onClick={() => Taro.navigateBack()}>返回修改</Button>
      )}
    </View>
  )
}
