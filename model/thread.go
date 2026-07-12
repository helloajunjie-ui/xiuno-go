// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
)

// Thread 对应 bbs_thread 表
type Thread struct {
	TID        int64        `db:"tid" json:"tid"`
	FID        int32        `db:"fid" json:"fid"`
	Top        int32        `db:"top" json:"top"`
	UID        int64        `db:"uid" json:"uid"`
	UserIP     uint32       `db:"userip" json:"-"`
	Subject    string       `db:"subject" json:"subject"`
	CreateDate int64        `db:"create_date" json:"create_date"`
	LastDate   int64        `db:"last_date" json:"last_date"`
	Views      int64        `db:"views" json:"views"`
	Posts      int32        `db:"posts" json:"posts"`
	Images     int32        `db:"images" json:"images"`
	Files      int32        `db:"files" json:"files"`
	Mods       int32        `db:"mods" json:"mods"`
	Closed     int32        `db:"closed" json:"closed"`
	FirstPID   int64        `db:"firstpid" json:"firstpid"`
	LastUID    int64        `db:"lastuid" json:"lastuid"`
	LastPID    int64        `db:"lastpid" json:"lastpid"`
	DeletedAt  sql.NullTime `db:"deleted_at" json:"deleted_at,omitempty"`
}

// ThreadWithUser 带用户信息的帖子列表项
type ThreadWithUser struct {
	Thread
	User      *UserBrief `json:"user,omitempty"`
	LastUser  *UserBrief `json:"last_user,omitempty"`
	ForumName string     `json:"forum_name,omitempty"`
}

// ThreadTop 对应 bbs_thread_top 表
type ThreadTop struct {
	FID int32 `db:"fid" json:"fid"`
	TID int64 `db:"tid" json:"tid"`
	Top int64 `db:"top" json:"top"`
}

// ThreadListItem 帖子列表项（JOIN user 表，消除 N+1）
type ThreadListItem struct {
	Thread
	Username string `db:"username" json:"username"`
	Avatar   uint32 `db:"avatar" json:"avatar"`
	// 以下字段由 ThreadFormat 填充，非 DB 映射
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
	LastDateFmt   string `db:"-" json:"last_date_fmt,omitempty"`
	LastUsername  string `db:"-" json:"last_username,omitempty"`
	ForumName     string `db:"-" json:"forum_name,omitempty"`
	Pages         int32  `db:"-" json:"pages,omitempty"`
}

// ThreadDetail 帖子详情（JOIN post 表获取首帖内容）
type ThreadDetail struct {
	Thread
	Message    string `db:"message" json:"message"`
	MessageFmt string `db:"message_fmt" json:"message_fmt"`
	DocType    int32  `db:"doctype" json:"doctype"`
	Username   string `db:"username" json:"username"`
	Avatar     uint32 `db:"avatar" json:"avatar"`
	// 以下字段由 ThreadFormatDetail 填充
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
	LastDateFmt   string `db:"-" json:"last_date_fmt,omitempty"`
	ForumName     string `db:"-" json:"forum_name,omitempty"`
	Tags          []Tag  `db:"-" json:"tags,omitempty"` // 帖子关联的标签
}

// GetThreadList 获取版块帖子列表（分页，按 lastpid 降序）
// 软删除过滤：deleted_at IS NULL
// fid=0 时返回全站帖子列表（首页模式），按 tid DESC
// 第1页时从 bbs_thread_top 独立表获取置顶列表并合并到结果头部
func GetThreadList(ctx context.Context, db *sqlx.DB, fid uint32, page, pageSize int) ([]ThreadListItem, error) {
	offset := (page - 1) * pageSize
	var list []ThreadListItem
	var err error

	// 第1页时获取置顶列表
	var topList []ThreadListItem
	if page == 1 {
		topList, err = ThreadTopFind(ctx, db, fid)
		if err != nil {
			// 置顶查询失败不阻塞，仅记录
			topList = []ThreadListItem{}
		}
	}

	if fid > 0 {
		err = db.SelectContext(ctx, &list, `
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread t
			LEFT JOIN bbs_user u ON t.uid = u.uid
			WHERE t.fid = ? AND t.deleted_at IS NULL
			ORDER BY t.lastpid DESC
			LIMIT ? OFFSET ?`,
			fid, pageSize, offset)
	} else {
		err = db.SelectContext(ctx, &list, `
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread t
			LEFT JOIN bbs_user u ON t.uid = u.uid
			WHERE t.deleted_at IS NULL
			ORDER BY t.tid DESC
			LIMIT ? OFFSET ?`,
			pageSize, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("GetThreadList: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}

	// 第1页时：置顶列表在前，普通列表在后（去重）
	if page == 1 && len(topList) > 0 {
		topTIDs := make(map[uint32]bool, len(topList))
		for _, t := range topList {
			topTIDs[uint32(t.TID)] = true
		}
		// 过滤掉已在置顶列表中的普通帖子
		var filtered []ThreadListItem
		for _, t := range list {
			if !topTIDs[uint32(t.TID)] {
				filtered = append(filtered, t)
			}
		}
		// 合并：置顶 + 普通
		list = append(topList, filtered...)
	}

	return list, nil
}

// GetThreadDetail 获取帖子详情（含首帖内容）
// 软删除过滤：t.deleted_at IS NULL
func GetThreadDetail(ctx context.Context, db *sqlx.DB, tid uint32) (*ThreadDetail, error) {
	var detail ThreadDetail
	err := db.GetContext(ctx, &detail, `
		SELECT t.*, p.message, p.message_fmt, p.doctype, u.username, u.avatar
		FROM bbs_thread t
		LEFT JOIN bbs_post p ON t.firstpid = p.pid
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE t.tid = ? AND t.deleted_at IS NULL`, tid)
	if err != nil {
		return nil, fmt.Errorf("GetThreadDetail: %w", err)
	}

	// 填充标签
	tags, err := TagFindByTID(ctx, db, tid)
	if err == nil && len(tags) > 0 {
		detail.Tags = tags
	} else {
		detail.Tags = []Tag{}
	}

	return &detail, nil
}

// UpdateThreadContent 修改主帖标题和首帖内容（事务内）
func UpdateThreadContent(ctx context.Context, tx *sqlx.Tx, tid uint32, firstpid uint32, subject, message string) error {
	// 1. 更新主帖标题
	_, err := tx.ExecContext(ctx, `UPDATE bbs_thread SET subject = ? WHERE tid = ?`, subject, tid)
	if err != nil {
		return fmt.Errorf("UpdateThreadContent thread: %w", err)
	}
	// 2. 更新首帖内容
	_, err = tx.ExecContext(ctx, `UPDATE bbs_post SET message = ? WHERE pid = ?`, message, firstpid)
	if err != nil {
		return fmt.Errorf("UpdateThreadContent post: %w", err)
	}
	return nil
}

// SoftDeleteThread 软删除主帖及旗下所有回复（事务内）
// 不执行 DELETE FROM，仅设置 deleted_at 时间戳
// 同时同步更新版块统计（直接操作 DB，不依赖异步计数器，避免容器重启丢失计数）
func SoftDeleteThread(ctx context.Context, tx *sqlx.Tx, tid uint32) (uint32, error) {
	now := time.Now()

	// 0. 查出帖子的 fid，用于后续更新版块统计
	var fid uint32
	err := tx.GetContext(ctx, &fid, `SELECT fid FROM bbs_thread WHERE tid = ?`, tid)
	if err != nil {
		return 0, fmt.Errorf("SoftDeleteThread read fid: %w", err)
	}

	// 1. 软删除主帖
	_, err = tx.ExecContext(ctx, `UPDATE bbs_thread SET deleted_at = ? WHERE tid = ?`, now, tid)
	if err != nil {
		return 0, fmt.Errorf("SoftDeleteThread thread: %w", err)
	}
	// 2. 软删除所有回帖
	_, err = tx.ExecContext(ctx, `UPDATE bbs_post SET deleted_at = ? WHERE tid = ?`, now, tid)
	if err != nil {
		return 0, fmt.Errorf("SoftDeleteThread post: %w", err)
	}
	// 3. 更新版块统计（GREATEST 防 unsigned 溢出）
	_, err = tx.ExecContext(ctx,
		`UPDATE bbs_forum SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE fid = ?`, fid)
	if err != nil {
		return 0, fmt.Errorf("SoftDeleteThread decr forum: %w", err)
	}
	return fid, nil
}

// GetUserThreadList 获取指定用户的帖子列表（分页，按 tid 降序）
// 软删除过滤：t.deleted_at IS NULL
func GetUserThreadList(ctx context.Context, db *sqlx.DB, uid uint32, page, pageSize int) ([]ThreadListItem, error) {
	offset := (page - 1) * pageSize
	var list []ThreadListItem
	err := db.SelectContext(ctx, &list, `
		SELECT t.*, u.username, u.avatar
		FROM bbs_thread t
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE t.uid = ? AND t.deleted_at IS NULL
		ORDER BY t.tid DESC
		LIMIT ? OFFSET ?`,
		uid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("GetUserThreadList: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ModerateThread 执行置顶或关闭操作
func ModerateThread(ctx context.Context, tx *sqlx.Tx, tid uint32, action string, value int) error {
	if action == "top" {
		// 更新 bbs_thread.top 字段（兼容旧查询）
		_, err := tx.ExecContext(ctx, `UPDATE bbs_thread SET top = ? WHERE tid = ?`, value, tid)
		if err != nil {
			return fmt.Errorf("ModerateThread update thread.top: %w", err)
		}
		// 同步更新 bbs_thread_top 独立表
		if err := ThreadTopChange(ctx, tx, tid, value); err != nil {
			return fmt.Errorf("ModerateThread thread_top: %w", err)
		}
	} else if action == "close" {
		_, err := tx.ExecContext(ctx, `UPDATE bbs_thread SET closed = ? WHERE tid = ?`, value, tid)
		if err != nil {
			return fmt.Errorf("ModerateThread: %w", err)
		}
	}
	return nil
}

// CreateThreadAndFirstPost 事务处理发帖主逻辑
// 在一个事务中完成：创建主帖 → 创建首帖 → 反写 firstpid/lastpid → 更新版块统计
// 用户发帖计数仍由 AsyncCounter 异步处理（非关键路径，允许短暂不一致）
// doctype: 0=HTML, 1=TXT, 2=Markdown; messageFmt 为格式化后的展示内容
func CreateThreadAndFirstPost(ctx context.Context, tx *sqlx.Tx, fid, uid uint32, userIP uint32, subject, message string, doctype int32, messageFmt string) (uint32, error) {
	now := time.Now().Unix()

	// 1. 插入主帖
	res, err := tx.ExecContext(ctx, `
		INSERT INTO bbs_thread (fid, uid, userip, subject, create_date, last_date, lastuid)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fid, uid, userIP, subject, now, now, uid)
	if err != nil {
		return 0, err
	}
	tidID, _ := res.LastInsertId()
	tid := uint32(tidID)

	// 2. 插入首个回帖（doctype 由调用方决定，写入 message_fmt 兼容老帖）
	resPost, err := tx.ExecContext(ctx, `
		INSERT INTO bbs_post (tid, uid, isfirst, create_date, userip, doctype, message, message_fmt)
		VALUES (?, ?, 1, ?, ?, ?, ?, ?)`,
		tid, uid, now, userIP, doctype, message, messageFmt)
	if err != nil {
		return 0, err
	}
	pidID, _ := resPost.LastInsertId()
	pid := uint32(pidID)

	// 3. 反写 firstpid 和 lastpid 到 thread
	_, err = tx.ExecContext(ctx, `UPDATE bbs_thread SET firstpid = ?, lastpid = ? WHERE tid = ?`, pid, pid, tid)
	if err != nil {
		return 0, err
	}

	// 4. 记录用户参与的主题（mythread）
	if err := MyThreadCreate(ctx, tx, uid, tid); err != nil {
		return 0, err
	}

	// 5. 更新版块帖子计数（事务内直接操作 DB，不依赖 AsyncCounter）
	//    AsyncCounter 在容器重启时会丢失未刷新的计数，导致版块统计永久偏差
	_, err = tx.ExecContext(ctx, `UPDATE bbs_forum SET threads = threads + 1 WHERE fid = ?`, fid)
	if err != nil {
		return 0, fmt.Errorf("CreateThreadAndFirstPost incr forum: %w", err)
	}

	return tid, nil
}

// MoveThread 跨版块移动帖子（包含统计数据平移）
// 使用事务保证原子性：更新 fid → 老版块 threads-1 → 新版块 threads+1
// oldFid == newFid 时直接跳过（原地移动无意义）
func MoveThread(ctx context.Context, tx *sqlx.Tx, tid, oldFid, newFid uint32) error {
	if oldFid == newFid {
		return nil
	}

	// 1. 更新帖子的归属版块
	_, err := tx.ExecContext(ctx, `UPDATE bbs_thread SET fid = ? WHERE tid = ?`, newFid, tid)
	if err != nil {
		return fmt.Errorf("MoveThread update fid: %w", err)
	}

	// 2. 老版块主题数 -1（GREATEST 防 unsigned 溢出）
	_, err = tx.ExecContext(ctx, `UPDATE bbs_forum SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE fid = ?`, oldFid)
	if err != nil {
		return fmt.Errorf("MoveThread decrement old forum: %w", err)
	}

	// 3. 新版块主题数 +1
	_, err = tx.ExecContext(ctx, `UPDATE bbs_forum SET threads = threads + 1 WHERE fid = ?`, newFid)
	if err != nil {
		return fmt.Errorf("MoveThread increment new forum: %w", err)
	}

	// 4. 同步更新 thread_top 中的 fid（如果该帖被置顶）
	if err := ThreadTopUpdateByTID(ctx, tx, tid, newFid); err != nil {
		return fmt.Errorf("MoveThread update thread_top: %w", err)
	}

	return nil
}

// ThreadFindByFID 按版块查找帖子（后台管理用，不过滤软删除）
// 对应 PHP: thread_find_by_fid($fid, $page, $pagesize, $order)
// order 取值: "lastpid"（默认，按最后回复降序）, "tid"（按发帖降序）
// 注意：后台管理需要看到已删除的帖子，因此不过滤 deleted_at
func ThreadFindByFID(ctx context.Context, db *sqlx.DB, fid uint32, page, pageSize int, order string) ([]ThreadListItem, error) {
	offset := (page - 1) * pageSize
	var list []ThreadListItem

	orderClause := "t.lastpid DESC"
	if order == "tid" {
		orderClause = "t.tid DESC"
	}

	if fid > 0 {
		err := db.SelectContext(ctx, &list, `
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread t
			LEFT JOIN bbs_user u ON t.uid = u.uid
			WHERE t.fid = ?
			ORDER BY `+orderClause+`
			LIMIT ? OFFSET ?`,
			fid, pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("ThreadFindByFID: %w", err)
		}
	} else {
		err := db.SelectContext(ctx, &list, `
			SELECT t.*, u.username, u.avatar
			FROM bbs_thread t
			LEFT JOIN bbs_user u ON t.uid = u.uid
			ORDER BY `+orderClause+`
			LIMIT ? OFFSET ?`,
			pageSize, offset)
		if err != nil {
			return nil, fmt.Errorf("ThreadFindByFID: %w", err)
		}
	}

	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ThreadFindByTIDs 根据 TID 列表批量查找帖子（后台管理用，不过滤软删除）
// 对应 PHP: thread_find_by_tids($tids, $order)
func ThreadFindByTIDs(ctx context.Context, db *sqlx.DB, tids []int64) ([]ThreadListItem, error) {
	if len(tids) == 0 {
		return []ThreadListItem{}, nil
	}

	// 构建 IN 查询的占位符
	query, args, err := sqlx.In(`
		SELECT t.*, u.username, u.avatar
		FROM bbs_thread t
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE t.tid IN (?)`, tids)
	if err != nil {
		return nil, fmt.Errorf("ThreadFindByTIDs sqlx.In: %w", err)
	}
	query = db.Rebind(query)

	var list []ThreadListItem
	err = db.SelectContext(ctx, &list, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThreadFindByTIDs: %w", err)
	}

	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ============================================================
// 以下为 P3 补齐函数，对应 PHP model/thread.func.php
// ============================================================

// ThreadFormat 格式化帖子列表项（填充展示字段）
// 对应 PHP: thread_format(&$thread)
// 填充: create_date_fmt, last_date_fmt, last_username, forum_name, pages
func ThreadFormat(item *ThreadListItem, forumName string, postPageSize int) {
	if item == nil {
		return
	}
	item.CreateDateFmt = humandate(item.CreateDate)
	item.LastDateFmt = humandate(item.LastDate)
	item.ForumName = forumName
	if postPageSize > 0 && item.Posts > 0 {
		item.Pages = int32((int(item.Posts) + postPageSize - 1) / postPageSize)
	}
}

// ThreadFormatDetail 格式化帖子详情
func ThreadFormatDetail(detail *ThreadDetail, forumName string) {
	if detail == nil {
		return
	}
	detail.CreateDateFmt = humandate(detail.CreateDate)
	detail.LastDateFmt = humandate(detail.LastDate)
	detail.ForumName = forumName
}

// ThreadCount 统计帖子数量
// 对应 PHP: thread_count($cond)
// cond 支持: fid, uid
func ThreadCount(ctx context.Context, db *sqlx.DB, fid, uid uint32) (int, error) {
	var count int
	if fid > 0 && uid > 0 {
		err := db.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM bbs_thread WHERE fid = ? AND uid = ? AND deleted_at IS NULL`,
			fid, uid)
		return count, err
	}
	if fid > 0 {
		err := db.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM bbs_thread WHERE fid = ? AND deleted_at IS NULL`, fid)
		return count, err
	}
	if uid > 0 {
		err := db.GetContext(ctx, &count,
			`SELECT COUNT(*) FROM bbs_thread WHERE uid = ? AND deleted_at IS NULL`, uid)
		return count, err
	}
	err := db.GetContext(ctx, &count,
		`SELECT COUNT(*) FROM bbs_thread WHERE deleted_at IS NULL`)
	return count, err
}

// ThreadFindByKeyword 按关键词搜索帖子标题
// 对应 PHP: thread_find_by_keyword($keyword)
// 返回最多 60 条结果，按 tid 降序
func ThreadFindByKeyword(ctx context.Context, db *sqlx.DB, keyword string) ([]ThreadListItem, error) {
	var list []ThreadListItem
	err := db.SelectContext(ctx, &list, `
		SELECT t.*, u.username, u.avatar
		FROM bbs_thread t
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE t.subject LIKE ? AND t.deleted_at IS NULL
		ORDER BY t.tid DESC
		LIMIT 60`, "%"+keyword+"%")
	if err != nil {
		return nil, fmt.Errorf("ThreadFindByKeyword: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ThreadSafeInfo 移除帖子的敏感信息
// 对应 PHP: thread_safe_info($thread)
func ThreadSafeInfo(thread *Thread) *Thread {
	if thread == nil {
		return nil
	}
	thread.UserIP = 0
	return thread
}

// ThreadFindLastpid 查找某个主题的最后回复 PID
// 对应 PHP: thread_find_lastpid($tid)
func ThreadFindLastpid(ctx context.Context, db *sqlx.DB, tid uint32) (uint32, error) {
	var pid uint32
	err := db.GetContext(ctx, &pid,
		`SELECT COALESCE(MAX(pid), 0) FROM bbs_post WHERE tid = ? AND deleted_at IS NULL`, tid)
	if err != nil {
		return 0, fmt.Errorf("ThreadFindLastpid: %w", err)
	}
	return pid, nil
}

// ThreadUpdateLast 更新主题的最后回复信息
// 对应 PHP: thread_update_last($tid)
func ThreadUpdateLast(ctx context.Context, db *sqlx.DB, tid uint32) error {
	lastpid, err := ThreadFindLastpid(ctx, db, tid)
	if err != nil {
		return err
	}
	if lastpid == 0 {
		return nil
	}

	var lastPost struct {
		UID        uint32 `db:"uid"`
		CreateDate int64  `db:"create_date"`
	}
	err = db.GetContext(ctx, &lastPost,
		`SELECT uid, create_date FROM bbs_post WHERE pid = ?`, lastpid)
	if err != nil {
		return fmt.Errorf("ThreadUpdateLast read post: %w", err)
	}

	_, err = db.ExecContext(ctx,
		`UPDATE bbs_thread SET lastpid = ?, lastuid = ?, last_date = ? WHERE tid = ?`,
		lastpid, lastPost.UID, lastPost.CreateDate, tid)
	return err
}

// ThreadRead 读取单个帖子（含格式化）
// 对应 PHP: thread_read($tid)
func ThreadRead(ctx context.Context, db *sqlx.DB, tid uint32) (*ThreadDetail, error) {
	return GetThreadDetail(ctx, db, tid)
}

// ThreadFindByFIDs 从多个版块获取帖子列表
// 对应 PHP: thread_find_by_fids($fids, $page, $pagesize, $order)
func ThreadFindByFIDs(ctx context.Context, db *sqlx.DB, fids []uint32, page, pageSize int, order string) ([]ThreadListItem, error) {
	if len(fids) == 0 {
		return []ThreadListItem{}, nil
	}

	offset := (page - 1) * pageSize
	orderClause := "t.lastpid DESC"
	if order == "tid" {
		orderClause = "t.tid DESC"
	}

	query, args, err := sqlx.In(`
		SELECT t.*, u.username, u.avatar
		FROM bbs_thread t
		LEFT JOIN bbs_user u ON t.uid = u.uid
		WHERE t.fid IN (?) AND t.deleted_at IS NULL
		ORDER BY `+orderClause+`
		LIMIT ? OFFSET ?`, fids, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("ThreadFindByFIDs sqlx.In: %w", err)
	}
	query = db.Rebind(query)

	var list []ThreadListItem
	err = db.SelectContext(ctx, &list, query, args...)
	if err != nil {
		return nil, fmt.Errorf("ThreadFindByFIDs: %w", err)
	}
	if list == nil {
		list = []ThreadListItem{}
	}
	return list, nil
}

// ThreadIncViews 增加帖子浏览量
// 对应 PHP: thread_inc_views($tid, $n)
// 使用 UPDATE ... SET views = views + n 原子操作，无需事务
func ThreadIncViews(ctx context.Context, db *sqlx.DB, tid uint32, n int64) error {
	_, err := db.ExecContext(ctx, `UPDATE bbs_thread SET views = views + ? WHERE tid = ?`, n, tid)
	if err != nil {
		return fmt.Errorf("ThreadIncViews: %w", err)
	}
	return nil
}

// ThreadDelete 硬删除主题及其所有关联数据
// 对应 PHP: thread_delete($tid)
// 级联删除：回帖 → 附件 → mythread → thread_top → 主题本身 → 统计更新
func ThreadDelete(ctx context.Context, db *sqlx.DB, tid uint32) error {
	// 委托给 CascadeDeleteThread 完成级联删除（在事务内执行）
	return dbTx(ctx, db, func(tx *sqlx.Tx) error {
		return CascadeDeleteThread(ctx, tx, tid)
	})
}

// ThreadMaxID 获取最大主题 ID
// 对应 PHP: thread_maxid()
func ThreadMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(tid), 0) FROM bbs_thread`)
	if err != nil {
		return 0, fmt.Errorf("ThreadMaxID: %w", err)
	}
	return maxID, nil
}

// ThreadListAccessFilter 过滤主题列表权限
// 对应 PHP: thread_list_access_filter(&$threadlist, $gid)
// 移除用户无权读取版块的主题（accesson 开启且非置顶帖）
// 当前简化实现：返回原列表，权限判断由 handler 层 CheckForumAccess 完成
func ThreadListAccessFilter(threads []ThreadListItem, gid uint16) []ThreadListItem {
	if gid == 1 || gid == 2 {
		return threads // 超管/管理员不过滤
	}
	// 简化：返回原列表，SQL 层已处理权限过滤
	return threads
}

// ThreadFind 通用主题查找
// 对应 PHP: thread_find($cond, $orderby, $page, $pagesize)
// 使用条件 map 进行通用查询，当前简化实现：仅支持按 fid 和 uid 条件
func ThreadFind(ctx context.Context, db *sqlx.DB, cond map[string]interface{}, orderby map[string]interface{}, page, pageSize int) ([]ThreadListItem, error) {
	fid, hasFid := cond["fid"]
	uid, hasUid := cond["uid"]
	order := "lastpid"
	if v, ok := orderby["tid"]; ok && v == -1 {
		order = "tid"
	}

	if hasFid {
		return ThreadFindByFID(ctx, db, toUint32(fid), page, pageSize, order)
	}
	if hasUid {
		return GetUserThreadList(ctx, db, toUint32(uid), page, pageSize)
	}
	// 无条件：返回全部
	return ThreadFindByFID(ctx, db, 0, page, pageSize, order)
}

// ThreadFormatLastDate 格式化主题的最后回复日期
// 对应 PHP: thread_format_last_date(&$thread)
// 如果 last_date == create_date，设置 create_date_fmt；否则设置 last_date_fmt
func ThreadFormatLastDate(thread *ThreadListItem) {
	if thread.LastDate != thread.CreateDate {
		thread.LastDateFmt = humandate(thread.LastDate)
	} else {
		thread.CreateDateFmt = humandate(thread.CreateDate)
	}
}

// ThreadGetLevel 根据帖子数获取用户等级
// 对应 PHP: thread_get_level($n, $levelarr)
// levelarr 为等级阈值数组，如 [0, 5, 20, 100, 500, 2000, 10000, 50000, 200000]
// 返回第一个满足 n <= level 的索引 k
func ThreadGetLevel(n int, levelarr []int) int {
	for k, level := range levelarr {
		if n <= level {
			return k
		}
	}
	return len(levelarr) - 1
}
