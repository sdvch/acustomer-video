// Package llm 统一封装文本类大模型调用（默认 DeepSeek，OpenAI 兼容协议）。
package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"acustomer-video-server/config"
)

// Client DeepSeek 客户端（OpenAI 兼容 /chat/completions）
type Client struct {
	apiKey  string
	baseURL string
	model   string
	hc      *http.Client
}

// New 从配置创建客户端
func New(cfg *config.Config) *Client {
	timeout := time.Duration(cfg.DeepSeekTimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Client{
		apiKey:  cfg.DeepSeekAPIKey,
		baseURL: strings.TrimRight(cfg.DeepSeekBaseURL, "/"),
		model:   cfg.DeepSeekModel,
		hc:      &http.Client{Timeout: timeout},
	}
}

// Enabled 是否已配置可用
func (c *Client) Enabled() bool { return c != nil && c.apiKey != "" }

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// Chat 单轮对话，system + user 两条消息；5xx/网络错误重试一次
func (c *Client) Chat(ctx context.Context, system, user string, temperature float64) (string, error) {
	if !c.Enabled() {
		return "", fmt.Errorf("未配置 DEEPSEEK_API_KEY")
	}
	if temperature <= 0 {
		temperature = 0.3
	}
	body, _ := json.Marshal(chatRequest{
		Model: c.model,
		Messages: []chatMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: temperature,
	})

	var lastErr error
	for attempt := 0; attempt < 2; attempt++ {
		text, retryable, err := c.do(ctx, body)
		if err == nil {
			return text, nil
		}
		lastErr = err
		if !retryable || ctx.Err() != nil {
			break
		}
		time.Sleep(time.Duration(attempt+1) * 800 * time.Millisecond)
	}
	return "", lastErr
}

func (c *Client) do(ctx context.Context, body []byte) (string, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.hc.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("调用大模型失败: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode >= 500 {
		return "", true, fmt.Errorf("大模型服务异常(%d): %s", resp.StatusCode, truncate(string(raw), 200))
	}
	if resp.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("大模型返回错误(%d): %s", resp.StatusCode, truncate(string(raw), 300))
	}
	var out chatResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return "", false, fmt.Errorf("解析大模型响应失败: %w", err)
	}
	if out.Error != nil {
		return "", false, fmt.Errorf("大模型错误: %s", out.Error.Message)
	}
	if len(out.Choices) == 0 {
		return "", false, fmt.Errorf("大模型返回为空")
	}
	return strings.TrimSpace(out.Choices[0].Message.Content), false, nil
}

// ChatJSON 要求模型输出 JSON 并反序列化到 out。
// DeepSeek 偶尔会带 ```json 围栏或前后解释，这里做容错提取。
func (c *Client) ChatJSON(ctx context.Context, system, user string, out interface{}, temperature float64) error {
	text, err := c.Chat(ctx, system, user, temperature)
	if err != nil {
		return err
	}
	payload := extractJSON(text)
	if payload == "" {
		return fmt.Errorf("大模型未返回合法 JSON: %s", truncate(text, 200))
	}
	if err := json.Unmarshal([]byte(payload), out); err != nil {
		return fmt.Errorf("解析 JSON 失败: %w | 原文: %s", err, truncate(text, 200))
	}
	return nil
}

// extractJSON 从模型输出里截取第一个完整的 JSON 对象/数组
func extractJSON(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if i := strings.Index(s, "\n"); i >= 0 {
			s = s[i+1:]
		}
		if i := strings.LastIndex(s, "```"); i >= 0 {
			s = s[:i]
		}
		s = strings.TrimSpace(s)
	}
	start := strings.IndexAny(s, "{[")
	if start < 0 {
		return ""
	}
	open := s[start]
	closeCh := byte('}')
	if open == '[' {
		closeCh = ']'
	}
	depth := 0
	inStr := false
	escaped := false
	for i := start; i < len(s); i++ {
		ch := s[i]
		if inStr {
			if escaped {
				escaped = false
			} else if ch == '\\' {
				escaped = true
			} else if ch == '"' {
				inStr = false
			}
			continue
		}
		switch ch {
		case '"':
			inStr = true
		case open:
			depth++
		case closeCh:
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
