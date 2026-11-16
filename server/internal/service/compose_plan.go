package service

import (
	"fmt"

	"acustomer-video-server/internal/model"
)

// SegmentDuration 单个素材在成片中的时长（毫秒）。
//   - 图片：静帧时长 = clipMs
//   - 视频：截取前 min(素材时长, clipMs)，素材本身更短就整段用
func SegmentDuration(a model.Asset, clipMs int64) int64 {
	if a.Type == model.AssetImage {
		return clipMs
	}
	if a.DurationMs <= 0 {
		return clipMs
	}
	if a.DurationMs < clipMs {
		return a.DurationMs
	}
	return clipMs
}

// PlanSegments 计算各段时长与预计总时长，handler（校验）与 worker（实际裁剪）共用，
// 保证"校验时算的时长"与"真正合成的时长"一致。
func PlanSegments(assets []model.Asset, clipMs int64) ([]int64, int64) {
	durs := make([]int64, 0, len(assets))
	var total int64
	for _, a := range assets {
		d := SegmentDuration(a, clipMs)
		durs = append(durs, d)
		total += d
	}
	return durs, total
}

// DurationVerdict 时长是否合规 + 给用户的可执行建议
type DurationVerdict struct {
	TotalMs   int64
	MinMs     int64
	MaxMs     int64
	OK        bool
	Message   string
	SuggestMs int64 // 建议的单段时长（毫秒），仅在不合规时给出
}

// CheckDuration 按 SOP 校验成片总时长是否落在 [minSec, maxSec]
func CheckDuration(totalMs int64, assetCount int, minSec, maxSec, minClipSec, maxClipSec int) DurationVerdict {
	v := DurationVerdict{
		TotalMs: totalMs,
		MinMs:   int64(minSec) * 1000,
		MaxMs:   int64(maxSec) * 1000,
	}
	if totalMs >= v.MinMs && totalMs <= v.MaxMs {
		v.OK = true
		return v
	}
	if assetCount <= 0 {
		v.Message = fmt.Sprintf("素材数量不足，成片需 %d-%d 秒", minSec, maxSec)
		return v
	}
	// 建议值：按目标中位时长均分到每段，并夹到单段允许区间
	mid := (v.MinMs + v.MaxMs) / 2
	suggest := mid / int64(assetCount)
	v.SuggestMs = suggest
	suggestSec := float64(suggest) / 1000.0
	if totalMs < v.MinMs {
		v.Message = fmt.Sprintf("预计成片 %.1f 秒，短于 SOP 要求的 %d 秒：建议每段调到 %.1f 秒（当前共 %d 段），或再补 %d 段素材",
			float64(totalMs)/1000, minSec, suggestSec, assetCount, ceilDiv(v.MinMs-totalMs, int64(minClipSec)*1000))
	} else {
		v.Message = fmt.Sprintf("预计成片 %.1f 秒，超过 SOP 要求的 %d 秒：建议每段调到 %.1f 秒，或减少 %d 段素材",
			float64(totalMs)/1000, maxSec, suggestSec, totalMs/v.MaxMs)
	}
	return v
}

func ceilDiv(a, b int64) int64 {
	if b <= 0 {
		return 0
	}
	return (a + b - 1) / b
}
