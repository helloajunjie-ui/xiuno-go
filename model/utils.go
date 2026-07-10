package model

import (
	"fmt"
	"time"
)

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
