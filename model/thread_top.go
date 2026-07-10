package model

import (
	"context"
	"fmt"
	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// ThreadTopChange 设置/取消置顶
// 对应 PHP: thread_top_change()
// top=0 时删除置顶记录，top>0 时 INSERT OR REPLACE
func ThreadTopChange(ctx context.Context, ext DBOrTx, tid uint32, top int) error {
	if top <= 0 {
		// 取消置顶：删除记录
		_, err := ext.ExecContext(ctx, `DELETE FROM bbs_thread_top WHERE tid = ?`, tid)
		if err != nil {
			return fmt.Errorf("ThreadTopChange delete: %w", err)
		}
		return nil
	}

	// 获取帖子所在版块
	var fid uint32
	err := ext.GetContext(ctx, &fid, `SELECT fid FROM bbs_thread WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("ThreadTopChange read thread fid: %w", err)
	}

	// INSERT OR REPLACE（MySQL 用 REPLACE INTO）
	_, err = ext.ExecContext(ctx,
		`REPLACE INTO bbs_thread_top (fid, tid, top) VALUES (?, ?, ?)`,
		fid, tid, top)
	if err != nil {
		return fmt.Errorf("ThreadTopChange replace: %w", err)
	}
	return nil
}

// ThreadTopDelete 删除指定主题的置顶记录（删帖时调用）
// 对应 PHP: thread_top_delete()
func ThreadTopDelete(ctx context.Context, ext DBOrTx, tid uint32) error {
	_, err := ext.ExecContext(ctx, `DELETE FROM bbs_thread_top WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("ThreadTopDelete: %w", err)
	}
	return nil
}

// ThreadTopFind 获取置顶主题列表
// 对应 PHP: thread_top_find()
// fids 为空或包含 0 时返回全局置顶(top=3)
// fids 包含具体 fid 时返回对应版块的版内置顶(top=1)
// 支持多版块联合查询
func ThreadTopFind(ctx context.Context, db *sqlx.DB, fids ...uint32) ([]ThreadListItem, error) {
	var list []ThreadListItem
	var err error

	// 检查是否包含全局置顶请求
	hasGlobal := len(fids) == 0
	for _, fid := range fids {
		if fid == 0 {
			hasGlobal = true
			break
		}
	}

	if hasGlobal {
		// 全局置顶：top=3
		err = db.SelectContext(ctx, &list, `
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread_top tt
			INNER JOIN bbs_thread t ON tt.tid = t.tid AND t.deleted_at IS NULL
			LEFT JOIN bbs_user u ON t.uid = u.uid
			WHERE tt.top = 3
			ORDER BY tt.tid DESC
			LIMIT 100`)
	} else {
		// 版内置顶：top=1，支持多版块
		query, args, err := sqlx.In(`
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread_top tt
			INNER JOIN bbs_thread t ON tt.tid = t.tid AND t.deleted_at IS NULL
			LEFT JOIN bbs_user u ON t.uid = u.uid
			WHERE tt.fid IN (?) AND tt.top = 1
			ORDER BY tt.tid DESC
			LIMIT 100`, fids)
		if err != nil {
			return nil, fmt.Errorf("ThreadTopFind build query: %w", err)
		}
		query = db.Rebind(query)
		err = db.SelectContext(ctx, &list, query, args...)
	}

	if err != nil {
		return nil, fmt.Errorf("ThreadTopFind: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ThreadTopUpdateByTID 移动帖子时更新 thread_top 中的 fid
// 对应 PHP: thread_top_update_by_tid()
func ThreadTopUpdateByTID(ctx context.Context, ext DBOrTx, tid uint32, newFid uint32) error {
	_, err := ext.ExecContext(ctx,
		`UPDATE bbs_thread_top SET fid = ? WHERE tid = ?`, newFid, tid)
	if err != nil {
		return fmt.Errorf("ThreadTopUpdateByTID: %w", err)
	}
	return nil
}

// ThreadTopFindCache 从缓存获取置顶主题列表
// 对应 PHP: thread_top_find_cache()
// 当前简化实现：直接调用 ThreadTopFind，不缓存
func ThreadTopFindCache(ctx context.Context, db *sqlx.DB, fids ...uint32) ([]ThreadListItem, error) {
	return ThreadTopFind(ctx, db, fids...)
}

// ThreadTopCacheDelete 删除置顶主题缓存
// 对应 PHP: thread_top_cache_delete()
// 当前简化实现：空操作，因为未使用缓存
func ThreadTopCacheDelete(ctx context.Context, cache core.Cache) {
	// 简化：未使用缓存，无需操作
	_ = ctx
	_ = cache
}
