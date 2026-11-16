package handler

import (
	"context"
	"net/http"
	"time"

	"acustomer-video-server/config"
	"acustomer-video-server/internal/middleware"
	"acustomer-video-server/internal/repository"
	"acustomer-video-server/internal/service"
	"github.com/gin-gonic/gin"
)

// Login 微信小程序登录：用 wx.login 的 code 换 openid，再签发本项目 JWT。
// 未配置 WECHAT_APP_ID/SECRET 时降级为本地开发单用户（dev_mode=true），便于本机联调。
func Login(cfg *config.Config) gin.HandlerFunc {
	wx := service.NewWeChatService(cfg.WeChatAppID, cfg.WeChatAppSecret)

	return func(c *gin.Context) {
		var req struct {
			Platform string `json:"platform"`
			Code     string `json:"code"`
			Nickname string `json:"nickname"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, 400, "参数错误")
			return
		}
		if req.Platform == "" {
			req.Platform = "wechat"
		}
		if req.Code == "" {
			fail(c, http.StatusBadRequest, 400, "code 不能为空")
			return
		}

		var (
			openID  string
			unionID string
			devMode bool
		)

		if cfg.WeChatLoginConfigured() && req.Platform == "wechat" {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
			defer cancel()
			sess, err := wx.Code2Session(ctx, req.Code)
			if err != nil {
				// 微信侧的 code 过期/重复使用属于客户端需重试的错误
				fail(c, http.StatusBadRequest, 4001, "微信登录失败，请重试: "+err.Error())
				return
			}
			openID, unionID = sess.OpenID, sess.UnionID
		} else {
			// 本地开发模式：所有请求落到同一个开发用户
			openID, devMode = "dev_local_user", true
		}

		uid, _, err := repository.UpsertUser(openID, unionID, req.Platform, req.Nickname)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "登录失败: "+err.Error())
			return
		}

		token, err := middleware.IssueToken(cfg.JWTSecret, uid, 24*7)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "签发令牌失败")
			return
		}
		ok(c, gin.H{
			"token":    token,
			"dev_mode": devMode,
			"user": gin.H{
				"id":       uid,
				"platform": req.Platform,
				"open_id":  maskOpenID(openID),
			},
		})
	}
}

// Profile 当前登录用户信息
func Profile() gin.HandlerFunc {
	return func(c *gin.Context) {
		u, err := repository.GetUser(currentUserID(c))
		if err != nil || u == nil {
			fail(c, http.StatusNotFound, 404, "用户不存在")
			return
		}
		u.OpenID = maskOpenID(u.OpenID)
		ok(c, u)
	}
}

// maskOpenID 只保留首尾各 4 位，避免把完整 openid 暴露给前端
func maskOpenID(s string) string {
	if len(s) <= 10 {
		return s
	}
	return s[:4] + "****" + s[len(s)-4:]
}
