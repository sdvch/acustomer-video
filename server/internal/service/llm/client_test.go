package llm

import "testing"

// 文本大模型只用 DeepSeek（无备选），client 只依赖 DEEPSEEK_* 配置。
// 这里覆盖纯函数：extractJSON 容错提取与 truncate。

func TestExtractJSON(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"裸对象", `{"a":1}`, `{"a":1}`},
		{"裸数组", `[1,2,3]`, `[1,2,3]`},
		{"json围栏", "```json\n{\"a\":1}\n```", `{"a":1}`},
		{"围栏无语言标注", "```\n{\"a\":1}\n```", `{"a":1}`},
		{"前后带解释文字", `好的，结果如下：\n{"hooks":[{"text":"x"}]}\n希望有帮助`, `{"hooks":[{"text":"x"}]}`},
		{"嵌套对象", `{"a":{"b":[1,{"c":2}]}}`, `{"a":{"b":[1,{"c":2}]}}`},
		{"字符串内的花括号", `{"s":"包含 } 和 { 的文本"}`, `{"s":"包含 } 和 { 的文本"}`},
		{"字符串内的转义引号", `{"s":"说\"你好\"}"}`, `{"s":"说\"你好\"}"}`},
		{"数组开头的嵌套", `前缀 [{"x":1},{"y":[2]}] 后缀`, `[{"x":1},{"y":[2]}]`},
		{"无JSON返回空", `这里没有 JSON`, ``},
		{"只有围栏无内容", "```json\n```", ``},
		{"截断的JSON返回空", `{"a":`, ``},
		{"先出现数组括号", `说明 [1] 与 {"a":2} 并存`, `[1]`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := extractJSON(tc.in); got != tc.want {
				t.Errorf("extractJSON(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("短文本", 10); got != "短文本" {
		t.Errorf("短文本不应截断: %q", got)
	}
	if got := truncate("0123456789ABC", 10); got != "0123456789..." {
		t.Errorf("长文本应截断加省略号: %q", got)
	}
}

func TestClientEnabledNil(t *testing.T) {
	var c *Client
	if c.Enabled() {
		t.Error("nil 客户端不应可用")
	}
}
