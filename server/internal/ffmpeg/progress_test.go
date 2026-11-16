package ffmpeg

import (
	"strings"
	"testing"
)

func TestParseProgressFields(t *testing.T) {
	p := &Progress{}
	parseProgress("frame=120", p)
	parseProgress("out_time_us=1500000", p)
	parseProgress("bitrate=12000.1kbits/s", p) // 忽略未知 key
	parseProgress("progress=continue", p)
	parseProgress("progress=end", p)
	if p.Frame != 120 {
		t.Errorf("Frame = %d, want 120", p.Frame)
	}
	if p.OutTimeUS != 1500000 {
		t.Errorf("OutTimeUS = %d, want 1500000", p.OutTimeUS)
	}
	if !p.Done {
		t.Error("progress=end 应置 Done")
	}
}

func TestParseProgressMalformed(t *testing.T) {
	p := &Progress{OutTimeUS: 99, Frame: 9}
	for _, line := range []string{"", "   ", "no-equals-sign", "=", "out_time_us=abc", "out_time_us=", "frame=1.5"} {
		parseProgress(line, p)
	}
	if p.OutTimeUS != 99 || p.Frame != 9 {
		t.Errorf("畸形行不应覆盖已有值: %+v", p)
	}
	if p.Done {
		t.Error("畸形行不应置 Done")
	}
}

func TestReadProgressLoop(t *testing.T) {
	in := strings.NewReader("frame=10\nout_time_us=1000000\nframe=20\nout_time_us=2000000\nprogress=end\nframe=999\n")
	var got []int64
	readProgressLoop(in, func(us int64) { got = append(got, us) })
	if len(got) == 0 || got[len(got)-1] != 2000000 {
		t.Errorf("应回调最新 out_time_us，实际 %v", got)
	}
	// progress=end 后应停止读取（frame=999 不应再触发回调）
	for _, v := range got {
		if v > 2000000 {
			t.Errorf("end 之后不应继续回调，实际出现 %d", v)
		}
	}
}

func TestReadProgressLoopEmpty(t *testing.T) {
	called := false
	readProgressLoop(strings.NewReader(""), func(int64) { called = true })
	if called {
		t.Error("空输入不应触发回调")
	}
}
