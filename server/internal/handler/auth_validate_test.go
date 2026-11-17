package handler

import "testing"

// 手机号格式与密码强度校验（注册/登录/绑定的统一入口）
func TestValidPhone(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{
		{"13812345678", true},
		{"19912345678", true},
		{"12345678901", false},  // 第二位 2
		{"23812345678", false},  // 不以 1 开头
		{"1381234567", false},   // 10 位
		{"138123456789", false}, // 12 位
		{"1381234567a", false},  // 含字母
		{"", false},
		{"1381234 5678", false},
	}
	for _, tc := range cases {
		if got := validPhone(tc.in); got != tc.want {
			t.Errorf("validPhone(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestValidPassword(t *testing.T) {
	if !validPassword("12345678") {
		t.Error("8 位密码应通过")
	}
	if validPassword("1234567") {
		t.Error("7 位密码应拒绝")
	}
	if validPassword("") {
		t.Error("空密码应拒绝")
	}
}
