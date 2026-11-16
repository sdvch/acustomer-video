package handler

import (
	"acustomer-video-server/config"
	"acustomer-video-server/internal/middleware"
	"acustomer-video-server/internal/service"
	"github.com/gin-gonic/gin"
)

// NewRouter 组装路由。M2/M3/M4 里程碑会在此追加更多路由组。
func NewRouter(cfg *config.Config) *gin.Engine {
	r := gin.New()
	r.Use(middleware.Recover(), middleware.CORS())

	oss := service.NewOSSService(cfg)
	assetH := NewAssetHandler(cfg, oss)
	projectH := NewProjectHandler(cfg, oss)
	aiH := NewAIHandler(cfg)

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", Health(cfg))
		v1.POST("/auth/login", Login(cfg))

		authed := v1.Group("")
		authed.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			authed.GET("/auth/profile", Profile())

			// M2: 素材上传链路
			authed.POST("/assets/presign", assetH.Presign)
			authed.POST("/assets/upload", assetH.Upload) // 本地降级模式
			authed.POST("/assets/confirm", assetH.Confirm)
			authed.GET("/assets", assetH.List)
			authed.PATCH("/assets/sort", assetH.Sort)
			authed.DELETE("/assets/:id", assetH.Delete)

			// M4: 项目 / 合成任务 / 成片
			authed.POST("/projects", projectH.Create)
			authed.GET("/projects", projectH.List)
			authed.GET("/projects/:id", projectH.Detail)
			authed.POST("/projects/:id/compose", projectH.Compose)
			authed.GET("/jobs/:id", projectH.JobStatus)
			authed.GET("/videos", projectH.ListVideos)
			authed.GET("/videos/:id/download-url", projectH.DownloadURL)
			authed.DELETE("/videos/:id", projectH.DeleteVideo)

			// SOP 阶段五：文案生成（DeepSeek）
			authed.POST("/ai/copy", aiH.GenerateCopy)
		}
	}
	return r
}
