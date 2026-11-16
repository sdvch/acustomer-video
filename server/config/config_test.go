package config

import (
	"strings"
	"testing"
)

func validConfig() *Config {
	return &Config{
		DBName:          "acustomer_video",
		TargetWidth:     1080,
		TargetHeight:    1920,
		TargetFPS:       30,
		WorkerCount:     2,
		MinClipSeconds:  2,
		MaxClipSeconds:  8,
		MinDurationSec:  25,
		MaxDurationSec:  35,
		MinAssetCount:   3,
		MaxAssetCount:   12,
		DBCharset:       "utf8mb4",
		StorageDir:      "../storage",
		StorageBase:     "/storage",
		PublicBaseURL:   "",
		OSSObjectPrefix: "acustomer-video/",
	}
}

// 历史事故：STORAGE_DIR 相对路径曾在容器内解析到卷外导致数据丢失。
// 这里锁定默认值与访问前缀的约定，防止有人无意改掉部署假设。
func TestStorageDefaults(t *testing.T) {
	c := validConfig()
	if err := c.validate(); err != nil {
		t.Fatalf("合法配置不应报错: %v", err)
	}
	if c.StorageBase != "/storage" {
		t.Errorf("StorageBase 默认应为 /storage，实际 %q", c.StorageBase)
	}
	if !strings.HasPrefix(c.OSSObjectPrefix, "acustomer-video/") {
		t.Errorf("OSSObjectPrefix 应带项目前缀，实际 %q", c.OSSObjectPrefix)
	}
}

func TestValidateDBIsolation(t *testing.T) {
	c := validConfig()
	c.DBName = "storyboard"
	if err := c.validate(); err == nil {
		t.Error("DB_NAME 与旧项目 storyboard 相同时必须报错（强制隔离）")
	}
	c.DBName = "Storyboard"
	if err := c.validate(); err == nil {
		t.Error("DB_NAME 隔离校验应不区分大小写")
	}
	c.DBName = ""
	if err := c.validate(); err == nil {
		t.Error("DB_NAME 为空必须报错")
	}
}

func TestValidateDurationRange(t *testing.T) {
	c := validConfig()
	c.MinDurationSec = 36
	if err := c.validate(); err == nil {
		t.Error("MinDuration > MaxDuration 必须报错")
	}
	c = validConfig()
	c.MaxDurationSec = 0
	if err := c.validate(); err == nil {
		t.Error("MaxDuration <= 0 必须报错")
	}
	c = validConfig()
	c.TargetFPS = 0
	if err := c.validate(); err == nil {
		t.Error("成片规格为 0 必须报错")
	}
}

func TestValidateClamping(t *testing.T) {
	c := validConfig()
	c.WorkerCount = 0
	c.MinClipSeconds = 0
	c.MaxClipSeconds = 1 // 小于 MinClip（钳到 1 后相等）
	if err := c.validate(); err != nil {
		t.Fatalf("应自动钳制而不是报错: %v", err)
	}
	if c.WorkerCount != 1 {
		t.Errorf("WorkerCount < 1 应钳到 1，实际 %d", c.WorkerCount)
	}
	if c.MinClipSeconds != 1 {
		t.Errorf("MinClipSeconds < 1 应钳到 1，实际 %d", c.MinClipSeconds)
	}
	if c.MaxClipSeconds < c.MinClipSeconds {
		t.Errorf("钳制后 MaxClip(%d) 不应小于 MinClip(%d)", c.MaxClipSeconds, c.MinClipSeconds)
	}
}

func TestValidateAssetCountRange(t *testing.T) {
	c := validConfig()
	c.MaxAssetCount = 2 // 小于 MinAssetCount=3
	if err := c.validate(); err == nil {
		t.Error("MaxAssetCount < MinAssetCount 必须报错")
	}
}

func TestOSSConfigured(t *testing.T) {
	c := validConfig()
	if c.OSSConfigured() {
		t.Error("未配置任何 OSS 凭证时不应视为已启用")
	}
	c.OSSAccessKeyID = "ak"
	if c.OSSConfigured() {
		t.Error("只配 AK 不配 SK/bucket/endpoint 时不应视为已启用")
	}
	c.OSSAccessKeySecret, c.OSSBucket, c.OSSEndpoint = "sk", "b", "e"
	if !c.OSSConfigured() {
		t.Error("四项齐全时应视为已启用")
	}
}

func TestWeChatLoginConfigured(t *testing.T) {
	c := validConfig()
	if c.WeChatLoginConfigured() {
		t.Error("未配置 AppID/Secret 时应走本地开发单用户")
	}
	c.WeChatAppID = "wx123"
	if c.WeChatLoginConfigured() {
		t.Error("只配 AppID 不应视为已配置")
	}
	c.WeChatAppSecret = "secret"
	if !c.WeChatLoginConfigured() {
		t.Error("AppID 与 Secret 齐全时应视为已配置")
	}
}

func TestDSN(t *testing.T) {
	c := validConfig()
	c.DBUser, c.DBPassword, c.DBHost, c.DBPort = "u", "p", "h", 13306
	dsn := c.DSN()
	want := "u:p@tcp(h:13306)/acustomer_video?charset=utf8mb4&parseTime=True&loc=Local&multiStatements=true"
	if dsn != want {
		t.Errorf("DSN 格式不符\n got: %s\nwant: %s", dsn, want)
	}
}

func TestGetEnvFallback(t *testing.T) {
	t.Setenv("TEST_ACV_KEY", "value")
	if got := getEnv("TEST_ACV_KEY", "def"); got != "value" {
		t.Errorf("已设置时应返回环境值，实际 %q", got)
	}
	t.Setenv("TEST_ACV_EMPTY", "")
	if got := getEnv("TEST_ACV_EMPTY", "def"); got != "def" {
		t.Errorf("空字符串应回退默认值，实际 %q", got)
	}
	if got := getEnv("TEST_ACV_MISSING", "def"); got != "def" {
		t.Errorf("未设置时应返回默认值，实际 %q", got)
	}
}

func TestGetEnvIntFallback(t *testing.T) {
	t.Setenv("TEST_ACV_INT", "42")
	if got := getEnvInt("TEST_ACV_INT", 7); got != 42 {
		t.Errorf("合法数字应被解析，实际 %d", got)
	}
	t.Setenv("TEST_ACV_INT_BAD", "not-a-number")
	if got := getEnvInt("TEST_ACV_INT_BAD", 7); got != 7 {
		t.Errorf("非法数字应回退默认值，实际 %d", got)
	}
	t.Setenv("TEST_ACV_INT_EMPTY", "")
	if got := getEnvInt("TEST_ACV_INT_EMPTY", 7); got != 7 {
		t.Errorf("空字符串应回退默认值，实际 %d", got)
	}
}
