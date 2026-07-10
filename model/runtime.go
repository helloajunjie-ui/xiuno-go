package model

import (
	"context"
	"fmt"
	"sync"
	"time"

	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// Runtime 运行时统计
// 对应 PHP: model/runtime.func.php
// 存储在 core.Cache 中，key = "runtime"
type Runtime struct {
	Users         int64 `json:"users"`            // 用户总数
	Threads       int64 `json:"threads"`          // 主题总数
	Posts         int64 `json:"posts"`            // 回帖总数（不含首帖）
	TodayUsers    int64 `json:"todayusers"`       // 今日登录用户数
	TodayPosts    int64 `json:"todayposts"`       // 今日发帖数
	TodayThreads  int64 `json:"todaythreads"`     // 今日主题数
	Onlines       int64 `json:"onlines"`          // 在线人数
	Cron1LastDate int64 `json:"cron_1_last_date"` // 5分钟 cron 最后执行时间
	Cron2LastDate int64 `json:"cron_2_last_date"` // 每日 cron 最后执行时间
}

var (
	runtimeCache     *Runtime
	runtimeCacheMu   sync.RWMutex
	runtimeCacheTime time.Time
	runtimeCacheTTL  = 30 * time.Second // 缓存有效期，避免每次请求都查库
)

// GetRuntime 获取运行时统计（带内存缓存）
// 缓存过期后重新从 DB 聚合查询
func GetRuntime(ctx context.Context, db *sqlx.DB, cache core.Cache) (*Runtime, error) {
	runtimeCacheMu.RLock()
	if runtimeCache != nil && time.Since(runtimeCacheTime) < runtimeCacheTTL {
		r := runtimeCache
		runtimeCacheMu.RUnlock()
		return r, nil
	}
	runtimeCacheMu.RUnlock()

	runtimeCacheMu.Lock()
	defer runtimeCacheMu.Unlock()

	// 双重检查
	if runtimeCache != nil && time.Since(runtimeCacheTime) < runtimeCacheTTL {
		return runtimeCache, nil
	}

	// 1. 尝试从 core.Cache 读取（持久化存储）
	rt := &Runtime{}
	data, ok := cache.Get(context.Background(), "runtime")
	if ok && len(data) > 0 {
		// 从 cache 恢复（简化：只恢复关键字段，在线人数实时计算）
		// 这里不反序列化，直接重新聚合
	}

	// 2. 聚合查询
	var err error
	rt.Users, err = getUserCount(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("GetRuntime users: %w", err)
	}

	rt.Threads, err = getThreadCount(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("GetRuntime threads: %w", err)
	}

	totalPosts, err := getPostCount(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("GetRuntime posts: %w", err)
	}
	rt.Posts = totalPosts - rt.Threads // 减去首帖

	// 今日统计
	rt.TodayThreads, err = getTodayThreadCount(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("GetRuntime todaythreads: %w", err)
	}
	rt.TodayPosts, err = getTodayPostCount(ctx, db)
	if err != nil {
		return nil, fmt.Errorf("GetRuntime todayposts: %w", err)
	}

	// 在线人数（简化：返回 1 避免显示 0）
	rt.Onlines = 1

	runtimeCache = rt
	runtimeCacheTime = time.Now()

	return rt, nil
}

// SaveRuntime 保存运行时统计到 cache
func SaveRuntime(ctx context.Context, cache core.Cache, rt *Runtime) error {
	// 序列化为 JSON 存入 cache
	data := []byte(fmt.Sprintf(`{"users":%d,"threads":%d,"posts":%d,"todayusers":%d,"todayposts":%d,"todaythreads":%d,"onlines":%d,"cron_1_last_date":%d,"cron_2_last_date":%d}`,
		rt.Users, rt.Threads, rt.Posts, rt.TodayUsers, rt.TodayPosts, rt.TodayThreads, rt.Onlines, rt.Cron1LastDate, rt.Cron2LastDate))
	cache.Set(ctx, "runtime", data, 0)
	return nil
}

// ResetRuntimeCache 重置运行时内存缓存（cron 调用）
func ResetRuntimeCache() {
	runtimeCacheMu.Lock()
	defer runtimeCacheMu.Unlock()
	runtimeCache = nil
}

// --- 内部聚合查询 ---

func getUserCount(ctx context.Context, db *sqlx.DB) (int64, error) {
	var count int64
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_user`)
	return count, err
}

func getThreadCount(ctx context.Context, db *sqlx.DB) (int64, error) {
	var count int64
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_thread WHERE deleted_at IS NULL`)
	return count, err
}

// RuntimeSet 设置运行时键值
// 对应 PHP: runtime_set($k, $v)
// 存储到 core.Cache 中，key = "runtime_"+k
func RuntimeSet(ctx context.Context, cache core.Cache, k string, v string) error {
	cache.Set(ctx, "runtime_"+k, []byte(v), 0)
	return nil
}

// RuntimeDelete 删除运行时键值
// 对应 PHP: runtime_delete($k)
func RuntimeDelete(ctx context.Context, cache core.Cache, k string) error {
	cache.Del(ctx, "runtime_"+k)
	return nil
}

// RuntimeTruncate 清空所有运行时数据
// 对应 PHP: runtime_truncate()
func RuntimeTruncate(ctx context.Context, cache core.Cache) error {
	// 清除已知的 runtime key
	keys := []string{"runtime", "runtime_users", "runtime_threads", "runtime_posts",
		"runtime_todayusers", "runtime_todayposts", "runtime_todaythreads",
		"runtime_onlines", "runtime_cron_1_last_date", "runtime_cron_2_last_date"}
	for _, k := range keys {
		cache.Del(ctx, k)
	}
	// 重置内存缓存
	ResetRuntimeCache()
	return nil
}

func getPostCount(ctx context.Context, db *sqlx.DB) (int64, error) {
	var count int64
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_post WHERE deleted_at IS NULL`)
	return count, err
}

func getTodayThreadCount(ctx context.Context, db *sqlx.DB) (int64, error) {
	today := time.Now().Unix() - int64(time.Now().Hour())*3600 - int64(time.Now().Minute())*60
	var count int64
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_thread WHERE create_date >= ? AND deleted_at IS NULL`, today)
	return count, err
}

func getTodayPostCount(ctx context.Context, db *sqlx.DB) (int64, error) {
	today := time.Now().Unix() - int64(time.Now().Hour())*3600 - int64(time.Now().Minute())*60
	var count int64
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_post WHERE create_date >= ? AND deleted_at IS NULL`, today)
	return count, err
}
