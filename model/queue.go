package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

/*
	MySQL 模拟队列
	对应 PHP: model/queue.func.php
	表结构: bbs_queue (queueid, v, expiry), UNIQUE KEY(queueid, v), KEY(expiry)

	注意：顺序可能是乱的，不是严格意义上的队列。
	主要用于后台管理「主题查找」功能，将扫描结果暂存到队列，然后分页操作。
*/

// QueuePush 添加到队列
// 对应 PHP: queue_push($queueid, $v, $expiry = 0)
func QueuePush(ctx context.Context, db *sqlx.DB, queueid uint32, v int64, expirySec int) error {
	expiry := int64(0)
	if expirySec > 0 {
		expiry = time.Now().Unix() + int64(expirySec)
	}
	_, err := db.ExecContext(ctx,
		`INSERT INTO bbs_queue (queueid, v, expiry) VALUES (?, ?, ?)
		 ON DUPLICATE KEY UPDATE expiry = VALUES(expiry)`,
		queueid, v, expiry)
	return err
}

// QueuePop 弹出某个值（取出并删除一条）
// 对应 PHP: queue_pop($queueid)
func QueuePop(ctx context.Context, db *sqlx.DB, queueid uint32) (int64, bool, error) {
	var v int64
	err := db.GetContext(ctx, &v,
		`SELECT v FROM bbs_queue WHERE queueid = ? LIMIT 1`, queueid)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, false, nil
		}
		return 0, false, err
	}
	// 删除已取出的记录
	_, delErr := db.ExecContext(ctx,
		`DELETE FROM bbs_queue WHERE queueid = ? AND v = ?`, queueid, v)
	if delErr != nil {
		return 0, false, delErr
	}
	return v, true, nil
}

// QueueDelete 删除某个值
// 对应 PHP: queue_delete($queueid, $v)
func QueueDelete(ctx context.Context, db *sqlx.DB, queueid uint32, v int64) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM bbs_queue WHERE queueid = ? AND v = ?`, queueid, v)
	return err
}

// QueueDestroy 销毁某个队列
// 对应 PHP: queue_destory($queueid)
func QueueDestroy(ctx context.Context, db *sqlx.DB, queueid uint32) error {
	_, err := db.ExecContext(ctx,
		`DELETE FROM bbs_queue WHERE queueid = ?`, queueid)
	return err
}

// QueueCount 获取队列中的元素数量
// 对应 PHP: queue_count($queueid)
func QueueCount(ctx context.Context, db *sqlx.DB, queueid uint32) (int, error) {
	var n int
	err := db.GetContext(ctx, &n,
		`SELECT COUNT(*) FROM bbs_queue WHERE queueid = ?`, queueid)
	return n, err
}

// QueueFind 提取整个队列（分页）
// 对应 PHP: queue_find($queueid, $page, $pagesize)
func QueueFind(ctx context.Context, db *sqlx.DB, queueid uint32, page, pageSize int) ([]int64, error) {
	offset := (page - 1) * pageSize
	var values []int64
	err := db.SelectContext(ctx, &values,
		`SELECT v FROM bbs_queue WHERE queueid = ? ORDER BY v ASC LIMIT ? OFFSET ?`,
		queueid, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return values, nil
}

// QueueGC 清理过期的队列数据
// 对应 PHP: queue_gc()
// 在每日 cron 任务中调用
func QueueGC(ctx context.Context, db *sqlx.DB) error {
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx,
		`DELETE FROM bbs_queue WHERE expiry > 0 AND expiry < ?`, now)
	return err
}
