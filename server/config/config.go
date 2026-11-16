// Package config 加载并校验本项目配置。
// 注意：本项目为独立项目，所有配置项均从自身 .env 读取，不引用 storyboard-system 的任何配置文件。
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/joho/godotenv"
)

// Config 全局配置单例
type Config struct {
	ServerPort int
	GinMode    string
	JWTSecret  string

	DBHost     string
	DBPort     int
	DBUser     string
	DBPassword string
	DBName     string
	DBCharset  string

	// OSS：新 Bucket；AK/SK 复用现有 RAM 子账号凭证（值来自本项目 .env）
	OSSEndpoint        string
	OSSPublicEndpoint  string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSBucket          string
	OSSObjectPrefix    string
	OSSRegion          string

	StorageDir  string // 本地降级目录
	StorageBase string // 本地降级访问前缀

	// 微信小程序登录（code2session）。未配置时降级为本地开发单用户。
	WeChatAppID     string
	WeChatAppSecret string

	// DeepSeek（文案等文本类大模型统一走这里）
	DeepSeekAPIKey     string
	DeepSeekBaseURL    string
	DeepSeekModel      string
	DeepSeekTimeoutSec int

	FFmpegPath  string
	FFprobePath string
	WorkDir     string // ffmpeg 临时工作目录

	WorkerCount       int
	JobLeaseSeconds   int
	ComposeTimeoutMin int

	DefaultClipSeconds int
	MinClipSeconds     int // 单镜头最短（SOP 3s，这里放宽到 2s）
	MaxClipSeconds     int // 单镜头最长（SOP 5s，这里放宽到 8s）
	MinDurationSec     int // 成片最短（SOP 25s）
	MaxDurationSec     int // 成片最长（SOP 35s）
	MinAssetCount      int
	MaxAssetCount      int
	TargetWidth        int
	TargetHeight       int
	TargetFPS          int
	TargetBitrate      string
}

var (
	once sync.Once
	cfg  *Config
)

// Load 读取 .env（可选）并返回全局配置单例
func Load() (*Config, error) {
	var err error
	once.Do(func() {
		_ = godotenv.Load() // .env 不存在时用进程环境（容器场景）

		c := &Config{
			ServerPort:         getEnvInt("SERVER_PORT", 18080),
			GinMode:            getEnv("GIN_MODE", "release"),
			JWTSecret:          getEnv("JWT_SECRET", "acustomer-video-dev-secret-change-me"),
			DBHost:             getEnv("DB_HOST", "127.0.0.1"),
			DBPort:             getEnvInt("DB_PORT", 13306),
			DBUser:             getEnv("DB_USER", "acustomer"),
			DBPassword:         getEnv("DB_PASSWORD", ""),
			DBName:             getEnv("DB_NAME", "acustomer_video"),
			DBCharset:          getEnv("DB_CHARSET", "utf8mb4"),
			OSSEndpoint:        getEnv("ALIYUN_OSS_ENDPOINT", ""),
			OSSPublicEndpoint:  getEnv("ALIYUN_OSS_PUBLIC_ENDPOINT", ""),
			OSSAccessKeyID:     getEnv("ALIYUN_OSS_ACCESS_KEY_ID", ""),
			OSSAccessKeySecret: getEnv("ALIYUN_OSS_ACCESS_KEY_SECRET", ""),
			OSSBucket:          getEnv("ALIYUN_OSS_BUCKET", ""),
			OSSObjectPrefix:    getEnv("ALIYUN_OSS_OBJECT_PREFIX", "acustomer-video/"),
			OSSRegion:          getEnv("ALIYUN_OSS_REGION", ""),
			StorageDir:         getEnv("STORAGE_DIR", "../storage"),
			StorageBase:        getEnv("STORAGE_BASE_PATH", "/storage"),
			WeChatAppID:        getEnv("WECHAT_APP_ID", ""),
			WeChatAppSecret:    getEnv("WECHAT_APP_SECRET", ""),
			DeepSeekAPIKey:     getEnv("DEEPSEEK_API_KEY", ""),
			DeepSeekBaseURL:    getEnv("DEEPSEEK_BASE_URL", "https://api.deepseek.com"),
			DeepSeekModel:      getEnv("DEEPSEEK_MODEL", "deepseek-v4-flash"),
			DeepSeekTimeoutSec: getEnvInt("DEEPSEEK_REQUEST_TIMEOUT_SECONDS", 120),
			FFmpegPath:         getEnv("FFMPEG_PATH", "ffmpeg"),
			FFprobePath:        getEnv("FFPROBE_PATH", "ffprobe"),
			WorkDir:            getEnv("WORK_DIR", "../storage/work"),
			WorkerCount:        getEnvInt("WORKER_COUNT", 2),
			JobLeaseSeconds:    getEnvInt("JOB_LEASE_SECONDS", 120),
			ComposeTimeoutMin:  getEnvInt("COMPOSE_TIMEOUT_MINUTES", 15),
			DefaultClipSeconds: getEnvInt("DEFAULT_CLIP_SECONDS", 4),
			MinClipSeconds:     getEnvInt("MIN_CLIP_SECONDS", 2),
			MaxClipSeconds:     getEnvInt("MAX_CLIP_SECONDS", 8),
			MinDurationSec:     getEnvInt("MIN_DURATION_SECONDS", 25),
			MaxDurationSec:     getEnvInt("MAX_DURATION_SECONDS", 35),
			MinAssetCount:      getEnvInt("MIN_ASSET_COUNT", 3),
			MaxAssetCount:      getEnvInt("MAX_ASSET_COUNT", 12),
			TargetWidth:        getEnvInt("TARGET_WIDTH", 1080),
			TargetHeight:       getEnvInt("TARGET_HEIGHT", 1920),
			TargetFPS:          getEnvInt("TARGET_FPS", 30),
			TargetBitrate:      getEnv("TARGET_BITRATE", "12M"),
		}
		if err = c.validate(); err != nil {
			return
		}
		cfg = c
	})
	return cfg, err
}

func (c *Config) validate() error {
	if c.DBName == "" {
		return fmt.Errorf("DB_NAME 不能为空")
	}
	// 强制隔离：本项目数据库名不得与旧项目相同
	if strings.EqualFold(c.DBName, "storyboard") {
		return fmt.Errorf("DB_NAME=%q 与旧项目冲突，本项目必须使用独立数据库", c.DBName)
	}
	if c.TargetWidth <= 0 || c.TargetHeight <= 0 || c.TargetFPS <= 0 {
		return fmt.Errorf("成片规格必须为正数")
	}
	if c.WorkerCount < 1 {
		c.WorkerCount = 1
	}
	if c.MinClipSeconds < 1 {
		c.MinClipSeconds = 1
	}
	if c.MaxClipSeconds < c.MinClipSeconds {
		c.MaxClipSeconds = c.MinClipSeconds
	}
	if c.MinDurationSec < 1 || c.MaxDurationSec <= c.MinDurationSec {
		return fmt.Errorf("MIN_DURATION_SECONDS/MAX_DURATION_SECONDS 配置不合法")
	}
	if c.MinAssetCount < 1 || c.MaxAssetCount < c.MinAssetCount {
		return fmt.Errorf("MIN_ASSET_COUNT/MAX_ASSET_COUNT 配置不合法")
	}
	return nil
}

// WeChatLoginConfigured 是否配置了微信 code2session（否则使用本地开发单用户）
func (c *Config) WeChatLoginConfigured() bool {
	return c.WeChatAppID != "" && c.WeChatAppSecret != ""
}

// LLMConfigured 是否配置了 DeepSeek
func (c *Config) LLMConfigured() bool {
	return c.DeepSeekAPIKey != ""
}

// DSN 返回 MySQL 连接串
func (c *Config) DSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=True&loc=Local&multiStatements=true",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName, c.DBCharset)
}

// OSSConfigured 是否配置了 OSS（否则降级本地存储）
func (c *Config) OSSConfigured() bool {
	return c.OSSAccessKeyID != "" && c.OSSAccessKeySecret != "" && c.OSSBucket != "" && c.OSSEndpoint != ""
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}
