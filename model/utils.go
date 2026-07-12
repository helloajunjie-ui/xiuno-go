// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"fmt"
	"net"
	"time"
)

// IP2Long 将 net.IP 转换为 uint32（MySQL INET_ATON() 的 Go 实现）
// 仅支持 IPv4，非 IPv4 返回 0
func IP2Long(ip net.IP) uint32 {
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0
	}
	return uint32(ipv4[0])<<24 | uint32(ipv4[1])<<16 | uint32(ipv4[2])<<8 | uint32(ipv4[3])
}

// modelLong2IP 将 uint32 格式的 IP 地址转换为点分十进制字符串
// MySQL INET_NTOA() 的 Go 实现
// 输入: 2130706433 → 输出: "127.0.0.1"
func modelLong2IP(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}

// toUint32 将 interface{} 安全转换为 uint32
// 用于 ThreadFind 等通用查询函数的条件参数转换
func toUint32(v interface{}) uint32 {
	switch val := v.(type) {
	case uint32:
		return val
	case int:
		return uint32(val)
	case int64:
		return uint32(val)
	case float64:
		return uint32(val)
	default:
		return 0
	}
}

// humandate 将 Unix 时间戳格式化为人类可读的日期字符串
// 对应 PHP: humandate() — Xiuno 的时间格式化函数
// 规则：
//   - 今天: 显示 "HH:MM"
//   - 昨天: 显示 "昨天 HH:MM"
//   - 今年: 显示 "MM-DD HH:MM"
//   - 更早: 显示 "YYYY-MM-DD HH:MM"
func humandate(ts int64) string {
	if ts == 0 {
		return ""
	}
	t := time.Unix(ts, 0)
	now := time.Now()
	loc := t.Location()

	// 计算今天的起始时间戳
	year, month, day := now.Date()
	todayStart := time.Date(year, month, day, 0, 0, 0, 0, loc)
	// 计算昨天的起始时间戳
	yesterdayStart := todayStart.AddDate(0, 0, -1)

	if t.After(todayStart) || t.Equal(todayStart) {
		// 今天: HH:MM
		return fmt.Sprintf("%02d:%02d", t.Hour(), t.Minute())
	} else if t.After(yesterdayStart) || t.Equal(yesterdayStart) {
		// 昨天: 昨天 HH:MM
		return fmt.Sprintf("昨天 %02d:%02d", t.Hour(), t.Minute())
	} else if t.Year() == now.Year() {
		// 今年: MM-DD HH:MM
		return fmt.Sprintf("%02d-%02d %02d:%02d", t.Month(), t.Day(), t.Hour(), t.Minute())
	}
	// 更早: YYYY-MM-DD HH:MM
	return fmt.Sprintf("%d-%02d-%02d %02d:%02d", t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute())
}

// dateYMD 将 Unix 时间戳格式化为 YYYY-M-D（PHP date('Y-n-j') 兼容）
func dateYMD(ts int64) string {
	if ts == 0 {
		return "0000-00-00"
	}
	t := time.Unix(ts, 0)
	return fmt.Sprintf("%d-%d-%d", t.Year(), t.Month(), t.Day())
}
