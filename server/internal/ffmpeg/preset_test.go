package ffmpeg

import (
	"slices"
	"strings"
	"testing"
)

func newTestPreset() *Preset {
	return DefaultPreset("ffmpeg", "ffprobe")
}

// 成片必须满足 SOP 表10：1080x1920@30fps、H.264 High@4.1、CBR 12Mbps。
func TestVideoEncodeArgsMatchesSOP(t *testing.T) {
	p := newTestPreset()
	args := p.videoEncodeArgs()
	joined := strings.Join(args, " ")

	mustContain := []string{
		"-c:v libx264",
		"-profile:v high",
		"-level:v 4.1",
		"-b:v 12M",
		"-minrate 12M", // CBR 必须有 minrate，否则纯色画面会被压到几 Mbps
		"-maxrate 12M",
		"-x264-params nal-hrd=cbr:force-cfr=1", // 真 CBR + 恒定帧率
		"-g 60",                                // 2 秒一个关键帧（30fps）
		"-pix_fmt yuv420p",
		"-video_track_timescale 30000",
	}
	for _, want := range mustContain {
		if !strings.Contains(joined, want) {
			t.Errorf("视频编码参数缺少 %q\n实际: %s", want, joined)
		}
	}
}

// 音频固定 AAC 192k / 48kHz / 双声道，否则 concat 时音轨不一致会失败。
func TestAudioEncodeArgs(t *testing.T) {
	joined := strings.Join(newTestPreset().audioEncodeArgs(), " ")
	for _, want := range []string{"-c:a aac", "-b:a 192k", "-ar 48000", "-ac 2"} {
		if !strings.Contains(joined, want) {
			t.Errorf("音频编码参数缺少 %q，实际: %s", want, joined)
		}
	}
}

func TestScaleFilterFitMode(t *testing.T) {
	pad := newTestPreset() // 默认 pad
	if got := pad.scaleFilter(); !strings.Contains(got, "pad=1080:1920") {
		t.Errorf("pad 模式应补黑边，实际: %s", got)
	}

	crop := newTestPreset()
	crop.FitMode = "crop"
	got := crop.scaleFilter()
	if !strings.Contains(got, "crop=1080:1920") {
		t.Errorf("crop 模式应裁切填满，实际: %s", got)
	}
	if !strings.HasSuffix(got, "setsar=1") {
		t.Errorf("SAR 必须归一到 1，实际: %s", got)
	}
}

// 图片段时长：durMs 优先，缺省时回落到 preset.ClipSeconds。
func TestBuildImageSegmentDuration(t *testing.T) {
	p := newTestPreset()

	withMs := strings.Join(p.buildImageSegmentArgs("/tmp/a.jpg", "/tmp/a.mp4", 6500), " ")
	if !strings.Contains(withMs, "-t 6.500") {
		t.Errorf("应按 durMs=6500 输出 -t 6.500，实际: %s", withMs)
	}

	noMs := strings.Join(p.buildImageSegmentArgs("/tmp/a.jpg", "/tmp/a.mp4", 0), " ")
	if !strings.Contains(noMs, "-t 4.000") {
		t.Errorf("durMs=0 时应回落 4 秒，实际: %s", noMs)
	}

	if !strings.Contains(withMs, "-loop 1") {
		t.Error("图片应使用 -loop 1 生成静帧视频")
	}
	if !strings.Contains(withMs, "anullsrc") {
		t.Error("图片无音轨，必须补 anullsrc 静音轨才能 concat")
	}
	if !strings.Contains(withMs, "+faststart") {
		t.Error("成片必须 +faststart，保证微信端秒开")
	}
}

func TestBuildVideoSegmentTrim(t *testing.T) {
	p := newTestPreset()

	trimmed := strings.Join(p.buildVideoSegmentArgs("/tmp/v.mp4", "/tmp/o.mp4", 3200), " ")
	if !strings.Contains(trimmed, "-t 3.200") {
		t.Errorf("应按 trimMs 截取，实际: %s", trimmed)
	}

	whole := strings.Join(p.buildVideoSegmentArgs("/tmp/v.mp4", "/tmp/o.mp4", 0), " ")
	if strings.Contains(whole, "-t ") {
		t.Errorf("trimMs=0 表示整段使用，不应出现 -t，实际: %s", whole)
	}

	// 视频段也要统一 fps，否则 concat 后时长会漂移
	if !strings.Contains(trimmed, "fps=30") {
		t.Errorf("视频段需统一帧率，实际: %s", trimmed)
	}
}

func TestEscapeConcatPath(t *testing.T) {
	if got := escapeConcatPath("/tmp/plain.mp4"); got != "/tmp/plain.mp4" {
		t.Errorf("普通路径不应被改写，得到 %q", got)
	}
	// concat demuxer 用单引号包裹路径，路径内的单引号必须转义
	if got, want := escapeConcatPath("/tmp/it's.mp4"), `/tmp/it'\''s.mp4`; got != want {
		t.Errorf("单引号转义错误，得到 %q，期望 %q", got, want)
	}
}

// DefaultPreset 是 SOP 规格的唯一来源，任何字段被改坏都应在此暴露。
func TestDefaultPresetValues(t *testing.T) {
	p := DefaultPreset("ffmpeg", "ffprobe")
	checks := []struct {
		name string
		got  int
		want int
	}{
		{"Width", p.Width, 1080},
		{"Height", p.Height, 1920},
		{"FPS", p.FPS, 30},
		{"DurationMinSec", p.DurationMinSec, 25},
		{"DurationMaxSec", p.DurationMaxSec, 35},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %d，期望 %d", c.name, c.got, c.want)
		}
	}
	if p.Bitrate != "12M" {
		t.Errorf("Bitrate = %s，期望 12M", p.Bitrate)
	}
	if !slices.Contains([]string{"pad", "crop"}, p.FitMode) {
		t.Errorf("FitMode = %s，只允许 pad/crop", p.FitMode)
	}
}
