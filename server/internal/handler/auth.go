package handler

import (
	"context"
	"net/http"
	"time"

	"golang.org/x/crypto/bcrypt"

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

// validPhone 大陆手机号：1 开头 + 3-9 第二位 + 共 11 位数字
func validPhone(s string) bool {
	if len(s) != 11 {
		return false
	}
	if s[0] != '1' {
		return false
	}
	if s[1] < '3' || s[1] > '9' {
		return false
	}
	for i := 2; i < 11; i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}

// validPassword 至少 8 位
func validPassword(s string) bool {
	return len(s) >= 8
}

// Register 手机号+密码注册
func Register(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone"`
			Password string `json:"password"`
			Nickname string `json:"nickname"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, 400, "参数错误")
			return
		}
		if !validPhone(req.Phone) {
			fail(c, http.StatusBadRequest, 400, "手机号格式不正确")
			return
		}
		if !validPassword(req.Password) {
			fail(c, http.StatusBadRequest, 400, "密码至少 8 位")
			return
		}
		if existing, _ := repository.GetUserByPhone(req.Phone); existing != nil {
			fail(c, http.StatusConflict, 409, "该手机号已注册，请直接登录")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "注册失败")
			return
		}
		if req.Nickname == "" {
			req.Nickname = "用户" + req.Phone[len(req.Phone)-4:]
		}
		uid, err := repository.CreateUserWithPhone(req.Phone, string(hash), req.Nickname)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "注册失败: "+err.Error())
			return
		}
		token, err := middleware.IssueToken(cfg.JWTSecret, uid, 24*7)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "签发令牌失败")
			return
		}
		ok(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       uid,
				"nickname": req.Nickname,
				"phone":    req.Phone,
				"platform": "phone",
			},
		})
	}
}

// PhoneLogin 手机号+密码登录
func PhoneLogin(cfg *config.Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, 400, "参数错误")
			return
		}
		u, _ := repository.GetUserByPhone(req.Phone)
		if u == nil || u.PasswordHash == "" {
			fail(c, http.StatusUnauthorized, 401, "手机号或密码错误")
			return
		}
		if bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)) != nil {
			fail(c, http.StatusUnauthorized, 401, "手机号或密码错误")
			return
		}
		if u.Status == 0 {
			fail(c, http.StatusForbidden, 403, "账号已被禁用，请联系客服")
			return
		}
		token, err := middleware.IssueToken(cfg.JWTSecret, u.ID, 24*7)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "签发令牌失败")
			return
		}
		ok(c, gin.H{
			"token": token,
			"user": gin.H{
				"id":       u.ID,
				"nickname": u.Nickname,
				"phone":    u.Phone,
				"platform": u.Platform,
			},
		})
	}
}

// BindPhone 已登录用户（微信）绑定手机号并设置密码
func BindPhone() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Phone    string `json:"phone"`
			Password string `json:"password"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, http.StatusBadRequest, 400, "参数错误")
			return
		}
		if !validPhone(req.Phone) {
			fail(c, http.StatusBadRequest, 400, "手机号格式不正确")
			return
		}
		if !validPassword(req.Password) {
			fail(c, http.StatusBadRequest, 400, "密码至少 8 位")
			return
		}
		uid := currentUserID(c)
		if existing, _ := repository.GetUserByPhone(req.Phone); existing != nil && existing.ID != uid {
			fail(c, http.StatusConflict, 409, "该手机号已被其他账号绑定")
			return
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			fail(c, http.StatusInternalServerError, 500, "绑定失败")
			return
		}
		if err := repository.BindPhone(uid, req.Phone, string(hash)); err != nil {
			fail(c, http.StatusInternalServerError, 500, "绑定失败: "+err.Error())
			return
		}
		ok(c, gin.H{"phone": req.Phone})
	}
}
