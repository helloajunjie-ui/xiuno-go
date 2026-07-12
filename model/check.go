// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"net/mail"
	"regexp"
	"strings"
	"unicode/utf8"
)

// 编译一次正则，避免每次调用都编译
var (
	// 允许的字符：字母、数字、中文、下划线、横线
	reUsername = regexp.MustCompile(`^[\p{L}\p{N}_\-]+$`)
	// 简单手机号校验（中国大陆 1xx 开头 11 位）
	reMobile = regexp.MustCompile(`^1[0-9]{10}$`)
)

// IsWord 检查字符串是否只包含字母、数字、下划线、中文
// 对应 PHP: is_word()
func IsWord(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if !isWordRune(r) {
			return false
		}
	}
	return true
}

func isWordRune(r rune) bool {
	// 字母、数字、下划线、横线
	if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9') || r == '_' || r == '-' {
		return true
	}
	// 中文字符范围
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	// 日文假名
	if r >= 0x3040 && r <= 0x30FF {
		return true
	}
	return false
}

// IsMobile 校验手机号格式
// 对应 PHP: is_mobile()
func IsMobile(mobile string) bool {
	return reMobile.MatchString(mobile)
}

// IsEmail 校验邮箱格式
// 对应 PHP: is_email()
// 使用 Go 标准库 mail.ParseAddress，比 PHP 的 filter_var 更严格
func IsEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	_, err := mail.ParseAddress(email)
	return err == nil
}

// IsUsername 校验用户名
// 规则：2-15 个字符，只能包含字母、数字、中文、下划线、横线
// 对应 PHP: is_username()
func IsUsername(username string) bool {
	length := utf8.RuneCountInString(username)
	if length < 2 || length > 15 {
		return false
	}
	return reUsername.MatchString(username)
}

// IsPassword 校验密码强度
// 规则：最少 6 个字符，最多 256 个字符
// 对应 PHP: is_password()
// 注意：Go 版不做字符类型校验（大小写+数字），只做长度校验，
// 因为 bcrypt 可以处理任意字节序列，让用户自由选择密码复杂度
func IsPassword(password string) bool {
	length := utf8.RuneCountInString(password)
	if length < 6 || length > 256 {
		return false
	}
	// 不允许纯空白密码
	if strings.TrimSpace(password) == "" {
		return false
	}
	return true
}

// SanitizeUsername 清理用户名中的危险字符（注入防护）
func SanitizeUsername(username string) string {
	// 只保留允许的字符
	var b strings.Builder
	for _, r := range username {
		if isWordRune(r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}
