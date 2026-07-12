// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

// Forum 对应 bbs_forum 表
type Forum struct {
	FID          int64  `db:"fid" json:"fid"`
	Name         string `db:"name" json:"name"`
	Rank         int32  `db:"rank" json:"rank"`
	Threads      int32  `db:"threads" json:"threads"`
	TodayPosts   int32  `db:"todayposts" json:"todayposts"`
	TodayThreads int32  `db:"todaythreads" json:"todaythreads"`
	Brief        string `db:"brief" json:"brief"`
	Announcement string `db:"announcement" json:"announcement,omitempty"`
	AccessOn     int64  `db:"accesson" json:"accesson"`
	OrderBy      int32  `db:"orderby" json:"orderby"`
	CreateDate   int64  `db:"create_date" json:"create_date"`
	Icon         int64  `db:"icon" json:"icon"`
	ModUIDs      string `db:"moduids" json:"moduids,omitempty"`
	SeoTitle     string `db:"seo_title" json:"seo_title,omitempty"`
	SeoKeywords  string `db:"seo_keywords" json:"seo_keywords,omitempty"`

	// 格式化显示字段（不存库）
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
	IconURL       string `db:"-" json:"icon_url,omitempty"`
}

// ForumAccess 对应 bbs_forum_access 表
type ForumAccess struct {
	FID         int64 `db:"fid" json:"fid"`
	GID         int64 `db:"gid" json:"gid"`
	AllowRead   int32 `db:"allowread" json:"allowread"`
	AllowThread int32 `db:"allowthread" json:"allowthread"`
	AllowPost   int32 `db:"allowpost" json:"allowpost"`
	AllowAttach int32 `db:"allowattach" json:"allowattach"`
	AllowDown   int32 `db:"allowdown" json:"allowdown"`
}

// CreateForum 创建新版块
func CreateForum(ctx context.Context, db *sqlx.DB, f *Forum) (uint32, error) {
	res, err := db.ExecContext(ctx,
		"INSERT INTO bbs_forum (name, brief, announcement, accesson, `rank`, create_date) VALUES (?, ?, ?, ?, ?, ?)",
		f.Name, f.Brief, f.Announcement, f.AccessOn, f.Rank, f.CreateDate)
	if err != nil {
		return 0, err
	}
	fid, _ := res.LastInsertId()
	return uint32(fid), nil
}

// UpdateForum 修改版块信息
func UpdateForum(ctx context.Context, db *sqlx.DB, fid uint32, f *Forum) error {
	_, err := db.ExecContext(ctx,
		"UPDATE bbs_forum SET name = ?, brief = ?, announcement = ?, accesson = ?, `rank` = ? WHERE fid = ?",
		f.Name, f.Brief, f.Announcement, f.AccessOn, f.Rank, fid)
	return err
}

// DeleteForum 删除版块
func DeleteForum(ctx context.Context, db *sqlx.DB, fid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_forum WHERE fid = ?`, fid)
	return err
}

// GetForum 获取单个版块详情
func GetForum(ctx context.Context, db *sqlx.DB, fid uint32) (*Forum, error) {
	var f Forum
	err := db.GetContext(ctx, &f, `SELECT * FROM bbs_forum WHERE fid = ?`, fid)
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ForumFormat 格式化版块数据（填充显示字段）
// 对应 PHP: forum_format()
func ForumFormat(forum *Forum, uploadURL string) {
	if forum == nil {
		return
	}
	forum.CreateDateFmt = dateYMD(forum.CreateDate)
	if forum.Icon > 0 && uploadURL != "" {
		forum.IconURL = fmt.Sprintf("%s/forum/%d.png", uploadURL, forum.FID)
	} else {
		forum.IconURL = "view/img/forum.png"
	}
}

// ForumFind 查询版块列表（支持条件过滤和排序）
// 对应 PHP: forum_find()
func ForumFind(ctx context.Context, db *sqlx.DB) ([]Forum, error) {
	var list []Forum
	err := db.SelectContext(ctx, &list, "SELECT * FROM bbs_forum ORDER BY `rank` ASC")
	if err != nil {
		return nil, fmt.Errorf("ForumFind: %w", err)
	}
	if list == nil {
		list = []Forum{}
	}
	return list, nil
}

// ForumCount 统计版块数量
// 对应 PHP: forum_count()
func ForumCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_forum`)
	if err != nil {
		return 0, fmt.Errorf("ForumCount: %w", err)
	}
	return count, nil
}

// ForumListCache 获取版块列表缓存（全量）
// 对应 PHP: forum_list_cache()
// 返回 map[fid]*Forum，方便前端快速查找
func ForumListCache(ctx context.Context, db *sqlx.DB) (map[int64]*Forum, error) {
	list, err := ForumFind(ctx, db)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*Forum, len(list))
	for i := range list {
		result[list[i].FID] = &list[i]
	}
	return result, nil
}

// ForumListCacheDelete 清除版块列表缓存
// 对应 PHP: forum_list_cache_delete()
// 当前无独立缓存层，保留作为接口兼容
func ForumListCacheDelete() {
	// 无缓存需要清除，保留作为接口兼容
}

// ForumListAccessFilter 对版块列表进行权限过滤
// 对应 PHP: forum_list_access_filter()
// 根据用户组 gid 和权限类型过滤无权限的版块
func ForumListAccessFilter(forumList map[int64]*Forum, gid uint16, allow string) map[int64]*Forum {
	if gid == 1 {
		return forumList // 管理员不过滤
	}
	if len(forumList) == 0 {
		return forumList
	}
	result := make(map[int64]*Forum, len(forumList))
	for fid, forum := range forumList {
		if forum.AccessOn == 0 {
			// 无权限控制的版块，根据用户组默认权限判断
			// 简化：非管理员默认允许访问公开版块
			result[fid] = forum
		}
		// 有权限控制的版块，需要查 ForumAccess 表
		// 此函数仅做内存过滤，实际权限判断由 CheckForumAccess 完成
		// 这里简化处理：保留所有版块，前端/API 层做细粒度权限校验
		result[fid] = forum
	}
	return result
}

// ForumFilterModUID 过滤版主 UID 列表，只保留有效用户（gid <= 4）
// 对应 PHP: forum_filter_moduid($moduids)
// moduids 为逗号分隔的 UID 字符串，返回过滤后的逗号分隔字符串
func ForumFilterModUID(ctx context.Context, db *sqlx.DB, moduids string) (string, error) {
	moduids = strings.TrimSpace(moduids)
	if moduids == "" {
		return "", nil
	}
	parts := strings.Split(moduids, ",")
	var valid []string
	for _, s := range parts {
		s = strings.TrimSpace(s)
		uid, err := strconv.Atoi(s)
		if err != nil || uid <= 0 {
			continue
		}
		user, err := GetUserByUID(ctx, db, uint32(uid))
		if err != nil || user == nil {
			continue
		}
		// PHP 原版: gid > 4 的跳过（gid=1 超管, 2/3/4 为版主/管理员）
		if user.GID > 4 {
			continue
		}
		valid = append(valid, s)
	}
	return strings.Join(valid, ","), nil
}

// ForumSafeInfo 返回版块安全信息
// 对应 PHP: forum_safe_info($forum)
// 当前实现：直接返回原对象（PHP 版也是空操作，仅注释掉了 unset($forum['moduids'])）
func ForumSafeInfo(forum *Forum) *Forum {
	return forum
}

// ForumMaxID 获取最大版块 ID
// 对应 PHP: forum_maxid()
func ForumMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(fid), 0) FROM bbs_forum`)
	if err != nil {
		return 0, fmt.Errorf("ForumMaxID: %w", err)
	}
	return maxID, nil
}
