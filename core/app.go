package core

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// AppCtx 全局依赖容器，替代 PHP 的 $_SERVER 超级全局变量
// 所有 handler 通过闭包注入此对象
type AppCtx struct {
	DB          *sqlx.DB
	Cache       Cache
	Conf        *Config
	Counter     *AsyncCounter // 异步计数器，解耦统计更新与核心事务
	Storage     Storage       // 文件存储驱动（LocalStorage / OSS / S3）
	RateLimiter *RateLimiter  // 内存限流器，防刷护城河
	Hook        *HookManager  // 插件 Hook 引擎（Filter + Action）
	Cron        Closer        // 计划任务（启动时自动运行）
	Policy      *Policy       // 统一权限判定中心，替代 GlobalPolicy 全局单例
}

// Closer 通用关闭接口
type Closer interface {
	Close()
}

// NewAppCtx 初始化应用上下文
func NewAppCtx(cfg *Config) *AppCtx {
	db := initDB(cfg)
	cache := NewCache(cfg)

	app := &AppCtx{
		DB:    db,
		Cache: cache,
		Conf:  cfg,
	}

	// 异步计数器依赖 DB，必须在 DB 初始化之后
	app.Counter = NewAsyncCounter(db)

	// 文件存储驱动（默认本地磁盘）
	app.Storage = NewLocalStorage("upload", "/upload/")

	// 内存限流器（零外部依赖）
	app.RateLimiter = NewRateLimiter()

	// 插件 Hook 引擎（Filter + Action）
	app.Hook = NewHookManager()

	// 权限策略实例（零外部依赖，仅使用基本类型）
	app.Policy = &Policy{}

	return app
}

func initDB(cfg *Config) *sqlx.DB {
	log.Printf("[DEBUG] 尝试连接数据库 DSN: %s", cfg.Database.DSN)
	db, err := sqlx.Connect("mysql", cfg.Database.DSN)
	if err != nil {
		log.Fatalf("[FATAL] 数据库连接失败: %v", err)
	}

	db.SetMaxOpenConns(cfg.Database.MaxOpen)
	db.SetMaxIdleConns(cfg.Database.MaxIdle)
	db.SetConnMaxLifetime(5 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Fatalf("[FATAL] 数据库 Ping 失败: %v", err)
	}

	if cfg.Database.TablePrefix == "" {
		cfg.Database.TablePrefix = "bbs_"
	}

	log.Printf("[INFO] 数据库连接成功, DSN: %s", maskDSN(cfg.Database.DSN))
	return db
}

func maskDSN(dsn string) string {
	bytes := []byte(dsn)
	colonIdx := -1
	atIdx := -1
	for i, b := range bytes {
		if b == ':' && colonIdx == -1 {
			colonIdx = i
		}
		if b == '@' {
			atIdx = i
			break
		}
	}
	if colonIdx != -1 && atIdx != -1 {
		return string(bytes[:colonIdx+1]) + "***" + string(bytes[atIdx:])
	}
	return dsn
}

// Close 关闭所有资源
func (app *AppCtx) Close() {
	if app.Cron != nil {
		app.Cron.Close()
	}
	if app.Counter != nil {
		app.Counter.Close()
	}
	if app.DB != nil {
		if err := app.DB.Close(); err != nil {
			log.Printf("[WARN] 数据库关闭异常: %v", err)
		}
	}
	if app.Cache != nil {
		if err := app.Cache.Close(); err != nil {
			log.Printf("[WARN] 缓存关闭异常: %v", err)
		}
	}
}

// Tx 事务闭包包装器，确保复杂逻辑的事务安全
func (app *AppCtx) Tx(fn func(tx *sqlx.Tx) error) error {
	tx, err := app.DB.Beginx()
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()
	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("[ERROR] 事务回滚失败: %v (原始错误: %v)", rbErr, err)
		}
		return err
	}
	return tx.Commit()
}

// Ensure sql.DB type is available
var _ = sql.ErrNoRows
