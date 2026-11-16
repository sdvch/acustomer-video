package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// ProbeResult ffprobe 输出的结构（只取需要的字段）
type ProbeResult struct {
	Format struct {
		Duration string `json:"duration"`
		BitRate  string `json:"bit_rate"`
	} `json:"format"`
	Streams []struct {
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		CodecName  string `json:"codec_name"`
		Profile    string `json:"profile"`
		Level      int    `json:"level"`
		SampleRate string `json:"sample_rate"`
		CodecType  string `json:"codec_type"`
	} `json:"streams"`
}

// probe 读取成片元信息
func (p *Preset) probe(path string) (*ProbeResult, error) {
	cmd := exec.Command(p.FFprobePath,
		"-v", "error",
		"-show_entries", "format=duration,bit_rate:stream=width,height,r_frame_rate,codec_name,profile,level,sample_rate,codec_type",
		"-of", "json", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe 失败: %w", err)
	}
	var r ProbeResult
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}
	return &r, nil
}

// Validate 按 SOP 表10 校验成片规格；返回 (元信息, 校验错误列表)
func (p *Preset) Validate(path string) (*ProbeResult, []string) {
	r, err := p.probe(path)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var errs []string

	// 定位视频流与音频流索引
	vIdx, aIdx := -1, -1
	for i := range r.Streams {
		if r.Streams[i].CodecType == "video" && vIdx < 0 {
			vIdx = i
		} else if r.Streams[i].CodecType == "audio" && aIdx < 0 {
			aIdx = i
		}
	}

	if vIdx < 0 {
		errs = append(errs, "缺少视频流")
	} else {
		v := r.Streams[vIdx]
		if v.Width != p.Width || v.Height != p.Height {
			errs = append(errs, fmt.Sprintf("分辨率 %dx%d != %dx%d", v.Width, v.Height, p.Width, p.Height))
		}
		if !fpsMatch(v.RFrameRate, p.FPS) {
			errs = append(errs, fmt.Sprintf("帧率 %s != %d", v.RFrameRate, p.FPS))
		}
		if v.CodecName != "h264" {
			errs = append(errs, "编码不是 h264: "+v.CodecName)
		}
		if !strings.EqualFold(v.Profile, "High") {
			errs = append(errs, "profile 不是 High: "+v.Profile)
		}
		// level 4.1 -> ffprobe 返回 41
		if v.Level != 41 {
			errs = append(errs, fmt.Sprintf("level=%d != 41", v.Level))
		}
		// 码率 SOP 要求 8-15Mbps（CBR 编码后应稳定在 12Mbps 附近）
		if br, err := strconv.ParseInt(r.Format.BitRate, 10, 64); err == nil && br > 0 {
			if br < 7_500_000 || br > 15_500_000 {
				errs = append(errs, fmt.Sprintf("码率 %.1fMbps 超出 SOP 区间 8-15Mbps", float64(br)/1e6))
			}
		}
	}

	if aIdx < 0 {
		errs = append(errs, "缺少音频流")
	} else {
		a := r.Streams[aIdx]
		if a.CodecName != "aac" {
			errs = append(errs, "音频编码不是 aac: "+a.CodecName)
		}
		if a.SampleRate != "48000" {
			errs = append(errs, "音频采样率 != 48000: "+a.SampleRate)
		}
	}

	// 时长：按 SOP 25-35s 区间断言，允许 1.5s 容差（首尾帧对齐会造成毫秒级偏差）
	minSec := float64(p.DurationMinSec)
	maxSec := float64(p.DurationMaxSec)
	if minSec <= 0 {
		minSec = 25
	}
	if maxSec <= minSec {
		maxSec = 35
	}
	dur, _ := strconv.ParseFloat(r.Format.Duration, 64)
	if dur < minSec-1.5 || dur > maxSec+1.5 {
		errs = append(errs, fmt.Sprintf("时长 %.2fs 超出 SOP 区间 %.0f-%.0fs", dur, minSec, maxSec))
	}

	return r, errs
}

func fpsMatch(rFrameRate string, target int) bool {
	parts := strings.SplitN(rFrameRate, "/", 2)
	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return false
	}
	den := 1
	if len(parts) == 2 {
		den, _ = strconv.Atoi(parts[1])
	}
	if den == 0 {
		return false
	}
	return num/den == target
}
