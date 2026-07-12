// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
)

// 自动建表 SQL（与 Go struct 字段对齐）
// 差异记录（相对于原始 PHP install.sql）：
//   - bbs_user: 新增 created_at, updated_at (time.Time)
//   - bbs_thread: 新增 deleted_at (sql.NullTime, 软删除)
//   - bbs_post: 新增 deleted_at (sql.NullTime, 软删除)
//   - 所有表引擎从 MyISAM 升级为 InnoDB（支持事务）
//   - 所有表字符集从 utf8 升级为 utf8mb4（与 DSN 对齐）

// schemaDDL 返回完整的建表 DDL 列表
func schemaDDL() []string {
	// bbs_table_day 单独定义，因为列名 `table` 是 MySQL 保留字，需要反引号
	// Go raw string literal 不支持内嵌反引号，所以用拼接
	tableDaySQL := `CREATE TABLE IF NOT EXISTS bbs_table_day (` +
		"\n  year smallint(11) unsigned NOT NULL DEFAULT '0' COMMENT '年'," +
		"\n  month tinyint(11) unsigned NOT NULL DEFAULT '0' COMMENT '月'," +
		"\n  day tinyint(11) unsigned NOT NULL DEFAULT '0' COMMENT '日'," +
		"\n  create_date int(11) unsigned NOT NULL DEFAULT '0' COMMENT '时间戳'," +
		"\n  `table` char(16) NOT NULL default '' COMMENT '表名'," +
		"\n  maxid int(11) unsigned NOT NULL DEFAULT '0' COMMENT '最大ID'," +
		"\n  count int(11) unsigned NOT NULL DEFAULT '0' COMMENT '总数'," +
		"\n  PRIMARY KEY (year, month, day, `table`)" +
		"\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;"

	return []string{
		`CREATE TABLE IF NOT EXISTS bbs_user (
  uid int(11) unsigned NOT NULL AUTO_INCREMENT COMMENT '用户编号',
  gid smallint(6) unsigned NOT NULL DEFAULT '0' COMMENT '用户组编号',
  email char(40) NOT NULL DEFAULT '' COMMENT '邮箱',
  username char(32) NOT NULL DEFAULT '' COMMENT '用户名',
  realname char(16) NOT NULL DEFAULT '' COMMENT '真实姓名',
  idnumber char(19) NOT NULL DEFAULT '' COMMENT '身份证号码',
  password char(60) NOT NULL DEFAULT '' COMMENT '密码(bcrypt)',
  password_sms char(16) NOT NULL DEFAULT '' COMMENT '短信验证码',
  salt char(16) NOT NULL DEFAULT '' COMMENT '密码混杂',
  mobile char(11) NOT NULL DEFAULT '' COMMENT '手机号',
  qq char(15) NOT NULL DEFAULT '' COMMENT 'QQ',
  threads int(11) NOT NULL DEFAULT '0' COMMENT '发帖数',
  posts int(11) NOT NULL DEFAULT '0' COMMENT '回帖数',
  credits int(11) NOT NULL DEFAULT '0' COMMENT '积分',
  golds int(11) NOT NULL DEFAULT '0' COMMENT '金币',
  rmbs int(11) NOT NULL DEFAULT '0' COMMENT '人民币',
  create_ip int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时IP',
  create_date int(11) unsigned NOT NULL DEFAULT '0' COMMENT '创建时间',
  login_ip int(11) unsigned NOT NULL DEFAULT '0' COMMENT '登录时IP',
  login_date int(11) unsigned NOT NULL DEFAULT '0' COMMENT '登录时间',
  logins int(11) unsigned NOT NULL DEFAULT '0' COMMENT '登录次数',
  avatar int(11) unsigned NOT NULL DEFAULT '0' COMMENT '头像更新时间',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP COMMENT '记录创建时间',
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '记录更新时间',
  PRIMARY KEY (uid),
  UNIQUE KEY username (username),
  UNIQUE KEY email (email),
  KEY gid (gid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_group (
  gid smallint(6) unsigned NOT NULL,
  name char(20) NOT NULL default '',
  creditsfrom int(11) NOT NULL default '0',
  creditsto int(11) NOT NULL default '0',
  allowread int(11) NOT NULL default '0',
  allowthread int(11) NOT NULL default '0',
  allowpost int(11) NOT NULL default '0',
  allowattach int(11) NOT NULL default '0',
  allowdown int(11) NOT NULL default '0',
  allowtop int(11) NOT NULL default '0',
  allowupdate int(11) NOT NULL default '0',
  allowdelete int(11) NOT NULL default '0',
  allowmove int(11) NOT NULL default '0',
  allowbanuser int(11) NOT NULL default '0',
  allowdeleteuser int(11) NOT NULL default '0',
  allowviewip int(11) unsigned NOT NULL default '0',
  PRIMARY KEY (gid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		"CREATE TABLE IF NOT EXISTS bbs_forum (" +
			"\n  fid int(11) unsigned NOT NULL auto_increment," +
			"\n  name char(16) NOT NULL default ''," +
			"\n  `rank` tinyint(3) unsigned NOT NULL default '0'," +
			"\n  threads mediumint(8) unsigned NOT NULL default '0'," +
			"\n  todayposts mediumint(8) unsigned NOT NULL default '0'," +
			"\n  todaythreads mediumint(8) unsigned NOT NULL default '0'," +
			"\n  brief text NOT NULL," +
			"\n  announcement text NOT NULL," +
			"\n  accesson int(11) unsigned NOT NULL default '0'," +
			"\n  `orderby` tinyint(11) NOT NULL default '0'," +
			"\n  create_date int(11) unsigned NOT NULL default '0'," +
			"\n  icon int(11) unsigned NOT NULL default '0'," +
			"\n  moduids char(120) NOT NULL default ''," +
			"\n  seo_title char(64) NOT NULL default ''," +
			"\n  seo_keywords char(64) NOT NULL default ''," +
			"\n  PRIMARY KEY (fid)" +
			"\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;",
		`CREATE TABLE IF NOT EXISTS bbs_forum_access (
  fid int(11) unsigned NOT NULL default '0',
  gid int(11) unsigned NOT NULL default '0',
  allowread tinyint(1) unsigned NOT NULL default '0',
  allowthread tinyint(1) unsigned NOT NULL default '0',
  allowpost tinyint(1) unsigned NOT NULL default '0',
  allowattach tinyint(1) unsigned NOT NULL default '0',
  allowdown tinyint(1) unsigned NOT NULL default '0',
  PRIMARY KEY (fid, gid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_thread (
  fid smallint(6) NOT NULL default '0',
  tid int(11) unsigned NOT NULL auto_increment,
  top tinyint(1) NOT NULL default '0',
  uid int(11) unsigned NOT NULL default '0',
  userip int(11) unsigned NOT NULL default '0',
  subject char(128) NOT NULL default '',
  create_date int(11) unsigned NOT NULL default '0',
  last_date int(11) unsigned NOT NULL default '0',
  views int(11) unsigned NOT NULL default '0',
  posts int(11) unsigned NOT NULL default '0',
  images tinyint(6) NOT NULL default '0',
  files tinyint(6) NOT NULL default '0',
  mods tinyint(6) NOT NULL default '0',
  closed tinyint(1) unsigned NOT NULL default '0',
  firstpid int(11) unsigned NOT NULL default '0',
  lastuid int(11) unsigned NOT NULL default '0',
  lastpid int(11) unsigned NOT NULL default '0',
  deleted_at datetime DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (tid),
  KEY (lastpid),
  KEY (fid, tid),
  KEY (fid, lastpid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_thread_top (
  fid smallint(6) NOT NULL default '0',
  tid int(11) unsigned NOT NULL default '0',
  top int(11) unsigned NOT NULL default '0',
  PRIMARY KEY (tid),
  KEY (top, tid),
  KEY (fid, top)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_post (
  tid int(11) unsigned NOT NULL default '0',
  pid int(11) unsigned NOT NULL auto_increment,
  uid int(11) unsigned NOT NULL default '0',
  isfirst int(11) unsigned NOT NULL default '0',
  create_date int(11) unsigned NOT NULL default '0',
  userip int(11) unsigned NOT NULL default '0',
  images smallint(6) NOT NULL default '0',
  files smallint(6) NOT NULL default '0',
  doctype tinyint(3) NOT NULL default '0',
  quotepid int(11) NOT NULL default '0',
  message longtext NOT NULL,
  message_fmt longtext NOT NULL,
  deleted_at datetime DEFAULT NULL COMMENT '软删除时间',
  PRIMARY KEY (pid),
  KEY (tid, pid),
  KEY (uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_attach (
  aid int(11) unsigned NOT NULL auto_increment,
  tid int(11) NOT NULL default '0',
  pid int(11) NOT NULL default '0',
  uid int(11) NOT NULL default '0',
  filesize int(8) unsigned NOT NULL default '0',
  width mediumint(8) unsigned NOT NULL default '0',
  height mediumint(8) unsigned NOT NULL default '0',
  filename char(120) NOT NULL default '',
  orgfilename char(120) NOT NULL default '',
  filetype char(20) NOT NULL default '',
  create_date int(11) unsigned NOT NULL default '0',
  comment char(100) NOT NULL default '',
  downloads int(11) NOT NULL default '0',
  credits int(11) NOT NULL default '0',
  golds int(11) NOT NULL default '0',
  rmbs int(11) NOT NULL default '0',
  isimage tinyint(1) NOT NULL default '0',
  PRIMARY KEY (aid),
  KEY pid (pid),
  KEY uid (uid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_mythread (
  uid int(11) unsigned NOT NULL default '0',
  tid int(11) unsigned NOT NULL default '0',
  PRIMARY KEY (uid, tid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_mypost (
  uid int(11) unsigned NOT NULL default '0',
  tid int(11) unsigned NOT NULL default '0',
  pid int(11) unsigned NOT NULL default '0',
  KEY (tid),
  PRIMARY KEY (uid, pid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_session (
  sid char(32) NOT NULL default '0',
  uid int(11) unsigned NOT NULL default '0',
  fid tinyint(3) unsigned NOT NULL default '0',
  url char(32) NOT NULL default '',
  ip int(11) unsigned NOT NULL default '0',
  useragent char(128) NOT NULL default '',
  data char(255) NOT NULL default '',
  bigdata tinyint(1) NOT NULL default '0',
  last_date int(11) unsigned NOT NULL default '0',
  PRIMARY KEY (sid),
  KEY ip (ip),
  KEY fid (fid),
  KEY uid_last_date (uid, last_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_session_data (
  sid char(32) NOT NULL default '0',
  last_date int(11) unsigned NOT NULL default '0',
  data text NOT NULL,
  PRIMARY KEY (sid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_modlog (
  logid int(11) unsigned NOT NULL auto_increment,
  uid int(11) unsigned NOT NULL default '0',
  tid int(11) unsigned NOT NULL default '0',
  pid int(11) unsigned NOT NULL default '0',
  subject char(32) NOT NULL default '',
  comment char(64) NOT NULL default '',
  rmbs int(11) NOT NULL default '0',
  create_date int(11) unsigned NOT NULL default '0',
  action char(16) NOT NULL default '',
  PRIMARY KEY (logid),
  KEY (uid, logid),
  KEY (tid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_kv (
  k char(32) NOT NULL default '',
  v mediumtext NOT NULL,
  expiry int(11) unsigned NOT NULL default '0',
  PRIMARY KEY(k)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_cache (
  k char(32) NOT NULL default '',
  v mediumtext NOT NULL,
  expiry int(11) unsigned NOT NULL default '0',
  PRIMARY KEY(k)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_queue (
		queueid int(11) unsigned NOT NULL default '0',
		v int(11) NOT NULL default '0',
		expiry int(11) unsigned NOT NULL default '0',
		UNIQUE KEY(queueid, v),
		KEY(expiry)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_tag (
		tagid int(11) unsigned NOT NULL auto_increment,
		name char(32) NOT NULL default '' COMMENT '标签名称',
		threads int(11) unsigned NOT NULL default '0' COMMENT '关联主题数',
		create_date int(11) unsigned NOT NULL default '0' COMMENT '创建时间',
		PRIMARY KEY (tagid),
		UNIQUE KEY name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		`CREATE TABLE IF NOT EXISTS bbs_thread_tag (
		tid int(11) unsigned NOT NULL default '0',
		tagid int(11) unsigned NOT NULL default '0',
		PRIMARY KEY (tid, tagid),
		KEY tagid (tagid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;`,
		tableDaySQL,
	}
}

// seedStatements 返回初始数据 INSERT 语句列表
func seedStatements() []string {
	return []string{
		`INSERT IGNORE INTO bbs_group SET gid=0, name='游客组', creditsfrom=0, creditsto=0, allowread=1, allowthread=0, allowpost=1, allowattach=0, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=1, name='管理员组', creditsfrom=0, creditsto=0, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=1, allowupdate=1, allowdelete=1, allowmove=1, allowbanuser=1, allowdeleteuser=1, allowviewip=1;`,
		`INSERT IGNORE INTO bbs_group SET gid=2, name='超级版主组', creditsfrom=0, creditsto=0, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=1, allowupdate=1, allowdelete=1, allowmove=1, allowbanuser=1, allowdeleteuser=1, allowviewip=1;`,
		`INSERT IGNORE INTO bbs_group SET gid=4, name='版主组', creditsfrom=0, creditsto=0, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=1, allowupdate=1, allowdelete=1, allowmove=1, allowbanuser=1, allowdeleteuser=0, allowviewip=1;`,
		`INSERT IGNORE INTO bbs_group SET gid=5, name='实习版主组', creditsfrom=0, creditsto=0, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=1, allowupdate=1, allowdelete=0, allowmove=1, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=6, name='待验证用户组', creditsfrom=0, creditsto=0, allowread=1, allowthread=0, allowpost=1, allowattach=0, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=7, name='禁止用户组', creditsfrom=0, creditsto=0, allowread=0, allowthread=0, allowpost=0, allowattach=0, allowdown=0, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=101, name='一级用户组', creditsfrom=0, creditsto=50, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=102, name='二级用户组', creditsfrom=50, creditsto=200, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=103, name='三级用户组', creditsfrom=200, creditsto=1000, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=104, name='四级用户组', creditsfrom=1000, creditsto=10000, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_group SET gid=105, name='五级用户组', creditsfrom=10000, creditsto=10000000, allowread=1, allowthread=1, allowpost=1, allowattach=1, allowdown=1, allowtop=0, allowupdate=0, allowdelete=0, allowmove=0, allowbanuser=0, allowdeleteuser=0, allowviewip=0;`,
		`INSERT IGNORE INTO bbs_forum SET fid=1, name='默认版块', brief='默认版块介绍';`,
	}
}

// AutoMigrate 自动建表 + 初始数据填充 + 运行时迁移
// 幂等设计：CREATE TABLE IF NOT EXISTS + INSERT IGNORE
// 已存在的表不会重复创建，已有数据不会重复插入
func AutoMigrate(db *sqlx.DB) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Println("[MIGRATE] 开始自动建表...")

	// 1. 逐条执行建表 SQL
	ddls := schemaDDL()
	for i, stmt := range ddls {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("[MIGRATE] 建表失败 (语句 #%d): %w\nSQL: %s", i, err, truncateSQL(stmt))
		}
	}
	log.Printf("[MIGRATE] 建表完成 (%d 条 DDL)", len(ddls))

	// 1b. 运行时迁移：修复已有表的列定义（CREATE TABLE IF NOT EXISTS 不会修改已存在的表）
	migrations := []struct {
		name string
		sql  string
	}{
		{
			name: "bbs_attach.filetype char(7) -> char(20)",
			sql:  "ALTER TABLE bbs_attach MODIFY COLUMN filetype char(20) NOT NULL default ''",
		},
	}
	for _, m := range migrations {
		if _, err := db.ExecContext(ctx, m.sql); err != nil {
			// 忽略列已存在等错误（幂等）
			log.Printf("[MIGRATE] 迁移 '%s' 跳过: %v", m.name, err)
		} else {
			log.Printf("[MIGRATE] 迁移 '%s' 完成", m.name)
		}
	}

	// 2. 清理旧版预置 admin 账号（密码为空，无法登录）
	// 旧版 seedStatements() 曾包含 INSERT IGNORE INTO bbs_user SET uid=1, gid=1, password=''
	// 该数据已写入持久卷，必须显式清理
	cleanResult, err := db.ExecContext(ctx, "DELETE FROM bbs_user WHERE password = '' OR password IS NULL")
	if err != nil {
		return fmt.Errorf("[MIGRATE] 清理空密码用户失败: %w", err)
	}
	if n, _ := cleanResult.RowsAffected(); n > 0 {
		log.Printf("[MIGRATE] 已清理 %d 个空密码的旧版预置账号", n)
	}

	// 3. 检查是否需要填充初始数据
	var groupCount int
	if err := db.GetContext(ctx, &groupCount, "SELECT COUNT(*) FROM bbs_group"); err != nil {
		return fmt.Errorf("[MIGRATE] 检查初始数据失败: %w", err)
	}

	if groupCount == 0 {
		log.Println("[MIGRATE] 检测到空数据库，开始填充初始数据...")
		seeds := seedStatements()
		for i, stmt := range seeds {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("[MIGRATE] 初始数据插入失败 (语句 #%d): %w\nSQL: %s", i, err, truncateSQL(stmt))
			}
		}
		log.Printf("[MIGRATE] 初始数据填充完成 (%d 条 INSERT)", len(seeds))
	} else {
		log.Printf("[MIGRATE] 数据库已有数据 (%d 个用户组)，跳过初始数据填充", groupCount)
	}

	// 4. 运行时修正：修复版块帖子计数（AsyncCounter 在容器重启时会丢失未刷新的计数）
	//    使用子查询从 bbs_thread 实时统计未删除的帖子数
	log.Println("[MIGRATE] 开始修正版块帖子计数...")
	if _, err := db.ExecContext(ctx, `
		UPDATE bbs_forum f
		SET f.threads = (
			SELECT COUNT(*) FROM bbs_thread t
			WHERE t.fid = f.fid AND t.deleted_at IS NULL
		)
	`); err != nil {
		log.Printf("[MIGRATE] 修正版块帖子计数失败: %v", err)
	} else {
		log.Println("[MIGRATE] 版块帖子计数修正完成")
	}

	log.Println("[MIGRATE] 自动建表完成")
	return nil
}

func truncateSQL(s string) string {
	// 压缩空白以便日志显示
	compacted := strings.Join(strings.Fields(s), " ")
	if len(compacted) > 200 {
		return compacted[:200] + "..."
	}
	return compacted
}
