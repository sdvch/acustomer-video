import { useMemo, useState } from 'react'
import Taro from '@tarojs/taro'
import { View, Text, Button, Input } from '@tarojs/components'
import { api } from '../../services/api'
import { useStore } from '../../store'
import type { CopyResult } from '../../types'
import './editor.scss'

const MIN_SEC = 25
const MAX_SEC = 35

export default function Editor() {
  const { assets, reorderAssets, removeAsset } = useStore()
  const [title, setTitle] = useState('我的获客视频')
  const [clipSeconds, setClipSeconds] = useState(4)
  const [submitting, setSubmitting] = useState(false)
  const [copyLoading, setCopyLoading] = useState(false)
  const [copy, setCopy] = useState<CopyResult | null>(null)

  function move(index: number, delta: number) {
    const next = [...assets]
    const target = index + delta
    if (target < 0 || target >= next.length) return
    ;[next[index], next[target]] = [next[target], next[index]]
    reorderAssets(next.map((a, i) => ({ ...a, sort_order: i })))
  }

  // 与后端 PlanSegments 同口径：图片=片段时长，视频=min(素材时长, 片段时长)
  const totalMs = useMemo(() => {
    const clipMs = clipSeconds * 1000
    return assets.reduce((sum, a) => {
      if (a.type === 1) return sum + clipMs
      if (!a.duration_ms) return sum + clipMs
      return sum + Math.min(a.duration_ms, clipMs)
    }, 0)
  }, [assets, clipSeconds])

  const totalSec = totalMs / 1000
  const durationOK = totalSec >= MIN_SEC && totalSec <= MAX_SEC
  const tooShort = totalSec < MIN_SEC
  const suggestClip = assets.length > 0 ? Math.round(30 / assets.length) : 4
  const barPct = Math.min(100, (totalSec / 40) * 100)
  const zoneLeft = (MIN_SEC / 40) * 100
  const zoneWidth = ((MAX_SEC - MIN_SEC) / 40) * 100

  async function handleCompose() {
    if (assets.length === 0) {
      Taro.showToast({ title: '请先上传素材', icon: 'none' })
      return
    }
    if (!assets.every((a) => a.authorized)) {
      Taro.showToast({ title: '请勾选所有素材授权', icon: 'none' })
      return
    }
    if (!durationOK) {
      Taro.showToast({ title: `成片需 ${MIN_SEC}-${MAX_SEC} 秒，当前 ${totalSec.toFixed(1)} 秒`, icon: 'none' })
      return
    }
    setSubmitting(true)
    try {
      const proj = await api.createProject(title)
      const ids = assets.map((a) => a.id)
      await api.sortAssets(ids)
      const job = await api.compose(proj.project_id, ids, clipSeconds)
      Taro.redirectTo({ url: `/pages/progress/progress?jobId=${job.job_id}&projectId=${proj.project_id}` })
    } catch (e: any) {
      Taro.showToast({ title: e.message || '发起合成失败', icon: 'none' })
      setSubmitting(false)
    }
  }

  async function handleAICopy() {
    setCopyLoading(true)
    try {
      const res = await api.generateCopy({ title, asset_count: assets.length, industry: '实体门店' })
      setCopy(res)
    } catch (e: any) {
      Taro.showToast({ title: e.message || '文案生成失败', icon: 'none' })
    } finally {
      setCopyLoading(false)
    }
  }

  function copyText(text: string) {
    Taro.setClipboardData({ data: text })
  }

  return (
    <View className='page editor'>
      <View className='card'>
        <Text className='field-label'>作品名称</Text>
        <Input className='title-input' value={title} onInput={(e) => setTitle(e.detail.value)} />
        <Text className='field-label'>每段时长（秒，建议 2-8）</Text>
        <Input
          className='title-input'
          type='number'
          value={String(clipSeconds)}
          onInput={(e) => {
            const v = Number(e.detail.value) || 4
            setClipSeconds(Math.max(2, Math.min(8, v)))
          }}
        />
      </View>

      {/* 时长治理：SOP 要求成片 25-35s */}
      <View className='card duration-card'>
        <View className='duration-head'>
          <Text className='field-label no-margin'>预计成片 {totalSec.toFixed(1)} 秒</Text>
          <Text className={durationOK ? 'tag ok' : 'tag bad'}>
            {durationOK ? '符合 25-35s' : tooShort ? '偏短' : '偏长'}
          </Text>
        </View>
        <View className='bar'>
          <View className='bar-zone' style={{ left: `${zoneLeft}%`, width: `${zoneWidth}%` }} />
          <View className={durationOK ? 'bar-fill ok' : 'bar-fill bad'} style={{ width: `${barPct}%` }} />
        </View>
        {!durationOK && (
          <Text className='hint'>
            {tooShort
              ? `还差 ${(MIN_SEC - totalSec).toFixed(1)} 秒：可把每段调到约 ${suggestClip} 秒，或再补素材`
              : `超出 ${(totalSec - MAX_SEC).toFixed(1)} 秒：可把每段调到约 ${suggestClip} 秒，或减少素材段数`}
          </Text>
        )}
      </View>

      <View className='section-title'>
        素材顺序（共 {assets.length} 段，可上下调整）
        {assets.some((a) => a.qc_json && a.qc_json !== '[]') && (
          <Text className='muted'> · 部分素材有质检提示</Text>
        )}
      </View>
      {assets.map((a, i) => (
        <View className='card asset-row' key={a.id}>
          <View className='asset-thumb'>
            <Text className='asset-type'>{a.type === 1 ? '图' : '视'}</Text>
          </View>
          <View className='asset-info'>
            <Text className='asset-name'>镜头 {i + 1} · {a.type === 1 ? '图片' : '视频'}</Text>
            <Text className='muted'>
              {a.width}×{a.height}
              {a.type === 2 && a.duration_ms > 0 ? ` · ${(a.duration_ms / 1000).toFixed(1)}s` : ''}
              {a.qc_status === 'warning' ? ' · 有提示' : ''}
            </Text>
          </View>
          <View className='asset-ops'>
            <Text className='op-btn' onClick={() => move(i, -1)}>↑</Text>
            <Text className='op-btn' onClick={() => move(i, 1)}>↓</Text>
            <Text className='op-btn del' onClick={() => removeAsset(a.id)}>×</Text>
          </View>
        </View>
      ))}

      <Button className='btn-secondary' onClick={handleAICopy} disabled={copyLoading}>
        {copyLoading ? 'AI 生成中...' : 'AI 生成获客文案（DeepSeek）'}
      </Button>

      {copy && (
        <View className='card copy-card'>
          {copy.hooks?.map((h, i) => (
            <View className='hook-item' key={i} onClick={() => copyText(h.text)}>
              <Text className='hook-angle'>{h.angle}</Text>
              <Text className='hook-text'>{h.text}</Text>
            </View>
          ))}
          {copy.body && <Text className='copy-body'>{copy.body}</Text>}
          {copy.cta && <Text className='copy-cta'>结尾：{copy.cta}</Text>}
          <Text className='muted'>点任意钩子可复制</Text>
        </View>
      )}

      <Button className='btn-primary compose-btn' onClick={handleCompose} disabled={submitting || !durationOK}>
        {submitting ? '提交中...' : durationOK ? `开始合成（${assets.length} 段）` : '成片时长不达标'}
      </Button>
    </View>
  )
}
