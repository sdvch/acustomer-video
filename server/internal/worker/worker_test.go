package worker

import "testing"

// clamp 是进度百分比上屏前的最后防线，必须夹在 [0,100]。
func TestClamp(t *testing.T) {
	cases := map[int]int{-5: 0, 0: 0, 1: 1, 50: 50, 100: 100, 101: 100, 1000000: 100}
	for in, want := range cases {
		if got := clamp(in); got != want {
			t.Errorf("clamp(%d) = %d, want %d", in, got, want)
		}
	}
}

func TestExtFromMime(t *testing.T) {
	cases := []struct {
		mime, key, want string
	}{
		{"", "a.mp4", ".mp4"}, // key 已带扩展名优先
		{"image/jpeg", "a.jpg", ".jpg"},
		{"image/jpeg", "noext", ".jpg"},
		{"image/png", "noext", ".png"},
		{"video/mp4", "noext", ".mp4"},
		{"application/octet-stream", "noext", ".mp4"}, // 未知 mime 兜底 .mp4
		{"", "", ".mp4"},
	}
	for _, c := range cases {
		if got := extFromMime(c.mime, c.key); got != c.want {
			t.Errorf("extFromMime(%q, %q) = %q, want %q", c.mime, c.key, got, c.want)
		}
	}
}
