// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

/*
	bbs_table_day 每日最大ID统计
	对应 PHP: model/table_day.func.php

	用途：记录 thread/post/user 表每天的最大 ID 和数量，
	用于削减 create_date 索引，加速冷热数据过滤。

	表结构：
	  year   smallint  - 年
	  month  tinyint   - 月
	  day    tinyint   - 日（0 表示月，month=0 AND day=0 表示年）
	  create_date int  - 时间戳
	  table  char(16)  - 表名
	  maxid  int       - 最大 ID
	  count  int       - 总数
	  PRIMARY KEY (year, month, day, table)
*/

// TableDayRecord 对应 bbs_table_day 表
type TableDayRecord struct {
	Year       uint16 `db:"year" json:"year"`
	Month      uint8  `db:"month" json:"month"`
	Day        uint8  `db:"day" json:"day"`
	CreateDate int64  `db:"create_date" json:"create_date"`
	Table      string `db:"table" json:"table"`
	MaxID      uint32 `db:"maxid" json:"maxid"`
	Count      uint32 `db:"count" json:"count"`
}

// tableDayTables 需要统计的表及其 ID 字段
var tableDayTables = map[string]string{
	"thread": "tid",
	"post":   "pid",
	"user":   "uid",
}

// TableDayRead 读取某天的记录
// 对应 PHP: table_day_read($table, $year, $month, $day)
func TableDayRead(ctx context.Context, db *sqlx.DB, table string, year uint16, month, day uint8) (*TableDayRecord, error) {
	var rec TableDayRecord
	err := db.GetContext(ctx, &rec,
		`SELECT * FROM bbs_table_day WHERE year = ? AND month = ? AND day = ? AND \`+"`table`"+` = ?`,
		year, month, day, table)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("TableDayRead: %w", err)
	}
	return &rec, nil
}

// TableDayMaxID 获取某天的最大 ID
// 支持两种日期格式：Unix 时间戳
// 对应 PHP: table_day_maxid($table, $date)
func TableDayMaxID(ctx context.Context, db *sqlx.DB, table string, date int64) (uint32, error) {
	// 不能小于 2014-9-24
	const minTime = 1411516800
	if date < minTime {
		return 0, nil
	}

	t := time.Unix(date, 0)
	year := uint16(t.Year())
	month := uint8(t.Month())
	day := uint8(t.Day())

	rec, err := TableDayRead(ctx, db, table, year, month, day)
	if err != nil {
		return 0, err
	}
	if rec == nil {
		return 0, nil
	}
	return rec.MaxID, nil
}

// TableDayCron 统计某天的数据并写入 bbs_table_day
// 在每日 cron 任务中执行，统计前一天的数据
// 对应 PHP: table_day_cron($crontime)
func TableDayCron(ctx context.Context, db *sqlx.DB, crontime int64) error {
	t := time.Unix(crontime, 0)
	year := uint16(t.Year())
	month := uint8(t.Month())
	day := uint8(t.Day())

	for table, col := range tableDayTables {
		// 查询该天之前（含该天）的最大 ID
		var maxID uint32
		err := db.GetContext(ctx, &maxID,
			fmt.Sprintf(`SELECT COALESCE(MAX(%s), 0) FROM bbs_%s WHERE create_date < ?`, col, table),
			crontime)
		if err != nil {
			return fmt.Errorf("TableDayCron maxid %s: %w", table, err)
		}

		// 查询该天之前的数量
		var count uint32
		err = db.GetContext(ctx, &count,
			fmt.Sprintf(`SELECT COUNT(*) FROM bbs_%s WHERE create_date < ?`, table),
			crontime)
		if err != nil {
			return fmt.Errorf("TableDayCron count %s: %w", table, err)
		}

		// 写入（REPLACE 语义）
		_, err = db.ExecContext(ctx,
			"REPLACE INTO bbs_table_day (year, month, day, create_date, `table`, maxid, `count`)"+
				" VALUES (?, ?, ?, ?, ?, ?, ?)",
			year, month, day, crontime, table, maxID, count)
		if err != nil {
			return fmt.Errorf("TableDayCron insert %s: %w", table, err)
		}
	}

	return nil
}

// TableDayRebuild 从用户创建日期开始重建所有日期的统计数据
// 对应 PHP: table_day_rebuild()
func TableDayRebuild(ctx context.Context, db *sqlx.DB) error {
	// 获取第一个用户的创建日期作为起始点
	var createDate int64
	err := db.GetContext(ctx, &createDate, `SELECT create_date FROM bbs_user ORDER BY uid ASC LIMIT 1`)
	if err != nil {
		// 如果没有用户，从当前时间往前推 30 天
		createDate = time.Now().AddDate(0, 0, -30).Unix()
	}

	now := time.Now().Unix()
	crontime := createDate
	for crontime < now {
		if err := TableDayCron(ctx, db, crontime); err != nil {
			return fmt.Errorf("TableDayRebuild at %d: %w", crontime, err)
		}
		crontime += 86400
	}

	return nil
}
