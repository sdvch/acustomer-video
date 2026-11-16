package service

import (
	"strings"
	"testing"

	"acustomer-video-server/internal/model"
)

func imgAsset(durMs int64) model.Asset {
	return model.Asset{Type: model.AssetImage, DurationMs: durMs}
}

func vidAsset(durMs int64) model.Asset {
	return model.Asset{Type: model.AssetVideo, DurationMs: durMs}
}

// 图片没有时长概念，一律取片段设定时长。
func TestSegmentDurationImage(t *testing.T) {
	if got := SegmentDuration(imgAsset(0), 5000); got != 5000 {
		t.Errorf("图片段应取 clipMs=5000，得到 %d", got)
	}
	// 图片即使带了 duration 字段也不该影响结果
	if got := SegmentDuration(imgAsset(99999), 5000); got != 5000 {
		t.Errorf("图片段不应受 duration 影响，得到 %d", got)
	}
}

// 视频取 min(素材时长, 片段时长)：素材更长要裁，更短就整段用。
func TestSegmentDurationVideo(t *testing.T) {
	cases := []struct {
		name   string
		asset  int64
		clipMs int64
		want   int64
	}{
		{"素材比片段长，裁剪", 12000, 5000, 5000},
		{"素材比片段短，整段使用", 3000, 5000, 3000},
		{"素材时长等于片段", 5000, 5000, 5000},
		{"素材时长未知，按片段时长", 0, 5000, 5000},
	}
	for _, c := range cases {
		if got := SegmentDuration(vidAsset(c.asset), c.clipMs); got != c.want {
			t.Errorf("%s: 得到 %d，期望 %d", c.name, got, c.want)
		}
	}
}

// handler 校验与 worker 实际裁剪共用 PlanSegments，两者口径必须一致。
func TestPlanSegmentsTotal(t *testing.T) {
	assets := []model.Asset{
		imgAsset(0),    // 4000
		vidAsset(2500), // 2500（素材短于片段）
		vidAsset(9000), // 4000（裁剪）
	}
	durs, total := PlanSegments(assets, 4000)

	want := []int64{4000, 2500, 4000}
	if len(durs) != len(want) {
		t.Fatalf("段数 = %d，期望 %d", len(durs), len(want))
	}
	for i := range want {
		if durs[i] != want[i] {
			t.Errorf("第 %d 段 = %d，期望 %d", i+1, durs[i], want[i])
		}
	}
	if total != 10500 {
		t.Errorf("总时长 = %d，期望 10500", total)
	}
}

func TestPlanSegmentsEmpty(t *testing.T) {
	durs, total := PlanSegments(nil, 4000)
	if len(durs) != 0 || total != 0 {
		t.Errorf("空素材应返回空片段与 0 时长，得到 len=%d total=%d", len(durs), total)
	}
}

// 合规区间：25-35s。
func TestCheckDurationOK(t *testing.T) {
	for _, ms := range []int64{25000, 30000, 35000} {
		v := CheckDuration(ms, 6, 25, 35, 3, 8)
		if !v.OK {
			t.Errorf("%dms 应判定合规，得到 message=%q", ms, v.Message)
		}
		if v.Message != "" {
			t.Errorf("合规时不应给建议文案，得到 %q", v.Message)
		}
	}
}

func TestCheckDurationTooShort(t *testing.T) {
	v := CheckDuration(10000, 3, 25, 35, 3, 8)
	if v.OK {
		t.Fatal("10s 应判定不合规")
	}
	if v.SuggestMs != 10000 { // (25000+35000)/2/3 = 10000
		t.Errorf("建议单段 = %d，期望 10000", v.SuggestMs)
	}
	if !strings.Contains(v.Message, "短于") {
		t.Errorf("应提示过短，得到 %q", v.Message)
	}
	// 提示里应给出「再补几段」（(25000-10000)/3000 → 5 段向上取整）
	if !strings.Contains(v.Message, "再补 5 段") {
		t.Errorf("应提示补 5 段素材，得到 %q", v.Message)
	}
}

func TestCheckDurationTooLong(t *testing.T) {
	v := CheckDuration(60000, 12, 25, 35, 3, 8)
	if v.OK {
		t.Fatal("60s 应判定不合规")
	}
	if !strings.Contains(v.Message, "超过") {
		t.Errorf("应提示过长，得到 %q", v.Message)
	}
	if v.SuggestMs != 2500 { // 30000/12
		t.Errorf("建议单段 = %d，期望 2500", v.SuggestMs)
	}
}

// 没有素材时不能除零，只给笼统提示。
func TestCheckDurationNoAssets(t *testing.T) {
	v := CheckDuration(0, 0, 25, 35, 3, 8)
	if v.OK {
		t.Fatal("0 素材应判定不合规")
	}
	if v.SuggestMs != 0 {
		t.Errorf("无素材时不应给建议时长，得到 %d", v.SuggestMs)
	}
	if v.Message == "" {
		t.Error("应给出提示文案")
	}
}

// 边界：区间端点必须闭合（25s 与 35s 都算合规）。
func TestCheckDurationBoundary(t *testing.T) {
	if !CheckDuration(25000, 6, 25, 35, 3, 8).OK {
		t.Error("25s 是合规下界，应通过")
	}
	if !CheckDuration(35000, 6, 25, 35, 3, 8).OK {
		t.Error("35s 是合规上界，应通过")
	}
	if CheckDuration(24999, 6, 25, 35, 3, 8).OK {
		t.Error("24999ms 低于下界，应不通过")
	}
	if CheckDuration(35001, 6, 25, 35, 3, 8).OK {
		t.Error("35001ms 高于上界，应不通过")
	}
}

func TestCeilDiv(t *testing.T) {
	cases := []struct{ a, b, want int64 }{
		{10, 3, 4}, {9, 3, 3}, {0, 3, 0}, {1, 1, 1}, {5, 0, 0},
	}
	for _, c := range cases {
		if got := ceilDiv(c.a, c.b); got != c.want {
			t.Errorf("ceilDiv(%d,%d) = %d，期望 %d", c.a, c.b, got, c.want)
		}
	}
}
