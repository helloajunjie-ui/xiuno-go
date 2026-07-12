// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

type limitRecord struct {
	count     int
	expiresAt int64
}

// RateLimiter 内存级滑动窗口限流器
// 零外部依赖，纯 Go 原生实现
// 后台 GC 协程每分钟清理过期记录，防止内存泄漏
type RateLimiter struct {
	mu      sync.Mutex
	records map[string]*limitRecord
}

// NewRateLimiter 创建限流器并启动后台 GC
func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		records: make(map[string]*limitRecord),
	}
	go func() {
		for {
			time.Sleep(1 * time.Minute)
			rl.cleanup()
		}
	}()
	return rl
}

func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	now := time.Now().Unix()
	for k, v := range rl.records {
		if now > v.expiresAt {
			delete(rl.records, k)
		}
	}
}

// Allow 判断是否放行
// key: 限流标识符（如 "uid:123" 或 "ip:192.168.1.1"）
// max: 窗口内最大请求数
// windowSec: 窗口大小（秒）
// 返回 true 表示通过
func (rl *RateLimiter) Allow(key string, max int, windowSec int64) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now().Unix()
	record, exists := rl.records[key]

	// 记录不存在或已过期，重置窗口
	if !exists || now > record.expiresAt {
		rl.records[key] = &limitRecord{
			count:     1,
			expiresAt: now + windowSec,
		}
		return true
	}

	// 在窗口期内，判断是否超限
	if record.count < max {
		record.count++
		return true
	}

	return false
}

// GetClientIP 从请求中提取真实客户端 IP
func GetClientIP(r *http.Request) string {
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return strings.TrimSpace(strings.Split(ip, ",")[0])
}

// RateLimitMiddleware 动态限流中间件
// action: 操作名称，用于构造限流 key（如 "register", "login", "thread", "post"）
// max: 窗口内最大请求数
// windowSec: 窗口大小（秒）
//
// 限流策略：
//   - 已登录用户（UID > 0）：按 UID 限流
//   - 游客（UID = 0）：按 IP 限流
func RateLimitMiddleware(app *AppCtx, action string, max int, windowSec int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var identifier string
			claims := GetClaims(r.Context())

			// 优先使用 UID 限流，游客则使用 IP
			if claims != nil && claims.UID > 0 {
				identifier = fmt.Sprintf("uid:%d", claims.UID)
			} else {
				identifier = fmt.Sprintf("ip:%s", GetClientIP(r))
			}

			key := fmt.Sprintf("%s:%s", action, identifier)

			if !app.RateLimiter.Allow(key, max, windowSec) {
				JSONError(w, http.StatusTooManyRequests, "操作过于频繁，请冷静一下")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
