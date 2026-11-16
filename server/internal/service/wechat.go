// Package service 提供本项目业务服务。
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// WeChatSession 微信 code2session 返回
type WeChatSession struct {
	OpenID     string `json:"openid"`
	UnionID    string `json:"unionid"`
	SessionKey string `json:"session_key"`
	ErrCode    int    `json:"errcode"`
	ErrMsg     string `json:"errmsg"`
}

// WeChatService 微信小程序登录服务
type WeChatService struct {
	appID  string
	secret string
	client *http.Client
}

// NewWeChatService 创建微信登录服务
func NewWeChatService(appID, secret string) *WeChatService {
	return &WeChatService{
		appID:  appID,
		secret: secret,
		client: &http.Client{Timeout: 8 * time.Second},
	}
}

// Code2Session 用小程序 wx.login 的 code 换取 openid。
// code 一次性，5 分钟内有效；重复使用会返回 40163。
func (s *WeChatService) Code2Session(ctx context.Context, code string) (*WeChatSession, error) {
	u := fmt.Sprintf("https://api.weixin.qq.com/sns/jscode2session?appid=%s&secret=%s&js_code=%s&grant_type=authorization_code",
		url.QueryEscape(s.appID), url.QueryEscape(s.secret), url.QueryEscape(code))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求微信服务器失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("微信服务器返回状态码 %d", resp.StatusCode)
	}
	var out WeChatSession
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("解析微信响应失败: %w", err)
	}
	if out.ErrCode != 0 {
		return nil, fmt.Errorf("微信登录失败(%d): %s", out.ErrCode, out.ErrMsg)
	}
	if out.OpenID == "" {
		return nil, fmt.Errorf("微信登录失败: openid 为空")
	}
	return &out, nil
}
