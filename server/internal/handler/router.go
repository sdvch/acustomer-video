package handler

import (
	"os"

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

	// 本地降级模式：素材与成片由后端直接提供静态访问；OSS 模式改用签名 URL
	if !oss.Enabled() {
		_ = os.MkdirAll(cfg.StorageDir, 0755)
		r.Static(cfg.StorageBase, cfg.StorageDir)
	}

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", Health(cfg))
		v1.POST("/auth/login", Login(cfg))
		v1.POST("/auth/login-phone", PhoneLogin(cfg))
		v1.POST("/auth/register", Register(cfg))

		authed := v1.Group("")
		authed.Use(middleware.JWTAuth(cfg.JWTSecret))
		{
			authed.GET("/auth/profile", Profile())
			authed.PUT("/auth/bind-phone", BindPhone())

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

		// 管理后台（scope=admin 独立令牌，httpOnly Cookie 或 Bearer）
		adminH := NewAdminHandler(cfg)
		admin := v1.Group("/admin")
		{
			admin.POST("/auth/login", adminH.Login)
			adminAuthed := admin.Group("")
			adminAuthed.Use(middleware.AdminAuth(cfg.JWTSecret))
			{
				adminAuthed.POST("/auth/logout", adminH.Logout)
				adminAuthed.GET("/auth/profile", adminH.Profile)

				// 用户管理
				adminAuthed.GET("/users", adminH.ListUsers)
				adminAuthed.GET("/users/:id", adminH.UserDetail)
				adminAuthed.PATCH("/users/:id/status", adminH.SetUserStatus)
				adminAuthed.POST("/users/:id/reset-password", adminH.ResetUserPassword)
				adminAuthed.GET("/users/:id/projects", adminH.UserProjects)

				// 数据总览 / 内容审查
				adminAuthed.GET("/stats/overview", adminH.StatsOverview)
				adminAuthed.DELETE("/videos/:id", adminH.DeleteVideo)

				// 管理员账号管理（handler 内校验超管权限）
				adminAuthed.GET("/admins", adminH.ListAdmins)
				adminAuthed.POST("/admins", adminH.CreateAdmin)
				adminAuthed.PATCH("/admins/:id", adminH.UpdateAdmin)
			}
		}
	}
	return r
}
