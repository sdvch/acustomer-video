package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() { gin.SetMode(gin.TestMode) }

func runWithAuth(secret, authHeader string) (*httptest.ResponseRecorder, any) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	var userID any
	r.GET("/p", JWTAuth(secret), func(c *gin.Context) {
		userID, _ = c.Get("userID")
		c.Status(http.StatusOK)
	})
	req := httptest.NewRequest(http.MethodGet, "/p", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w, userID
}

func TestJWTAuthValidToken(t *testing.T) {
	token, err := IssueToken("secret", 42, 7)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	w, uid := runWithAuth("secret", "Bearer "+token)
	if w.Code != http.StatusOK {
		t.Errorf("有效 token 应放行，实际 %d body=%s", w.Code, w.Body.String())
	}
	if uid != int64(42) {
		t.Errorf("应从 claims 注入 userID=42，实际 %v", uid)
	}
}

func TestJWTAuthMissingHeader(t *testing.T) {
	w, _ := runWithAuth("secret", "")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("缺失 Authorization 应 401，实际 %d", w.Code)
	}
}

func TestJWTAuthGarbageToken(t *testing.T) {
	w, _ := runWithAuth("secret", "Bearer not-a-jwt")
	if w.Code != http.StatusUnauthorized {
		t.Errorf("非法 token 应 401，实际 %d", w.Code)
	}
}

func TestJWTAuthExpiredToken(t *testing.T) {
	token, err := IssueToken("secret", 42, -1) // 已过期
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	time.Sleep(time.Second) // 确保 exp < now（秒级精度）
	w, _ := runWithAuth("secret", "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("过期 token 应 401，实际 %d", w.Code)
	}
}

func TestJWTAuthWrongSecret(t *testing.T) {
	token, err := IssueToken("other-secret", 42, 7)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	w, _ := runWithAuth("secret", "Bearer "+token)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("密钥不符的 token 应 401，实际 %d", w.Code)
	}
}

// alg=none 攻击：无签名的 token 必须被 HMAC 方法校验拦下。
func TestJWTAuthAlgNoneRejected(t *testing.T) {
	// header{"alg":"none"}.payload{uid:1}.（空签名）
	noneToken := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0.eyJ1aWQiOjF9."
	w, _ := runWithAuth("secret", "Bearer "+noneToken)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("alg=none token 应 401，实际 %d", w.Code)
	}
}

func TestIssueTokenFormat(t *testing.T) {
	token, err := IssueToken("secret", 1, 7)
	if err != nil {
		t.Fatalf("签发失败: %v", err)
	}
	if len(token) == 0 || token[0] == '.' {
		t.Error("token 不应为空")
	}
}
