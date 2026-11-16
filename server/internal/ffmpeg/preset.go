// Package ffmpeg 实现图片/视频素材拼接成竖屏成片的流水线。
// 成片规格对齐 SOP 表10：1080x1920@30fps、H.264 High@4.1、12Mbps、AAC 192k/48kHz、+faststart。
package ffmpeg

import (
	"fmt"
	"strings"
)

// Preset 合成规格（从 config 注入，便于测试）
type Preset struct {
	FFmpegPath  string
	FFprobePath string
	Width       int
	Height      int
	FPS         int
	Bitrate     string
	ClipSeconds int
	FontPath    string
	FitMode     string // pad / crop
	// 成片时长区间（SOP 25-35s），出口校验用，允许 1.5s 容差
	DurationMinSec int
	DurationMaxSec int
}

// DefaultPreset 默认 SOP 规格
func DefaultPreset(ffmpeg, ffprobe string) *Preset {
	return &Preset{
		FFmpegPath: ffmpeg, FFprobePath: ffprobe,
		Width: 1080, Height: 1920, FPS: 30, Bitrate: "12M",
		ClipSeconds: 4, FitMode: "pad",
		DurationMinSec: 25, DurationMaxSec: 35,
	}
}

// videoEncodeArgs 统一视频编码参数（concat 前各段必须完全一致）。
// 用 CBR（nal-hrd）+ minrate=maxrate 保证码率落在 SOP 要求的 8-15Mbps，
// 否则纯色/静态画面会被 x264 压到几 Mbps。
func (p *Preset) videoEncodeArgs() []string {
	return []string{
		"-c:v", "libx264",
		"-profile:v", "high",
		"-level:v", "4.1",
		"-preset", "medium",
		"-b:v", p.Bitrate,
		"-minrate", p.Bitrate,
		"-maxrate", p.Bitrate,
		"-bufsize", "24M",
		"-x264-params", "nal-hrd=cbr:force-cfr=1",
		"-g", fmt.Sprintf("%d", p.FPS*2),
		"-pix_fmt", "yuv420p",
		"-video_track_timescale", "30000",
	}
}

// audioEncodeArgs 统一音频编码参数
func (p *Preset) audioEncodeArgs() []string {
	return []string{
		"-c:a", "aac", "-b:a", "192k", "-ar", "48000", "-ac", "2",
	}
}

// scaleFilter 依 fit_mode 选择 pad（补黑边）或 crop（裁切填满）
func (p *Preset) scaleFilter() string {
	base := fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=decrease", p.Width, p.Height)
	if p.FitMode == "crop" {
		return base + fmt.Sprintf(",crop=%d:%d,setsar=1", p.Width, p.Height)
	}
	return base + fmt.Sprintf(",pad=%d:%d:(ow-iw)/2:(oh-ih)/2:black,setsar=1", p.Width, p.Height)
}

// buildImageSegmentArgs 图片 -> 视频片段（静帧 + 静音轨）；durMs>0 时用它，否则用 preset.ClipSeconds
func (p *Preset) buildImageSegmentArgs(input, output string, durMs int64) []string {
	durSec := float64(p.ClipSeconds)
	if durMs > 0 {
		durSec = float64(durMs) / 1000.0
	}
	vf := p.scaleFilter() + ",format=yuv420p"
	args := []string{
		"-y", "-loop", "1", "-framerate", fmt.Sprintf("%d", p.FPS), "-i", input,
		"-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo",
		"-vf", vf,
	}
	args = append(args, p.videoEncodeArgs()...)
	args = append(args, p.audioEncodeArgs()...)
	args = append(args, "-t", fmt.Sprintf("%.3f", durSec), "-shortest", "-movflags", "+faststart", output)
	return args
}

// buildVideoSegmentArgs 视频 -> 统一规格片段
func (p *Preset) buildVideoSegmentArgs(input, output string, trimMs int64) []string {
	vf := p.scaleFilter() + fmt.Sprintf(",fps=%d,format=yuv420p", p.FPS)
	args := []string{"-y", "-i", input}
	if trimMs > 0 {
		args = append(args, "-t", fmt.Sprintf("%.3f", float64(trimMs)/1000.0))
	}
	args = append(args,
		"-vf", vf,
		"-af", "aresample=48000",
	)
	args = append(args, p.videoEncodeArgs()...)
	args = append(args, p.audioEncodeArgs()...)
	args = append(args, "-movflags", "+faststart", output)
	return args
}

// escapeConcatPath concat demuxer 文件路径转义
func escapeConcatPath(p string) string {
	p = strings.ReplaceAll(p, "'", `'\''`)
	return p
}
