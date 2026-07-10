package model

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"xiuno/core"
)

// Online 在线用户（对应 bbs_session 表）
// 对应 PHP: reference/model/session.func.php
type Online struct {
	SID      string `db:"sid" json:"sid"`
	UID      uint32 `db:"uid" json:"uid"`
	IP       uint32 `db:"ip" json:"ip"`
	LastDate int64  `db:"last_date" json:"last_date"`
}

// OnlineListItem 在线用户列表项（含用户名等展示信息）
type OnlineListItem struct {
	SID         string `db:"sid" json:"sid"`
	UID         uint32 `db:"uid" json:"uid"`
	Username    string `json:"username"`
	GID         uint16 `json:"gid"`
	IP          uint32 `db:"ip" json:"ip"`
	IPFmt       string `json:"ip_fmt"`
	LastDate    int64  `db:"last_date" json:"last_date"`
	LastDateFmt string `json:"last_date_fmt"`
}

// OnlineCount 统计在线用户数量
// 对应 PHP: online_count()
func OnlineCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_session`)
	if err != nil {
		return 0, fmt.Errorf("OnlineCount: %w", err)
	}
	return count, nil
}

// OnlineFindCache 获取所有在线会话记录
// 对应 PHP: online_find_cache()
// 当前简化实现：直接查询数据库，不缓存
func OnlineFindCache(ctx context.Context, db *sqlx.DB) ([]Online, error) {
	var list []Online
	err := db.SelectContext(ctx, &list, `SELECT * FROM bbs_session`)
	if err != nil {
		return nil, fmt.Errorf("OnlineFindCache: %w", err)
	}
	if list == nil {
		list = []Online{}
	}
	return list, nil
}

// OnlineListCache 获取在线用户列表（含用户信息，缓存 300 秒）
// 对应 PHP: online_list_cache()
func OnlineListCache(ctx context.Context, db *sqlx.DB, cache core.Cache) ([]OnlineListItem, error) {
	// 尝试从缓存读取
	if data, ok := cache.Get(ctx, "online_list"); ok {
		var list []OnlineListItem
		if err := json.Unmarshal(data, &list); err == nil {
			return list, nil
		}
	}

	// 查询有 UID 的在线用户（已登录用户），按最后活动时间倒序，最多 500 条
	var list []OnlineListItem
	err := db.SelectContext(ctx, &list, `
		SELECT s.sid, s.uid, s.ip, s.last_date
		FROM bbs_session s
		WHERE s.uid > 0
		ORDER BY s.last_date DESC
		LIMIT 500`)
	if err != nil {
		return nil, fmt.Errorf("OnlineListCache: %w", err)
	}

	// 填充用户信息
	for i := range list {
		user, err := GetUserByUID(ctx, db, list[i].UID)
		if err == nil && user != nil {
			list[i].Username = user.Username
			list[i].GID = user.GID
		}
		list[i].IPFmt = long2ip(list[i].IP)
		list[i].LastDateFmt = time.Unix(list[i].LastDate, 0).Format("2006-1-2 15:04")
	}

	if list == nil {
		list = []OnlineListItem{}
	}

	// 写入缓存，有效期 300 秒
	if data, err := json.Marshal(list); err == nil {
		cache.Set(ctx, "online_list", data, 300*time.Second)
	}

	return list, nil
}

// long2ip 将 uint32 IP 转换为点分十进制字符串
func long2ip(ip uint32) string {
	return fmt.Sprintf("%d.%d.%d.%d", byte(ip>>24), byte(ip>>16), byte(ip>>8), byte(ip))
}
