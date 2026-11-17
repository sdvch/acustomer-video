package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"acustomer-video-server/internal/repository"
)

// JWTAuth 用户侧鉴权中间件：所有业务接口必须携带有效 token。
// - 拒绝 scope=admin 的管理端 token（作用域隔离）
// - 每次请求核对用户状态，被禁用立即失效
func JWTAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, abort := parseToken(c, secret)
		if abort {
			return
		}
		if scope, _ := claims["scope"].(string); scope == "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
			c.Abort()
			return
		}
		uid, ok := claims["uid"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
			c.Abort()
			return
		}
		// 禁用拦截：旧 token 再有效也立刻失效
		if repository.UserDisabled(int64(uid)) {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "账号已被禁用"})
			c.Abort()
			return
		}
		c.Set("userID", int64(uid))
		c.Next()
	}
}

// AdminAuth 管理端鉴权中间件：只接受 scope=admin 的 token，
// 来源兼容 httpOnly Cookie（admin_token）与 Authorization Bearer（curl 调试用）。
func AdminAuth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, abort := parseToken(c, secret)
		if abort {
			return
		}
		if scope, _ := claims["scope"].(string); scope != "admin" {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
			c.Abort()
			return
		}
		aid, ok := claims["admin_id"].(float64)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
			c.Abort()
			return
		}
		role, _ := claims["role"].(string)
		c.Set("adminID", int64(aid))
		c.Set("adminRole", role)
		c.Next()
	}
}

// parseToken 从 Cookie 或 Authorization 头解析并校验 JWT；失败时已写响应并返回 abort=true
func parseToken(c *gin.Context, secret string) (jwt.MapClaims, bool) {
	tokenStr := c.GetHeader("Authorization")
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")
	if tokenStr == "" {
		tokenStr, _ = c.Cookie("admin_token")
	}
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "未登录"})
		c.Abort()
		return nil, true
	}
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(secret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
		c.Abort()
		return nil, true
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "data": nil, "message": "invalid token"})
		c.Abort()
		return nil, true
	}
	return claims, false
}

// IssueToken 签发用户 JWT（uid + exp；无 scope 声明，向后兼容旧 token）
func IssueToken(secret string, uid int64, ttlHours int) (string, error) {
	claims := jwt.MapClaims{
		"uid": uid,
		"exp": time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// IssueAdminToken 签发管理端 JWT（scope=admin + role）
func IssueAdminToken(secret string, adminID int64, role string, ttlHours int) (string, error) {
	claims := jwt.MapClaims{
		"admin_id": adminID,
		"role":     role,
		"scope":    "admin",
		"exp":      time.Now().Add(time.Duration(ttlHours) * time.Hour).Unix(),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}
