package ffmpeg

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Segment 输入素材片段描述
type Segment struct {
	Type      int    // 1 图片 2 视频
	LocalPath string // 已下载到本地的文件路径
	DurMs     int64  // 片段时长（毫秒）：图片=静帧时长；视频=要裁切的时长；0 表示用默认值/整段
}

// ProgressCB 进度回调：stage 阶段名, pct 0-100
type ProgressCB func(stage string, pct int)

// Result 流水线产出
type Result struct {
	OutputPath string
	DurationMs int64
	Probe      *ProbeResult
}

// Pipeline ffmpeg 合成流水线
type Pipeline struct {
	preset *Preset
	codec  chan struct{} // 信号量，限制并发 ffmpeg 数
}

func NewPipeline(preset *Preset, maxConcurrent int) *Pipeline {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	return &Pipeline{preset: preset, codec: make(chan struct{}, maxConcurrent)}
}

// Run 执行完整流水线：normalize -> concat -> validate。
// workDir 由调用方创建并在结束后清理。
func (pl *Pipeline) Run(ctx context.Context, workDir string, segs []Segment, cb ProgressCB) (*Result, error) {
	if len(segs) == 0 {
		return nil, fmt.Errorf("没有可合成的素材")
	}

	// ---- 阶段1：逐段 normalize（进度 0-70）----
	segFiles := make([]string, 0, len(segs))
	for i, seg := range segs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		out := filepath.Join(workDir, fmt.Sprintf("seg_%03d.mp4", i))
		if _, err := os.Stat(seg.LocalPath); err != nil {
			return nil, fmt.Errorf("素材不存在: %s: %w", seg.LocalPath, err)
		}
		var args []string
		if seg.Type == 1 {
			args = pl.preset.buildImageSegmentArgs(seg.LocalPath, out, seg.DurMs)
		} else {
			args = pl.preset.buildVideoSegmentArgs(seg.LocalPath, out, seg.DurMs)
		}
		// 每段内部进度映射到 0-70
		segBase := i * 70 / len(segs)
		segSpan := 70 / len(segs)
		segTotalUS := int64(pl.preset.ClipSeconds) * 1e6
		if seg.DurMs > 0 {
			segTotalUS = seg.DurMs * 1000
		}
		runFFmpeg(ctx, pl, pl.preset.FFmpegPath, args, func(us int64) {
			if cb != nil {
				cb("transcode", segBase+pctFromTime(us, segTotalUS)*segSpan/100)
			}
		})
		if _, err := os.Stat(out); err != nil {
			return nil, fmt.Errorf("片段 %d 转码失败: %w", i, err)
		}
		segFiles = append(segFiles, out)
	}
	if cb != nil {
		cb("transcode", 70)
	}

	// ---- 阶段2：concat 拼接（进度 70-85）----
	if cb != nil {
		cb("concat", 72)
	}
	finalPath := filepath.Join(workDir, "final.mp4")
	if err := pl.concat(segFiles, finalPath, ctx); err != nil {
		return nil, err
	}
	if cb != nil {
		cb("concat", 85)
	}

	// ---- 阶段3：出口校验（SOP 表10）----
	if cb != nil {
		cb("validate", 87)
	}
	probe, errs := pl.preset.Validate(finalPath)
	if len(errs) > 0 {
		return nil, fmt.Errorf("成片规格校验不通过: %s", strings.Join(errs, "; "))
	}
	dur, _ := time.ParseDuration(strings.TrimSpace(probe.Format.Duration) + "s")
	if cb != nil {
		cb("validate", 90)
	}

	return &Result{OutputPath: finalPath, DurationMs: dur.Milliseconds(), Probe: probe}, nil
}

// concat 使用 concat demuxer 无损拼接（各段规格已统一）
func (pl *Pipeline) concat(segFiles []string, output string, ctx context.Context) error {
	listPath := filepath.Join(filepath.Dir(output), "inputs.txt")
	var b strings.Builder
	for _, f := range segFiles {
		abs, _ := filepath.Abs(f)
		b.WriteString(fmt.Sprintf("file '%s'\n", escapeConcatPath(abs)))
	}
	if err := os.WriteFile(listPath, []byte(b.String()), 0644); err != nil {
		return err
	}
	args := []string{
		"-y", "-f", "concat", "-safe", "0", "-i", listPath,
		"-c", "copy", "-movflags", "+faststart", output,
	}
	runFFmpeg(ctx, pl, pl.preset.FFmpegPath, args, nil)
	if _, err := os.Stat(output); err != nil {
		return fmt.Errorf("concat 拼接失败: %w", err)
	}
	return nil
}

// runFFmpeg 执行单条 ffmpeg 命令，受信号量控制，解析 -progress 进度。
func runFFmpeg(ctx context.Context, pl *Pipeline, bin string, args []string, onProgress func(int64)) error {
	pl.codec <- struct{}{}
	defer func() { <-pl.codec }()

	fullArgs := append([]string{}, args...)
	// 在 -y 之后插入进度参数
	if onProgress != nil {
		fullArgs = append([]string{"-progress", "pipe:1", "-nostats"}, fullArgs...)
	}
	cmd := exec.CommandContext(ctx, bin, fullArgs...)
	var stderr strings.Builder
	cmd.Stderr = &stderr

	var stdoutReader *os.File
	if onProgress != nil {
		pr, pw, err := os.Pipe()
		if err != nil {
			return err
		}
		cmd.Stdout = pw
		stdoutReader = pr
	}

	if err := cmd.Start(); err != nil {
		return err
	}
	if stdoutReader != nil {
		go func() {
			readProgressLoop(stdoutReader, onProgress)
			_ = stdoutReader.Close()
		}()
	}
	err := cmd.Wait()
	if stdoutReader != nil {
		_ = stdoutReader.Close()
	}
	if err != nil {
		return fmt.Errorf("ffmpeg 失败: %w | stderr: %s", err, tail(stderr.String(), 800))
	}
	return nil
}

func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
