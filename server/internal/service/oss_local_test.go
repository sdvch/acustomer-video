package service

import (
	"os"
	"strings"
	"testing"

	"acustomer-video-server/config"
)

// 直接构造 OSSService（绕过 NewOSSService 的 sync.Once 单例），
// 只测本地降级路径与 URL 拼接，不触碰真实 OSS 客户端。
func localService(storageDir, storageBase, publicBaseURL string) *OSSService {
	return &OSSService{cfg: &config.Config{
		StorageDir:      storageDir,
		StorageBase:     storageBase,
		PublicBaseURL:   publicBaseURL,
		OSSObjectPrefix: "acustomer-video/",
	}}
}

// 历史事故：本地模式 URL 曾是相对路径，小程序保存相册/播放拿不到资源。
// PUBLIC_BASE_URL 配置后必须拼出绝对地址。
func TestLocalURLWithPublicBase(t *testing.T) {
	s := localService("/data/storage", "/storage/", "http://8.152.208.234/")
	got := s.localURL("u1/a.mp4")
	if got != "http://8.152.208.234/storage/u1/a.mp4" {
		t.Errorf("应拼出绝对 URL，实际 %q", got)
	}
}

func TestLocalURLWithoutPublicBase(t *testing.T) {
	s := localService("/data/storage", "/storage", "")
	got := s.localURL("u1/a.mp4")
	if got != "/storage/u1/a.mp4" {
		t.Errorf("未配置 PUBLIC_BASE_URL 时应为前缀相对 URL，实际 %q", got)
	}
}

func TestPublicURLLocalFallback(t *testing.T) {
	s := localService("/data/storage", "/storage", "http://h")
	if got := s.PublicURL("u1/a.mp4"); got != "http://h/storage/u1/a.mp4" {
		t.Errorf("本地降级模式 PublicURL 应走 localURL，实际 %q", got)
	}
}

func TestCleanKey(t *testing.T) {
	cases := map[string]string{
		"u1/a.mp4":      "u1/a.mp4",
		"/u1/a.mp4":     "u1/a.mp4",
		"  u1/a.mp4  ":  "u1/a.mp4",
		"../u1/a.mp4":   "u1/a.mp4",
		"/../../secret": "secret",
	}
	for in, want := range cases {
		if got := cleanKey(in); got != want {
			t.Errorf("cleanKey(%q) = %q, want %q", in, got, want)
		}
	}
}

// BuildObjectKey 不得信任客户端文件名：路径穿越、非法字符、空名都要被清洗。
func TestBuildObjectKey(t *testing.T) {
	s := localService("/data/storage", "/storage", "")

	k := s.BuildObjectKey(7, "../../etc/passwd")
	if !strings.HasPrefix(k, "acustomer-video/u7/") {
		t.Errorf("objectKey 应带项目前缀与 user 维度，实际 %q", k)
	}
	if strings.Contains(k, "..") || strings.Contains(k, "/etc") {
		t.Errorf("objectKey 不应包含路径穿越片段，实际 %q", k)
	}

	k2 := s.BuildObjectKey(7, "我的 视频.mp4")
	if !strings.HasSuffix(k2, ".mp4") {
		t.Errorf("扩展名应保留小写，实际 %q", k2)
	}

	k3 := s.BuildObjectKey(7, "")
	if !strings.Contains(k3, "asset") {
		t.Errorf("空文件名应回退 asset，实际 %q", k3)
	}
}

// PutLocal + LocalPath 往返：文件落在 StorageDir 下，路径可回读。
func TestPutLocalAndLocalPath(t *testing.T) {
	dir := t.TempDir()
	s := localService(dir, "/storage", "")

	if err := s.PutLocal("u9/sub/a.mp4", strings.NewReader("hello")); err != nil {
		t.Fatalf("PutLocal 失败: %v", err)
	}
	if got := s.LocalPath("u9/sub/a.mp4"); !strings.HasPrefix(got, dir) {
		t.Errorf("LocalPath 必须落在 StorageDir 内，实际 %q", got)
	}
	data, err := os.ReadFile(s.LocalPath("u9/sub/a.mp4"))
	if err != nil || string(data) != "hello" {
		t.Errorf("写入内容应可回读，err=%v data=%q", err, string(data))
	}
}
