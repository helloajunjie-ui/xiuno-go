package core

import (
	"context"
	"sync"
	"time"
)

// Cache 缓存接口，默认实现为内存，可切换 Redis
// 设计目的：不强制依赖 Redis，单文件部署时用内存缓存即可运行
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool)
	Set(ctx context.Context, key string, val []byte, ttl time.Duration)
	Del(ctx context.Context, key string)
	Close() error
}

// memoryCache 内存缓存实现
type memoryCache struct {
	mu   sync.RWMutex
	data map[string]cacheItem
}

type cacheItem struct {
	val      []byte
	expireAt time.Time
}

func newMemoryCache() *memoryCache {
	c := &memoryCache{
		data: make(map[string]cacheItem),
	}
	// 后台协程定期清理过期 key
	go c.evictLoop()
	return c
}

func (c *memoryCache) Get(_ context.Context, key string) ([]byte, bool) {
	c.mu.RLock()
	item, ok := c.data[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !item.expireAt.IsZero() && time.Now().After(item.expireAt) {
		c.mu.Lock()
		delete(c.data, key)
		c.mu.Unlock()
		return nil, false
	}
	return item.val, true
}

func (c *memoryCache) Set(_ context.Context, key string, val []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	item := cacheItem{val: val}
	if ttl > 0 {
		item.expireAt = time.Now().Add(ttl)
	}
	c.data[key] = item
}

func (c *memoryCache) Del(_ context.Context, key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
}

func (c *memoryCache) Close() error { return nil }

func (c *memoryCache) evictLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, v := range c.data {
			if !v.expireAt.IsZero() && now.After(v.expireAt) {
				delete(c.data, k)
			}
		}
		c.mu.Unlock()
	}
}

// NewCache 根据配置创建缓存实例
func NewCache(cfg *Config) Cache {
	switch cfg.Cache.Driver {
	case "redis":
		// Redis 实现暂未引入依赖，后续按需添加
		return newMemoryCache()
	default:
		return newMemoryCache()
	}
}
