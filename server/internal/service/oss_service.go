// Package service 提供本项目业务服务。
package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"acustomer-video-server/config"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// OSSService 封装 OSS 操作；未配置时自动降级本地 storage。
// 注意：本项目使用独立新 Bucket；AK/SK 复用现有 RAM 子账号凭证。
type OSSService struct {
	client  *oss.Client
	bucket  *oss.Bucket
	enabled bool
	cfg     *config.Config
}

var (
	ossOnce sync.Once
	ossInst *OSSService
)

// NewOSSService 单例
func NewOSSService(cfg *config.Config) *OSSService {
	ossOnce.Do(func() {
		ossInst = initOSS(cfg)
	})
	return ossInst
}

func initOSS(cfg *config.Config) *OSSService {
	s := &OSSService{cfg: cfg}
	if !cfg.OSSConfigured() {
		log.Println("[OSS] 未配置，使用本地存储降级模式")
		return s
	}
	client, err := oss.New(cfg.OSSEndpoint, cfg.OSSAccessKeyID, cfg.OSSAccessKeySecret)
	if err != nil {
		log.Printf("[OSS] 创建客户端失败，降级本地存储: %v", err)
		return s
	}
	bucket, err := client.Bucket(cfg.OSSBucket)
	if err != nil {
		log.Printf("[OSS] 获取 bucket 失败，降级本地存储: %v", err)
		return s
	}
	s.client, s.bucket, s.enabled = client, bucket, true
	log.Printf("[OSS] 已启用 bucket=%s prefix=%s", cfg.OSSBucket, cfg.OSSObjectPrefix)
	return s
}

func (s *OSSService) Enabled() bool { return s != nil && s.enabled }

// BuildObjectKey 服务端生成 objectKey（不信任客户端文件名），带项目前缀与 user 维度
func (s *OSSService) BuildObjectKey(userID int64, filename string) string {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		ext = ".bin"
	}
	base := path.Base(filename)
	base = strings.TrimSuffix(base, filepath.Ext(base))
	base = sanitize(base)
	if base == "" {
		base = "asset"
	}
	key := fmt.Sprintf("%su%d/%s-%s%s", s.cfg.OSSObjectPrefix, userID, base, nowNano(), ext)
	return strings.TrimPrefix(path.Clean("/"+key), "/")
}

// SignPutURL 生成预签名 PUT 上传 URL（1 小时有效）
func (s *OSSService) SignPutURL(objectKey, contentType string) (string, error) {
	key := cleanKey(objectKey)
	if !s.Enabled() {
		return "", nil // 本地模式：客户端直接 POST 到 /assets/upload
	}
	u, err := s.bucket.SignURL(key, oss.HTTPPut, 3600, oss.ContentType(contentType))
	if err != nil {
		return "", err
	}
	return u, nil
}

// SignGetURL 生成签名下载 URL（1 小时有效）
func (s *OSSService) SignGetURL(objectKey string) (string, error) {
	key := cleanKey(objectKey)
	if !s.Enabled() {
		return s.localURL(key), nil
	}
	u, err := s.bucket.SignURL(key, oss.HTTPGet, 3600)
	if err != nil {
		return "", err
	}
	return u, nil
}

// PublicURL 返回预览 URL（本地模式返回 /storage/...；OSS 模式返回签名 URL）
func (s *OSSService) PublicURL(objectKey string) string {
	key := cleanKey(objectKey)
	if !s.Enabled() {
		return s.localURL(key)
	}
	u, err := s.SignGetURL(key)
	if err != nil {
		return ""
	}
	return u
}

// PutLocal 本地模式：写文件到 storage 目录
func (s *OSSService) PutLocal(objectKey string, r io.Reader) error {
	key := cleanKey(objectKey)
	full := filepath.Join(s.cfg.StorageDir, key)
	if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
		return err
	}
	f, err := os.Create(full)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

// LocalPath 返回 objectKey 对应的本地路径（供 worker ffmpeg 读取）
func (s *OSSService) LocalPath(objectKey string) string {
	return filepath.Join(s.cfg.StorageDir, cleanKey(objectKey))
}

// DownloadToLocal 从 OSS 下载素材到本地临时路径（供 ffmpeg 使用）；本地模式直接复制
func (s *OSSService) DownloadToLocal(objectKey, destPath string) error {
	key := cleanKey(objectKey)
	if s.Enabled() {
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}
		return s.bucket.GetObjectToFile(key, destPath)
	}
	src, err := os.Open(s.LocalPath(key))
	if err != nil {
		return err
	}
	defer src.Close()
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}
	dst, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	_, err = io.Copy(dst, src)
	return err
}

// UploadLocalFile 上传本地文件到 OSS（worker 成片回传）；本地模式为复制到 storage
func (s *OSSService) UploadLocalFile(localPath, objectKey string) error {
	key := cleanKey(objectKey)
	if s.Enabled() {
		return s.bucket.PutObjectFromFile(key, localPath)
	}
	return copyFile(localPath, filepath.Join(s.cfg.StorageDir, key))
}

func (s *OSSService) localURL(objectKey string) string {
	return strings.TrimRight(s.cfg.StorageBase, "/") + "/" + cleanKey(objectKey)
}

func cleanKey(k string) string {
	return strings.TrimPrefix(path.Clean("/"+strings.TrimSpace(k)), "/")
}

func sanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r == '-' || r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

func nowNano() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
