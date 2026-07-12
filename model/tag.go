// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// Tag 标签
type Tag struct {
	TagID      uint32 `db:"tagid" json:"tagid"`
	Name       string `db:"name" json:"name"`
	Threads    uint32 `db:"threads" json:"threads"`
	CreateDate uint32 `db:"create_date" json:"create_date"`
}

// TagThreadItem 标签关联的主题（列表用）
type TagThreadItem struct {
	TID        uint32 `db:"tid" json:"tid"`
	FID        uint32 `db:"fid" json:"fid"`
	Subject    string `db:"subject" json:"subject"`
	Username   string `db:"username" json:"username"`
	CreateDate uint32 `db:"create_date" json:"create_date"`
	LastDate   uint32 `db:"last_date" json:"last_date"`
	Views      uint32 `db:"views" json:"views"`
	Posts      uint32 `db:"posts" json:"posts"`
}

// TagList 获取标签列表（按主题数降序）
func TagList(ctx context.Context, db *sqlx.DB, page, pageSize int) ([]Tag, error) {
	offset := (page - 1) * pageSize
	var tags []Tag
	err := db.SelectContext(ctx, &tags,
		"SELECT * FROM bbs_tag ORDER BY threads DESC, tagid ASC LIMIT ? OFFSET ?",
		pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("查询标签列表失败: %w", err)
	}
	return tags, nil
}

// TagCount 标签总数
func TagCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, "SELECT COUNT(*) FROM bbs_tag")
	if err != nil {
		return 0, fmt.Errorf("统计标签数失败: %w", err)
	}
	return count, nil
}

// TagRead 读取单个标签
func TagRead(ctx context.Context, db *sqlx.DB, tagid uint32) (*Tag, error) {
	var tag Tag
	err := db.GetContext(ctx, &tag, "SELECT * FROM bbs_tag WHERE tagid=?", tagid)
	if err != nil {
		return nil, fmt.Errorf("读取标签失败: %w", err)
	}
	return &tag, nil
}

// TagReadByName 按名称读取标签
func TagReadByName(ctx context.Context, db *sqlx.DB, name string) (*Tag, error) {
	var tag Tag
	err := db.GetContext(ctx, &tag, "SELECT * FROM bbs_tag WHERE name=?", name)
	if err != nil {
		return nil, fmt.Errorf("按名称读取标签失败: %w", err)
	}
	return &tag, nil
}

// TagCreate 创建标签（幂等：已存在则返回现有 tagid）
func TagCreate(ctx context.Context, db *sqlx.DB, name string) (uint32, error) {
	// 先查是否存在
	existing, err := TagReadByName(ctx, db, name)
	if err == nil && existing != nil {
		return existing.TagID, nil
	}

	now := uint32(time.Now().Unix())
	result, err := db.ExecContext(ctx,
		"INSERT INTO bbs_tag (`name`, `threads`, `create_date`) VALUES (?, 0, ?)",
		name, now)
	if err != nil {
		// 唯一键冲突（并发情况）
		existing, err2 := TagReadByName(ctx, db, name)
		if err2 == nil && existing != nil {
			return existing.TagID, nil
		}
		return 0, fmt.Errorf("创建标签失败: %w", err)
	}
	id, _ := result.LastInsertId()
	return uint32(id), nil
}

// TagCreateOrGet 创建或获取标签，返回 tagid
func TagCreateOrGet(ctx context.Context, db *sqlx.DB, name string) (uint32, error) {
	return TagCreate(ctx, db, name)
}

// TagFindByTID 获取主题关联的所有标签
func TagFindByTID(ctx context.Context, db *sqlx.DB, tid uint32) ([]Tag, error) {
	var tags []Tag
	err := db.SelectContext(ctx, &tags,
		`SELECT t.* FROM bbs_tag t
		 INNER JOIN bbs_thread_tag tt ON t.tagid = tt.tagid
		 WHERE tt.tid = ?
		 ORDER BY t.threads DESC`, tid)
	if err != nil {
		return nil, fmt.Errorf("查询主题标签失败: %w", err)
	}
	return tags, nil
}

// TagFindThreads 获取标签下的主题列表
func TagFindThreads(ctx context.Context, db *sqlx.DB, tagid uint32, page, pageSize int) ([]TagThreadItem, error) {
	offset := (page - 1) * pageSize
	var items []TagThreadItem
	err := db.SelectContext(ctx, &items,
		`SELECT th.tid, th.fid, th.subject, u.username, th.create_date, th.last_date, th.views, th.posts
		 FROM bbs_thread th
		 INNER JOIN bbs_thread_tag tt ON th.tid = tt.tid
		 LEFT JOIN bbs_user u ON th.uid = u.uid
		 WHERE tt.tagid = ? AND th.deleted_at IS NULL
		 ORDER BY th.last_date DESC
		 LIMIT ? OFFSET ?`, tagid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("查询标签主题列表失败: %w", err)
	}
	return items, nil
}

// TagThreadCount 标签下的主题数
func TagThreadCount(ctx context.Context, db *sqlx.DB, tagid uint32) (int, error) {
	var count int
	err := db.GetContext(ctx, &count,
		"SELECT COUNT(*) FROM bbs_thread_tag WHERE tagid=?", tagid)
	if err != nil {
		return 0, fmt.Errorf("统计标签主题数失败: %w", err)
	}
	return count, nil
}

// TagSetThreadTags 设置主题的标签（先删后加）
// tags 是以逗号分隔的标签名称字符串，如 "golang,redis,api"
func TagSetThreadTags(ctx context.Context, db *sqlx.DB, tid uint32, tags string) error {
	// 先删除旧关联
	if _, err := db.ExecContext(ctx, "DELETE FROM bbs_thread_tag WHERE tid=?", tid); err != nil {
		return fmt.Errorf("删除旧标签关联失败: %w", err)
	}

	// 解析标签名
	names := parseTagNames(tags)
	if len(names) == 0 {
		return nil
	}

	// 创建或获取标签，建立关联
	for _, name := range names {
		tagid, err := TagCreateOrGet(ctx, db, name)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx,
			"INSERT IGNORE INTO bbs_thread_tag (tid, tagid) VALUES (?, ?)", tid, tagid); err != nil {
			return fmt.Errorf("关联标签失败: %w", err)
		}
	}

	// 更新标签的主题计数
	if _, err := db.ExecContext(ctx,
		`UPDATE bbs_tag SET threads = (
			SELECT COUNT(*) FROM bbs_thread_tag WHERE tagid = bbs_tag.tagid
		)`); err != nil {
		return fmt.Errorf("更新标签计数失败: %w", err)
	}

	return nil
}

// parseTagNames 解析逗号分隔的标签名，去重、去空、截断
func parseTagNames(s string) []string {
	parts := strings.Split(s, ",")
	seen := make(map[string]bool)
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" || len(p) > 32 {
			continue
		}
		if seen[p] {
			continue
		}
		seen[p] = true
		result = append(result, p)
	}
	return result
}
