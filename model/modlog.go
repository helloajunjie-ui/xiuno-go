package model

import (
	"context"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// ModLog 对应 bbs_modlog 表
type ModLog struct {
	LogID      int64  `db:"logid" json:"logid"`
	UID        int64  `db:"uid" json:"uid"`
	TID        int64  `db:"tid" json:"tid"`
	PID        int64  `db:"pid" json:"pid"`
	Subject    string `db:"subject" json:"subject"`
	Comment    string `db:"comment" json:"comment"`
	Rmbs       int64  `db:"rmbs" json:"rmbs"`
	CreateDate int64  `db:"create_date" json:"create_date"`
	Action     string `db:"action" json:"action"`
}

// ModLogItem 版务日志列表项（含操作用户名）
type ModLogItem struct {
	ModLog
	Username string `db:"username" json:"username"`
}

// CreateModLog 写入版务操作日志
func CreateModLog(ctx context.Context, db *sqlx.DB, uid, tid, pid uint32, subject, action, comment string) error {
	now := time.Now().Unix()
	_, err := db.ExecContext(ctx, `
		INSERT INTO bbs_modlog (uid, tid, pid, subject, comment, create_date, action)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		uid, tid, pid, subject, comment, now, action)
	return err
}

// FindModLog 查询版务日志列表（支持按 action 过滤，按时间倒序）
func FindModLog(ctx context.Context, db *sqlx.DB, action string, page, pageSize int) ([]ModLogItem, int, error) {
	// 先查总数
	var total int
	baseWhere := "FROM bbs_modlog"
	args := []interface{}{}
	if action != "" {
		baseWhere = "FROM bbs_modlog WHERE action = ?"
		args = append(args, action)
	}
	err := db.GetContext(ctx, &total, `SELECT COUNT(*) `+baseWhere, args...)
	if err != nil {
		return nil, 0, err
	}

	// 查列表（JOIN bbs_user 获取用户名）
	offset := (page - 1) * pageSize
	var list []ModLogItem
	sqlStr := `SELECT m.*, u.username FROM bbs_modlog m
		LEFT JOIN bbs_user u ON m.uid = u.uid ` +
		baseWhere + ` ORDER BY m.logid DESC LIMIT ? OFFSET ?`
	queryArgs := append(args, pageSize, offset)
	err = db.SelectContext(ctx, &list, sqlStr, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	if list == nil {
		list = []ModLogItem{}
	}
	return list, total, nil
}

// CountModLog 统计版务日志数量
func CountModLog(ctx context.Context, db *sqlx.DB, action string) (int, error) {
	var total int
	if action != "" {
		err := db.GetContext(ctx, &total, `SELECT COUNT(*) FROM bbs_modlog WHERE action = ?`, action)
		return total, err
	}
	err := db.GetContext(ctx, &total, `SELECT COUNT(*) FROM bbs_modlog`)
	return total, err
}

// ModLogFormat 格式化版务日志（填充显示字段）
// 对应 PHP: modlog_format()
func ModLogFormat(log *ModLog) {
	// 原版 PHP modlog_format() 仅格式化 create_date
	// 当前 ModLog 结构体 CreateDate 为 int64，前端自行格式化
	// 保留此函数作为接口兼容
}

// ModLogMaxID 获取最大日志 ID
// 对应 PHP: modlog_maxid()
func ModLogMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(logid), 0) FROM bbs_mod_log`)
	if err != nil {
		return 0, fmt.Errorf("ModLogMaxID: %w", err)
	}
	return maxID, nil
}

// ModLogUpdate 更新版务日志
// 对应 PHP: modlog_update($logid, $arr)
func ModLogUpdate(ctx context.Context, db *sqlx.DB, logid uint32, comment string) error {
	_, err := db.ExecContext(ctx,
		`UPDATE bbs_modlog SET comment = ? WHERE logid = ?`, comment, logid)
	if err != nil {
		return fmt.Errorf("ModLogUpdate: %w", err)
	}
	return nil
}

// ModLogRead 读取单条版务日志
// 对应 PHP: modlog_read($logid)
func ModLogRead(ctx context.Context, db *sqlx.DB, logid uint32) (*ModLog, error) {
	var log ModLog
	err := db.GetContext(ctx, &log, `SELECT * FROM bbs_modlog WHERE logid = ?`, logid)
	if err != nil {
		return nil, fmt.Errorf("ModLogRead: %w", err)
	}
	return &log, nil
}

// ModLogDelete 删除版务日志
// 对应 PHP: modlog_delete($logid)
func ModLogDelete(ctx context.Context, db *sqlx.DB, logid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_modlog WHERE logid = ?`, logid)
	if err != nil {
		return fmt.Errorf("ModLogDelete: %w", err)
	}
	return nil
}
