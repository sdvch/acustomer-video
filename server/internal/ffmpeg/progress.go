package ffmpeg

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Progress ffmpeg 进度快照
type Progress struct {
	OutTimeUS int64
	Frame     int64
	Done      bool
}

// parseProgress 逐行解析 -progress pipe:1 输出的 key=value
func parseProgress(line string, p *Progress) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	kv := strings.SplitN(line, "=", 2)
	if len(kv) != 2 {
		return
	}
	key, val := kv[0], strings.TrimSpace(kv[1])
	switch key {
	case "out_time_us":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			p.OutTimeUS = n
		}
	case "frame":
		if n, err := strconv.ParseInt(val, 10, 64); err == nil {
			p.Frame = n
		}
	case "progress":
		if val == "end" {
			p.Done = true
		}
	}
}

// readProgressLoop 在单独 goroutine 中读取 -progress 管道，回调最新进度。
// onUpdate(outTimeUS) 由上层换算为 0-100 百分比。
func readProgressLoop(r io.Reader, onUpdate func(outTimeUS int64)) {
	scanner := bufio.NewScanner(r)
	p := &Progress{}
	for scanner.Scan() {
		parseProgress(scanner.Text(), p)
		if p.OutTimeUS > 0 && onUpdate != nil {
			onUpdate(p.OutTimeUS)
		}
		if p.Done {
			break
		}
	}
}

// pctFromTime 按已完成时间/总目标时间换算阶段内百分比（0-100）
func pctFromTime(outTimeUS, totalUS int64) int {
	if totalUS <= 0 {
		return 0
	}
	pct := int(float64(outTimeUS) / float64(totalUS) * 100)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	return pct
}

// formatProgressErr 便于调试
func formatProgressErr(p *Progress) string {
	return fmt.Sprintf("frame=%d out_time_us=%d done=%v", p.Frame, p.OutTimeUS, p.Done)
}
