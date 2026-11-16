// Package worker 实现异步合成任务消费：claim → 下载素材 → ffmpeg 合成 → 上传成片 → complete。
package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/ffmpeg"
	"acustomer-video-server/internal/model"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service"
)

// Worker 合成任务消费者
type Worker struct {
	cfg      *config.Config
	oss      *service.OSSService
	pipeline *ffmpeg.Pipeline
	stop     chan struct{}
}

func New(cfg *config.Config, oss *service.OSSService) *Worker {
	// 确保工作目录父级存在（MkdirTemp 要求父目录已存在）
	_ = os.MkdirAll(cfg.WorkDir, 0755)
	_ = os.MkdirAll(cfg.StorageDir, 0755)
	preset := ffmpeg.DefaultPreset(cfg.FFmpegPath, cfg.FFprobePath)
	preset.Width, preset.Height = cfg.TargetWidth, cfg.TargetHeight
	preset.FPS = cfg.TargetFPS
	preset.Bitrate = cfg.TargetBitrate
	preset.ClipSeconds = cfg.DefaultClipSeconds
	preset.DurationMinSec = cfg.MinDurationSec
	preset.DurationMaxSec = cfg.MaxDurationSec
	return &Worker{
		cfg: cfg, oss: oss,
		pipeline: ffmpeg.NewPipeline(preset, cfg.WorkerCount),
		stop:     make(chan struct{}),
	}
}

// Start 启动 N 个消费 goroutine + 看门狗
func (w *Worker) Start() {
	n := w.cfg.WorkerCount
	for i := 0; i < n; i++ {
		go w.loop(i)
	}
	go w.watchdog()
	log.Printf("[worker] 启动 %d 个消费者，租约 %ds", n, w.cfg.JobLeaseSeconds)
}

func (w *Worker) Stop() { close(w.stop) }

func (w *Worker) loop(id int) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			job, err := repository.ClaimJob()
			if err != nil {
				log.Printf("[worker-%d] claim 失败: %v", id, err)
				continue
			}
			if job == nil {
				continue
			}
			w.process(job)
		}
	}
}

func (w *Worker) process(job *model.ComposeJob) {
	token := job.ClaimToken
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(w.cfg.ComposeTimeoutMin)*time.Minute)
	defer cancel()

	workDir, err := os.MkdirTemp(w.cfg.WorkDir, fmt.Sprintf("job-%d-*", job.ID))
	if err != nil {
		_ = repository.FailJob(job.ID, token, "创建工作目录失败: "+err.Error())
		return
	}
	defer os.RemoveAll(workDir)

	if err := repository.MarkJobGenerating(job.ID, token); err != nil {
		log.Printf("[worker] job=%d 标记 generating 失败: %v", job.ID, err)
		return
	}

	// 解析输入
	var input model.JobInput
	if err := json.Unmarshal([]byte(job.InputJSON), &input); err != nil {
		_ = repository.FailJob(job.ID, token, "解析合成输入失败: "+err.Error())
		return
	}

	// 下载素材 + 生成 Segment
	assets, err := repository.GetAssetsByIDs(job.UserID, input.AssetIDs)
	if err != nil || len(assets) == 0 {
		_ = repository.FailJob(job.ID, token, "素材不存在或不可用")
		return
	}
	var segs []ffmpeg.Segment
	// 逐段时长优先用发起合成时算好的快照，保证与校验结果一致
	clipMs := int64(input.ClipSeconds) * 1000
	if clipMs <= 0 {
		clipMs = int64(w.cfg.DefaultClipSeconds) * 1000
	}
	planMs, _ := service.PlanSegments(assets, clipMs)
	for i, a := range assets {
		if !a.Authorized {
			_ = repository.FailJob(job.ID, token, fmt.Sprintf("素材 %d 未勾选授权", a.ID))
			return
		}
		local := filepath.Join(workDir, fmt.Sprintf("src_%03d%s", i, extFromMime(a.Mime, a.ObjectKey)))
		if err := w.oss.DownloadToLocal(a.ObjectKey, local); err != nil {
			_ = repository.FailJob(job.ID, token, fmt.Sprintf("下载素材 %d 失败: %v", a.ID, err))
			return
		}
		segMs := clipMs
		if i < len(planMs) {
			segMs = planMs[i]
		}
		if len(input.SegmentMs) == len(assets) {
			segMs = input.SegmentMs[i]
		}
		segs = append(segs, ffmpeg.Segment{Type: a.Type, LocalPath: local, DurMs: segMs})
	}

	// 合成（进度回调限制写库频率）
	lastPct := -1
	res, err := w.pipeline.Run(ctx, workDir, segs, func(stage string, pct int) {
		// pipeline 已给出全局 0-90；上传阶段补到 100。限频写库（每 2%）
		if pct != lastPct && pct%2 == 0 {
			lastPct = pct
			_ = repository.UpdateJobProgress(job.ID, token, stage, clamp(pct))
		}
	})
	if err != nil {
		_ = repository.FailJob(job.ID, token, "合成失败: "+err.Error())
		return
	}

	// 上传成片
	_ = repository.MarkJobUploading(job.ID, token)
	objectKey := w.oss.BuildObjectKey(job.UserID, "final.mp4")
	if err := w.oss.UploadLocalFile(res.OutputPath, objectKey); err != nil {
		_ = repository.FailJob(job.ID, token, "上传成片失败: "+err.Error())
		return
	}
	_ = repository.UpdateJobProgress(job.ID, token, "uploading", 98)

	// 落库成片记录
	bitrate, _ := strconv.ParseInt(res.Probe.Format.BitRate, 10, 64)
	v := &model.Video{
		ProjectID: job.ProjectID, JobID: job.ID, ObjectKey: objectKey,
		FileURL: w.oss.PublicURL(objectKey), DurationMs: res.DurationMs,
		Width: w.cfg.TargetWidth, Height: w.cfg.TargetHeight, FPS: w.cfg.TargetFPS,
		BitrateBps: bitrate, Codec: "h264",
	}
	videoID, err := repository.InsertVideo(v)
	if err != nil {
		_ = repository.FailJob(job.ID, token, "保存成片记录失败: "+err.Error())
		return
	}
	if err := repository.CompleteJob(job.ID, token, videoID); err != nil {
		log.Printf("[worker] job=%d complete 失败: %v", job.ID, err)
		return
	}
	log.Printf("[worker] job=%d 完成 video=%d", job.ID, videoID)
}

func (w *Worker) watchdog() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-w.stop:
			return
		case <-ticker.C:
			n, err := repository.RecoverStaleJobs(w.cfg.JobLeaseSeconds)
			if err != nil {
				log.Printf("[watchdog] 恢复失败: %v", err)
			} else if n > 0 {
				log.Printf("[watchdog] 重置 %d 个超时任务", n)
			}
		}
	}
}

func clamp(p int) int {
	if p < 0 {
		return 0
	}
	if p > 100 {
		return 100
	}
	return p
}

func extFromMime(mime, key string) string {
	ext := filepath.Ext(key)
	if ext != "" {
		return ext
	}
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	default:
		return ".mp4"
	}
}
