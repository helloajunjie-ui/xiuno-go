// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"fmt"
)

// MyThread 对应 bbs_mythread 表
// 记录用户参与过的主题（发帖或回帖），用于"我的帖子"列表
type MyThread struct {
	UID uint32 `db:"uid" json:"uid"`
	TID uint32 `db:"tid" json:"tid"`
}

// MyThreadCreate 记录用户参与的主题（发帖时调用）
// 对应 PHP: mythread_create()
// 匿名发帖(uid==0)不记录
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadCreate(ctx context.Context, ext DBOrTx, uid, tid uint32) error {
	if uid == 0 {
		return nil // 匿名发帖不记录
	}
	_, err := ext.ExecContext(ctx,
		`INSERT IGNORE INTO bbs_mythread (uid, tid) VALUES (?, ?)`,
		uid, tid)
	if err != nil {
		return fmt.Errorf("MyThreadCreate: %w", err)
	}
	return nil
}

// MyThreadDelete 删除单条参与记录
// 对应 PHP: mythread_delete()
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadDelete(ctx context.Context, ext DBOrTx, uid, tid uint32) error {
	_, err := ext.ExecContext(ctx,
		`DELETE FROM bbs_mythread WHERE uid = ? AND tid = ?`,
		uid, tid)
	if err != nil {
		return fmt.Errorf("MyThreadDelete: %w", err)
	}
	return nil
}

// MyThreadDeleteByUID 删除用户的所有参与记录（删用户时调用）
// 对应 PHP: mythread_delete_by_uid()
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadDeleteByUID(ctx context.Context, ext DBOrTx, uid uint32) error {
	_, err := ext.ExecContext(ctx,
		`DELETE FROM bbs_mythread WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("MyThreadDeleteByUID: %w", err)
	}
	return nil
}

// MyThreadDeleteByTID 删除主题的所有参与记录（删帖时调用）
// 对应 PHP: mythread_delete_by_tid()
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadDeleteByTID(ctx context.Context, ext DBOrTx, tid uint32) error {
	_, err := ext.ExecContext(ctx,
		`DELETE FROM bbs_mythread WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("MyThreadDeleteByTID: %w", err)
	}
	return nil
}

// MyThreadFindByUID 查询用户参与过的主题列表（分页，按 tid 降序）
// 对应 PHP: mythread_find_by_uid()
// 返回 ThreadListItem（JOIN bbs_user 获取用户名和头像）
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadFindByUID(ctx context.Context, ext DBOrTx, uid uint32, page, pageSize int) ([]ThreadListItem, error) {
	offset := (page - 1) * pageSize
	var list []ThreadListItem
	err := ext.SelectContext(ctx, &list, `
		SELECT t.*, u.username, u.avatar
		FROM bbs_mythread m
		INNER JOIN bbs_thread t ON m.tid = t.tid AND t.deleted_at IS NULL
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE m.uid = ?
		ORDER BY m.tid DESC
		LIMIT ? OFFSET ?`,
		uid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("MyThreadFindByUID: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// MyThreadCountByUID 统计用户参与的主题总数
// ext 可以是 *sqlx.DB 或 *sqlx.Tx
func MyThreadCountByUID(ctx context.Context, ext DBOrTx, uid uint32) (int, error) {
	var count int
	err := ext.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM bbs_mythread m
		 INNER JOIN bbs_thread t ON m.tid = t.tid AND t.deleted_at IS NULL
		 WHERE m.uid = ?`, uid)
	if err != nil {
		return 0, fmt.Errorf("MyThreadCountByUID: %w", err)
	}
	return count, nil
}

// MyThreadRead 读取单条参与记录
// 对应 PHP: mythread_read($uid, $tid)
func MyThreadRead(ctx context.Context, ext DBOrTx, uid, tid uint32) (*MyThread, error) {
	var mt MyThread
	err := ext.GetContext(ctx, &mt,
		`SELECT * FROM bbs_mythread WHERE uid = ? AND tid = ?`, uid, tid)
	if err != nil {
		return nil, fmt.Errorf("MyThreadRead: %w", err)
	}
	return &mt, nil
}

// MyThreadDeleteByFID 删除版块下所有用户的参与记录（删版块时调用）
// 对应 PHP: mythread_delete_by_fid($fid)
func MyThreadDeleteByFID(ctx context.Context, ext DBOrTx, fid uint32) error {
	_, err := ext.ExecContext(ctx,
		`DELETE m FROM bbs_mythread m INNER JOIN bbs_thread t ON m.tid = t.tid WHERE t.fid = ?`, fid)
	if err != nil {
		return fmt.Errorf("MyThreadDeleteByFID: %w", err)
	}
	return nil
}

// MyThreadFind 通用查询 my_thread 记录
// 对应 PHP: mythread_find($cond, $orderby, $page, $pagesize)
// cond 支持: uid, tid
func MyThreadFind(ctx context.Context, ext DBOrTx, uid uint32, page, pageSize int) ([]MyThread, error) {
	offset := (page - 1) * pageSize
	var list []MyThread
	err := ext.SelectContext(ctx, &list, `
		SELECT * FROM bbs_mythread
		WHERE uid = ?
		ORDER BY tid DESC
		LIMIT ? OFFSET ?`,
		uid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("MyThreadFind: %w", err)
	}
	if list == nil {
		list = []MyThread{}
	}
	return list, nil
}
