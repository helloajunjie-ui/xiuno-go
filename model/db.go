// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// DBOrTx 兼容 *sqlx.DB 和 *sqlx.Tx 的公共接口
// 用于需要同时支持事务内和事务外调用的函数
// *sqlx.DB 和 *sqlx.Tx 都实现了这些方法
type DBOrTx interface {
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	GetContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
	SelectContext(ctx context.Context, dest interface{}, query string, args ...interface{}) error
}

// compile-time check: *sqlx.DB 和 *sqlx.Tx 都满足 DBOrTx
var _ DBOrTx = (*sqlx.DB)(nil)
var _ DBOrTx = (*sqlx.Tx)(nil)
