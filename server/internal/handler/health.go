package handler

import (
	"os/exec"

	"acustomer-video-server/config"
	"github.com/gin-gonic/gin"
)

// Health 健康检查：含 ffmpeg/ffprobe 可用性检测
func Health(cfg *config.Config) gin.HandlerFunc {
	ffmpegPath, ffprobeErr := exec.LookPath(cfg.FFmpegPath)
	_, probeErr := exec.LookPath(cfg.FFprobePath)
	return func(c *gin.Context) {
		data := gin.H{
			"status":         "ok",
			"service":        "acustomer-video-server",
			"ffmpeg":         ffmpegPath,
			"ffmpeg_ok":      ffprobeErr == nil,
			"ffprobe_ok":     probeErr == nil,
			"oss_configured": cfg.OSSConfigured(),
			"oss_bucket":     cfg.OSSBucket,
			"db_name":        cfg.DBName,
		}
		ok(c, data)
	}
}
