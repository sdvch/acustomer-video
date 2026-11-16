package ffmpeg

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// MediaInfo 素材探测结果（服务端 ffprobe，用于质检）
type MediaInfo struct {
	Kind       string // image / video
	Width      int
	Height     int
	DurationMs int64
	Codec      string
	FPS        float64
	BitrateBps int64
	HasAudio   bool
	SizeBytes  int64
}

type probeFormat struct {
	Duration string `json:"duration"`
	BitRate  string `json:"bit_rate"`
	Format   string `json:"format_name"`
	Size     string `json:"size"`
}

type probeStream struct {
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	CodecName  string `json:"codec_name"`
	CodecType  string `json:"codec_type"`
	RFrameRate string `json:"r_frame_rate"`
	AvgFPS     string `json:"avg_frame_rate"`
	Duration   string `json:"duration"`
	NbFrames   string `json:"nb_frames"`
}

// Inspect 用 ffprobe 读取素材真实元信息。
// 图片（mjpeg/png 等单帧）没有 duration，按 image 处理。
func (p *Preset) Inspect(path string) (*MediaInfo, error) {
	cmd := exec.Command(p.FFprobePath,
		"-v", "error",
		"-show_entries", "format=duration,bit_rate,format_name,size:stream=width,height,codec_name,codec_type,r_frame_rate,avg_frame_rate,duration,nb_frames",
		"-of", "json", path)
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("无法解析素材（ffprobe）: %w", err)
	}
	var raw struct {
		Format  probeFormat   `json:"format"`
		Streams []probeStream `json:"streams"`
	}
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("解析 ffprobe 输出失败: %w", err)
	}

	info := &MediaInfo{Kind: "video"}
	for _, s := range raw.Streams {
		switch s.CodecType {
		case "video":
			if info.Width == 0 {
				info.Width, info.Height = s.Width, s.Height
				info.Codec = s.CodecName
				info.FPS = parseFPS(s.RFrameRate, s.AvgFPS)
				if isStillImageFormat(raw.Format.Format) {
					info.Kind = "image"
				}
			}
		case "audio":
			info.HasAudio = true
		}
	}
	if info.Width == 0 {
		return nil, fmt.Errorf("素材中没有视频/图像流")
	}

	if d, err := strconv.ParseFloat(raw.Format.Duration, 64); err == nil && d > 0 {
		info.DurationMs = int64(d * 1000)
	}
	if b, err := strconv.ParseInt(raw.Format.BitRate, 10, 64); err == nil {
		info.BitrateBps = b
	}
	if sz, err := strconv.ParseInt(raw.Format.Size, 10, 64); err == nil {
		info.SizeBytes = sz
	}
	if info.Kind == "image" {
		info.DurationMs = 0
	}
	return info, nil
}

func isStillImageFormat(name string) bool {
	for _, f := range []string{"image2", "jpeg_pipe", "png_pipe", "webp_pipe", "bmp_pipe", "gif"} {
		if strings.Contains(name, f) {
			return true
		}
	}
	return false
}

func parseFPS(candidates ...string) float64 {
	for _, s := range candidates {
		if s == "" || s == "0/0" {
			continue
		}
		parts := strings.SplitN(s, "/", 2)
		num, err := strconv.ParseFloat(parts[0], 64)
		if err != nil || num == 0 {
			continue
		}
		den := 1.0
		if len(parts) == 2 {
			den, _ = strconv.ParseFloat(parts[1], 64)
		}
		if den == 0 {
			continue
		}
		return num / den
	}
	return 0
}
