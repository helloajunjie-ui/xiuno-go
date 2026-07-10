package model

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// Cron 计划任务管理器
// 对应 PHP: model/cron.func.php
// 在 AppCtx 启动时作为后台协程运行
type Cron struct {
	db        *sqlx.DB
	cache     core.Cache
	uploadDir string
	mu        sync.Mutex
	stop      chan struct{}
}

// NewCron 创建计划任务管理器并启动
func NewCron(db *sqlx.DB, cache core.Cache, uploadDir string) *Cron {
	c := &Cron{
		db:        db,
		cache:     cache,
		uploadDir: uploadDir,
		stop:      make(chan struct{}),
	}
	go c.loop()
	return c
}

// Close 停止计划任务
func (c *Cron) Close() {
	close(c.stop)
}

func (c *Cron) loop() {
	// 启动后立即执行一次
	c.run()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.run()
		case <-c.stop:
			return
		}
	}
}

func (c *Cron) run() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now().Unix()

	// 从 runtime cache 读取上次执行时间
	ctx := context.Background()
	rt, err := GetRuntime(ctx, c.db, c.cache)
	if err != nil {
		log.Printf("[Cron] 获取 runtime 失败: %v", err)
		return
	}

	// === 每 5 分钟任务 ===
	if now-rt.Cron1LastDate > 300 {
		if !c.acquireLock("cron_lock_1", 10) {
			return
		}
		defer c.releaseLock("cron_lock_1")

		// 更新在线人数（简化：固定为 1）
		rt.Onlines = 1

		rt.Cron1LastDate = now
		_ = SaveRuntime(ctx, c.cache, rt)
		ResetRuntimeCache()

		log.Println("[Cron] 5分钟任务完成")
	}

	// === 每日 0 点任务 ===
	if now-rt.Cron2LastDate > 86400 {
		if !c.acquireLock("cron_lock_2", 10) {
			return
		}
		defer c.releaseLock("cron_lock_2")

		// 今日统计清零
		rt.TodayPosts = 0
		rt.TodayThreads = 0
		rt.TodayUsers = 0

		// 版块今日统计清零
		if err := c.resetForumTodayStats(ctx); err != nil {
			log.Printf("[Cron] 版块今日统计清零失败: %v", err)
		}

		// 清理过期的队列数据
		if err := QueueGC(ctx, c.db); err != nil {
			log.Printf("[Cron] 队列清理失败: %v", err)
		}

		// 每日最大ID统计（前一天的数据）
		yesterday := time.Now().AddDate(0, 0, -1).Truncate(24 * time.Hour).Unix()
		if err := TableDayCron(ctx, c.db, yesterday); err != nil {
			log.Printf("[Cron] 每日最大ID统计失败: %v", err)
		}

		// 附件垃圾回收（清理过期临时文件和孤儿附件记录）
		AttachGC(ctx, c.db, c.uploadDir)

		// 更新 cron_2_last_date 为今天的 0 点
		today := time.Now().Truncate(24 * time.Hour).Unix()
		rt.Cron2LastDate = today

		_ = SaveRuntime(ctx, c.cache, rt)
		ResetRuntimeCache()

		log.Println("[Cron] 每日任务完成")
	}
}

// acquireLock 获取 cron 锁（防止多实例并发执行）
func (c *Cron) acquireLock(key string, ttlSec int) bool {
	_, ok := c.cache.Get(context.Background(), key)
	if ok {
		return false
	}
	c.cache.Set(context.Background(), key, []byte("1"), time.Duration(ttlSec)*time.Second)
	return true
}

// releaseLock 释放 cron 锁
func (c *Cron) releaseLock(key string) {
	c.cache.Del(context.Background(), key)
}

// resetForumTodayStats 将所有版块的今日统计清零
func (c *Cron) resetForumTodayStats(ctx context.Context) error {
	_, err := c.db.ExecContext(ctx,
		`UPDATE bbs_forum SET todayposts = 0, todaythreads = 0`)
	if err != nil {
		return fmt.Errorf("resetForumTodayStats: %w", err)
	}
	return nil
}
