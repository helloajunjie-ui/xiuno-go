package core

import (
	"log"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
)

// AsyncCounter 极简内存异步计数器
// 将并发 +1 操作合并为批量 UPDATE，消除 InnoDB 行锁热点
// 设计原则：不引入外部依赖（Kafka/Redis），纯 Go 原生实现
type AsyncCounter struct {
	db           *sqlx.DB
	mu           sync.Mutex
	forumThreads map[uint32]int // fid -> +N
	userThreads  map[uint32]int // uid -> +N
	userPosts    map[uint32]int // uid -> +N
	threadViews  map[uint32]int // tid -> +N
	threadPosts  map[uint32]int // tid -> +N 回复数
	closeCh      chan struct{}  // 关闭信号，通知后台协程退出
}

// NewAsyncCounter 创建异步计数器并启动后台刷入协程
func NewAsyncCounter(db *sqlx.DB) *AsyncCounter {
	c := &AsyncCounter{
		db:           db,
		forumThreads: make(map[uint32]int),
		userThreads:  make(map[uint32]int),
		userPosts:    make(map[uint32]int),
		threadViews:  make(map[uint32]int),
		threadPosts:  make(map[uint32]int),
		closeCh:      make(chan struct{}),
	}
	go c.startFlusher()
	return c
}

// IncrForumThread 版块主题数 +1（非阻塞）
func (c *AsyncCounter) IncrForumThread(fid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forumThreads[fid]++
}

// IncrUserThread 用户发帖数 +1（非阻塞）
func (c *AsyncCounter) IncrUserThread(uid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userThreads[uid]++
}

// IncrUserPost 用户回帖数 +1（非阻塞）
func (c *AsyncCounter) IncrUserPost(uid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userPosts[uid]++
}

// IncrThreadView 帖子浏览数 +1（非阻塞）
func (c *AsyncCounter) IncrThreadView(tid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.threadViews[tid]++
}

// IncrThreadPost 帖子回复数 +1（非阻塞）
func (c *AsyncCounter) IncrThreadPost(tid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.threadPosts[tid]++
}

// DecrForumThread 版块主题数 -1（非阻塞，用于软删除）
func (c *AsyncCounter) DecrForumThread(fid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.forumThreads[fid]--
}

// DecrUserThread 用户发帖数 -1（非阻塞，用于软删除）
func (c *AsyncCounter) DecrUserThread(uid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userThreads[uid]--
}

// DecrThreadPost 帖子回复数 -1（非阻塞，用于软删除回帖）
func (c *AsyncCounter) DecrThreadPost(tid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.threadPosts[tid]--
}

// DecrUserPost 用户回帖数 -1（非阻塞，用于软删除回帖）
func (c *AsyncCounter) DecrUserPost(uid uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.userPosts[uid]--
}

// startFlusher 后台定时刷入 DB
func (c *AsyncCounter) startFlusher() {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			c.flush()
		case <-c.closeCh:
			return
		}
	}
}

func (c *AsyncCounter) flush() {
	c.mu.Lock()
	ft := c.forumThreads
	ut := c.userThreads
	up := c.userPosts
	tv := c.threadViews
	tp := c.threadPosts
	c.forumThreads = make(map[uint32]int)
	c.userThreads = make(map[uint32]int)
	c.userPosts = make(map[uint32]int)
	c.threadViews = make(map[uint32]int)
	c.threadPosts = make(map[uint32]int)
	c.mu.Unlock()

	// 批量更新版块统计（支持正负 delta，GREATEST 防止 unsigned 字段跌破 0）
	for fid, delta := range ft {
		if delta != 0 {
			_, err := c.db.Exec(
				`UPDATE bbs_forum SET threads = GREATEST(CAST(threads AS SIGNED) + ?, 0), todaythreads = GREATEST(CAST(todaythreads AS SIGNED) + ?, 0) WHERE fid = ?`,
				delta, delta, fid)
			if err != nil {
				log.Printf("[Counter] 版块统计同步失败 fid=%d: %v", fid, err)
			}
		}
	}

	// 批量更新用户发帖统计
	for uid, delta := range ut {
		if delta != 0 {
			_, err := c.db.Exec(
				`UPDATE bbs_user SET threads = GREATEST(CAST(threads AS SIGNED) + ?, 0) WHERE uid = ?`,
				delta, uid)
			if err != nil {
				log.Printf("[Counter] 用户发帖统计同步失败 uid=%d: %v", uid, err)
			}
		}
	}

	// 批量更新用户回帖统计
	for uid, delta := range up {
		if delta != 0 {
			_, err := c.db.Exec(
				`UPDATE bbs_user SET posts = GREATEST(CAST(posts AS SIGNED) + ?, 0) WHERE uid = ?`,
				delta, uid)
			if err != nil {
				log.Printf("[Counter] 用户回帖统计同步失败 uid=%d: %v", uid, err)
			}
		}
	}

	// 批量更新帖子浏览数
	for tid, delta := range tv {
		if delta != 0 {
			_, err := c.db.Exec(
				`UPDATE bbs_thread SET views = GREATEST(CAST(views AS SIGNED) + ?, 0) WHERE tid = ?`,
				delta, tid)
			if err != nil {
				log.Printf("[Counter] 帖子浏览同步失败 tid=%d: %v", tid, err)
			}
		}
	}

	// 批量更新帖子回复数
	for tid, delta := range tp {
		if delta != 0 {
			_, err := c.db.Exec(
				`UPDATE bbs_thread SET posts = GREATEST(CAST(posts AS SIGNED) + ?, 0) WHERE tid = ?`,
				delta, tid)
			if err != nil {
				log.Printf("[Counter] 帖子回复同步失败 tid=%d: %v", tid, err)
			}
		}
	}
}

// Close 停止后台协程（等待最后一次 flush）
func (c *AsyncCounter) Close() {
	close(c.closeCh)
	c.flush()
}
