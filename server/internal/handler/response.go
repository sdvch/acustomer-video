package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Resp 统一响应结构
type Resp struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

func ok(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Resp{Code: 0, Data: data, Message: "ok"})
}

func fail(c *gin.Context, httpStatus int, code int, msg string) {
	c.JSON(httpStatus, Resp{Code: code, Data: nil, Message: msg})
}

// currentUserID 从 JWT 中间件注入的上下文取用户 ID；MVP 阶段未登录时用默认用户
func currentUserID(c *gin.Context) int64 {
	if v, ok := c.Get("userID"); ok {
		if id, ok := v.(int64); ok {
			return id
		}
	}
	return 1 // MVP 默认用户
}
