// composecli 是 ffmpeg 流水线的命令行验证工具，便于脱离 API 单独调试合成效果。
// 用法：go run ./cmd/composecli -out /tmp/out.mp4 img1.jpg vid1.mp4 img2.jpg
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"acustomer-video-server/internal/ffmpeg"
)

func main() {
	out := flag.String("out", "", "输出文件路径（默认写入输入目录 final.mp4）")
	clip := flag.Int("clip", 4, "图片静帧时长（秒）")
	fit := flag.String("fit", "pad", "适配模式 pad/crop")
	flag.Parse()
	inputs := flag.Args()
	if len(inputs) == 0 {
		fmt.Println("用法: composecli [-out path] [-clip 4] [-fit pad] <素材1> <素材2> ...")
		os.Exit(1)
	}

	workDir, err := os.MkdirTemp("", "composecli-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(workDir)

	preset := ffmpeg.DefaultPreset("ffmpeg", "ffprobe")
	preset.ClipSeconds = *clip
	preset.FitMode = *fit
	pl := ffmpeg.NewPipeline(preset, 2)

	var segs []ffmpeg.Segment
	for i, in := range inputs {
		// 图片按扩展名判断，其余当视频
		typ := 2
		ext := filepath.Ext(in)
		if ext == ".jpg" || ext == ".jpeg" || ext == ".png" || ext == ".webp" {
			typ = 1
		}
		segs = append(segs, ffmpeg.Segment{Type: typ, LocalPath: in})
		_ = i
	}

	res, err := pl.Run(context.Background(), workDir, segs, func(stage string, pct int) {
		fmt.Printf("[%s] %d%%\n", stage, pct)
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "合成失败:", err)
		os.Exit(1)
	}

	output := *out
	if output == "" {
		output = filepath.Join(filepath.Dir(inputs[0]), "final.mp4")
	}
	if err := copyFile(res.OutputPath, output); err != nil {
		panic(err)
	}
	fmt.Printf("成功: %s duration=%dms\n", output, res.DurationMs)
	fmt.Printf("probe: %+v\n", res.Probe)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	buf := make([]byte, 1024*1024)
	for {
		n, e := in.Read(buf)
		if n > 0 {
			if _, we := out.Write(buf[:n]); we != nil {
				return we
			}
		}
		if e != nil {
			break
		}
	}
	return nil
}
