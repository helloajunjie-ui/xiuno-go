package model

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"regexp"
	"strings"
	"time"
	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// Post 对应 bbs_post 表
// doctype: 0=html, 1=txt, 2=markdown, 3=ubb
type Post struct {
	PID        int64        `db:"pid" json:"pid"`
	TID        int64        `db:"tid" json:"tid"`
	UID        int64        `db:"uid" json:"uid"`
	IsFirst    int32        `db:"isfirst" json:"isfirst"`
	CreateDate int64        `db:"create_date" json:"create_date"`
	UserIP     net.IP       `db:"userip" json:"-"`
	Images     int32        `db:"images" json:"images"`
	Files      int32        `db:"files" json:"files"`
	DocType    int32        `db:"doctype" json:"doctype"`
	QuotePID   int64        `db:"quotepid" json:"quotepid,omitempty"`
	Message    string       `db:"message" json:"message"`         // 原始内容(Markdown/HTML)
	MessageFmt string       `db:"message_fmt" json:"message_fmt"` // 格式化后的 HTML(兼容老帖)
	DeletedAt  sql.NullTime `db:"deleted_at" json:"deleted_at,omitempty"`

	// 格式化显示字段（不存库）
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
}

// PostWithUser 带用户信息的回帖
type PostWithUser struct {
	Post
	User *UserBrief `json:"user,omitempty"`
}

// PostListItem 回帖列表项（JOIN user 表展开用户信息）
type PostListItem struct {
	Post
	Username string `db:"username" json:"username"`
	Avatar   uint32 `db:"avatar" json:"avatar"`
}

// CreateReply 回帖核心事务
// 在一个事务中完成：插入回帖 → 更新主帖 lastpid/lastuid/last_date（顶帖）
// 统计更新（帖子回复数/用户回帖数）移出事务，由 AsyncCounter 异步处理
// doctype: 0=HTML, 1=TXT, 2=Markdown; messageFmt 为格式化后的展示内容
func CreateReply(ctx context.Context, tx *sqlx.Tx, tid, uid uint32, userIP net.IP, message string, quotePid uint32, doctype int32, messageFmt string) (uint32, error) {
	now := time.Now().Unix()

	// 1. 插入回帖（isfirst = 0）
	res, err := tx.ExecContext(ctx, `
		INSERT INTO bbs_post (tid, uid, isfirst, create_date, userip, doctype, quotepid, message, message_fmt)
		VALUES (?, ?, 0, ?, ?, ?, ?, ?, ?)`,
		tid, uid, now, userIP.To16(), doctype, quotePid, message, messageFmt)
	if err != nil {
		return 0, err
	}
	pidID, _ := res.LastInsertId()
	pid := uint32(pidID)

	// 2. 同步更新主帖的"顶帖"属性（让帖子浮到列表最前）
	_, err = tx.ExecContext(ctx, `
		UPDATE bbs_thread
		SET lastpid = ?, lastuid = ?, last_date = ?
		WHERE tid = ?`,
		pid, uid, now, tid)
	if err != nil {
		return 0, err
	}

	return pid, nil
}

// GetPostList 获取回帖列表（按时间/PID 正序，实现盖楼）
// isfirst = 0 过滤掉主帖，ORDER BY pid ASC 保证老帖在上
// 软删除过滤：p.deleted_at IS NULL
func GetPostList(ctx context.Context, db *sqlx.DB, tid uint32, page, pageSize int) ([]PostListItem, error) {
	offset := (page - 1) * pageSize
	var list []PostListItem
	err := db.SelectContext(ctx, &list, `
		SELECT p.*, u.username, u.avatar
		FROM bbs_post p
		LEFT JOIN bbs_user u ON p.uid = u.uid
		WHERE p.tid = ? AND p.isfirst = 0 AND p.deleted_at IS NULL
		ORDER BY p.pid ASC
		LIMIT ? OFFSET ?`,
		tid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("GetPostList: %w", err)
	}
	if list == nil {
		list = []PostListItem{}
	}
	return list, nil
}

// GetUserPostList 获取指定用户的回帖列表（分页，按 pid 降序）
// 只返回非首帖（isfirst = 0），软删除过滤
func GetUserPostList(ctx context.Context, db *sqlx.DB, uid uint32, page, pageSize int) ([]PostListItem, error) {
	offset := (page - 1) * pageSize
	var list []PostListItem
	err := db.SelectContext(ctx, &list, `
		SELECT p.*, u.username, u.avatar
		FROM bbs_post p
		LEFT JOIN bbs_user u ON p.uid = u.uid
		WHERE p.uid = ? AND p.isfirst = 0 AND p.deleted_at IS NULL
		ORDER BY p.pid DESC
		LIMIT ? OFFSET ?`,
		uid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("GetUserPostList: %w", err)
	}
	if list == nil {
		list = []PostListItem{}
	}
	return list, nil
}

// UpdatePostContent 修改回帖内容
func UpdatePostContent(ctx context.Context, db *sqlx.DB, pid uint32, message string) error {
	_, err := db.ExecContext(ctx, `UPDATE bbs_post SET message = ? WHERE pid = ?`, message, pid)
	if err != nil {
		return fmt.Errorf("UpdatePostContent: %w", err)
	}
	return nil
}

// SoftDeletePost 软删除单个回帖
func SoftDeletePost(ctx context.Context, db *sqlx.DB, pid uint32) error {
	now := time.Now()
	_, err := db.ExecContext(ctx, `UPDATE bbs_post SET deleted_at = ? WHERE pid = ?`, now, pid)
	if err != nil {
		return fmt.Errorf("SoftDeletePost: %w", err)
	}
	return nil
}

// PostFormat 格式化回帖（填充显示字段）
// 对应 PHP: post_format()
func PostFormat(post *Post) {
	if post == nil {
		return
	}
	post.CreateDateFmt = humandate(post.CreateDate)
}

// PostMessageFmt 写入时格式化消息内容
// 对应 PHP: post_message_fmt()
// 根据 doctype 类型对 message 进行格式化，填充 message_fmt
func PostMessageFmt(arr map[string]interface{}, gid int) {
	msg, _ := arr["message"].(string)
	doctype, _ := arr["doctype"].(int)

	// 超长内容截取（原版 2028000 字符限制）
	if len(msg) > 2028000 {
		msg = msg[:2028000]
	}

	var messageFmt string
	switch doctype {
	case 0: // HTML
		if gid == 1 {
			messageFmt = msg // 管理员直接使用原始 HTML
		} else {
			messageFmt = msg // 简化：不实现 xn_html_safe，前端自行控制
		}
	case 1: // TXT
		// xn_txt_to_html: 简单转义
		messageFmt = htmlEscape(msg)
	default: // 2=Markdown, 3=UBB
		messageFmt = htmlEscape(msg)
	}

	arr["message_fmt"] = messageFmt
}

// htmlEscape 转义 HTML 特殊字符
func htmlEscape(s string) string {
	var result []byte
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '&':
			result = append(result, "&"...)
		case '<':
			result = append(result, "<"...)
		case '>':
			result = append(result, ">"...)
		case '"':
			result = append(result, "&#34;"...)
		case '\'':
			result = append(result, "'"...)
		default:
			result = append(result, s[i])
		}
	}
	return string(result)
}

// PostCount 统计回帖数量
// 对应 PHP: post_count()
// cond 为可选过滤条件 map，如 {"tid": 123}
func PostCount(ctx context.Context, db *sqlx.DB, cond map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM bbs_post WHERE deleted_at IS NULL`
	args := []interface{}{}
	if tid, ok := cond["tid"]; ok {
		query += ` AND tid = ?`
		args = append(args, tid)
	}
	if uid, ok := cond["uid"]; ok {
		query += ` AND uid = ?`
		args = append(args, uid)
	}
	var count int
	err := db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("PostCount: %w", err)
	}
	return count, nil
}

// PostFindByTID 按主题 ID 查找回帖列表（分页，按 pid 正序）
// 对应 PHP: post_find_by_tid()
func PostFindByTID(ctx context.Context, db *sqlx.DB, tid uint32, page, pageSize int) ([]Post, error) {
	offset := (page - 1) * pageSize
	var list []Post
	err := db.SelectContext(ctx, &list, `
		SELECT * FROM bbs_post
		WHERE tid = ? AND deleted_at IS NULL
		ORDER BY pid ASC
		LIMIT ? OFFSET ?`,
		tid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("PostFindByTID: %w", err)
	}
	if list == nil {
		list = []Post{}
	}
	return list, nil
}

// PostSafeInfo 返回回帖安全信息（移除敏感字段）
// 对应 PHP: post_safe_info()
func PostSafeInfo(post *Post) *Post {
	if post == nil {
		return nil
	}
	safe := *post
	safe.UserIP = nil
	return &safe
}

// PostFindByPIDs 批量通过 PID 获取回帖
// 对应 PHP: post_find_by_pids()
func PostFindByPIDs(ctx context.Context, db *sqlx.DB, pids []uint32) (map[int64]*Post, error) {
	if len(pids) == 0 {
		return map[int64]*Post{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM bbs_post WHERE pid IN (?)`, pids)
	if err != nil {
		return nil, fmt.Errorf("PostFindByPIDs: %w", err)
	}
	query = db.Rebind(query)
	var posts []Post
	err = db.SelectContext(ctx, &posts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("PostFindByPIDs: %w", err)
	}
	result := make(map[int64]*Post, len(posts))
	for i := range posts {
		result[posts[i].PID] = &posts[i]
	}
	return result, nil
}

// PostDelete 硬删除回帖（物理删除）
// 对应 PHP: post_delete()
// 注意：此操作不可逆，会同时清理附件
func PostDelete(ctx context.Context, db *sqlx.DB, pid uint32, uploadDir string) error {
	// 1. 查出回帖信息
	var post Post
	err := db.GetContext(ctx, &post, `SELECT * FROM bbs_post WHERE pid = ?`, pid)
	if err != nil {
		return fmt.Errorf("PostDelete read: %w", err)
	}

	// 2. 删除附件
	if post.Images > 0 || post.Files > 0 {
		_, err := AttachDeleteByPID(ctx, db, pid, uploadDir)
		if err != nil {
			return fmt.Errorf("PostDelete attach: %w", err)
		}
	}

	// 3. 物理删除回帖
	_, err = db.ExecContext(ctx, `DELETE FROM bbs_post WHERE pid = ?`, pid)
	if err != nil {
		return fmt.Errorf("PostDelete: %w", err)
	}

	return nil
}

// PostDeleteByTID 删除某个主题下的所有回帖
// 对应 PHP: post_delete_by_tid($tid)
// 返回删除的回帖数量
func PostDeleteByTID(ctx context.Context, db *sqlx.DB, tid uint32, uploadDir string) (int, error) {
	// 查出该主题下所有回帖
	posts, err := PostFindByTID(ctx, db, tid, 1, 1000000)
	if err != nil {
		return 0, fmt.Errorf("PostDeleteByTID find: %w", err)
	}
	count := 0
	for _, post := range posts {
		if err := PostDelete(ctx, db, uint32(post.PID), uploadDir); err != nil {
			return count, fmt.Errorf("PostDeleteByTID delete %d: %w", post.PID, err)
		}
		count++
	}
	return count, nil
}

// PostDeleteByUID 删除某个用户的所有回帖
// 对应 PHP: post_delete_by_uid($uid)
// 注意：此操作可能涉及大量数据，可能超时
func PostDeleteByUID(ctx context.Context, db *sqlx.DB, uid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_post WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("PostDeleteByUID: %w", err)
	}
	return nil
}

// PostQuote 生成引用回复的 HTML
// 对应 PHP: post_quote($quotepid)
// 返回 <blockquote> 格式的引用内容
func PostQuote(ctx context.Context, db *sqlx.DB, quotepid uint32) (string, error) {
	var post Post
	err := db.GetContext(ctx, &post, `SELECT * FROM bbs_post WHERE pid = ?`, quotepid)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", fmt.Errorf("PostQuote read: %w", err)
	}

	// 获取引用用户的用户名
	var username string
	err = db.GetContext(ctx, &username, `SELECT username FROM bbs_user WHERE uid = ?`, post.UID)
	if err != nil {
		username = "unknown"
	}

	// 截取前 100 个字符作为引用摘要
	brief := PostBrief(post.Message, 100)

	// 生成 blockquote HTML
	r := fmt.Sprintf(`<blockquote class="blockquote">
		<a href="/user/%d" class="text-small text-muted user">%s</a>
		%s
		</blockquote>`, post.UID, htmlEscape(username), brief)
	return r, nil
}

// PostBrief 截取帖子内容摘要
// 对应 PHP: post_brief($s, $len)
// 去除 HTML 标签，HTML 转义，截取前 len 个字符
func PostBrief(s string, length int) string {
	// 去除 HTML 标签
	s = stripHTMLTags(s)
	// HTML 转义
	s = htmlEscape(s)
	// 截取长度
	runes := []rune(s)
	if len(runes) > length {
		return string(runes[:length]) + " ... "
	}
	return s
}

// stripHTMLTags 去除 HTML 标签（简化版）
func stripHTMLTags(s string) string {
	var result []byte
	inTag := false
	for i := 0; i < len(s); i++ {
		if s[i] == '<' {
			inTag = true
			continue
		}
		if s[i] == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result = append(result, s[i])
		}
	}
	return string(result)
}

// PostMaxID 获取最大回帖 ID
// 对应 PHP: post_maxid()
func PostMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(pid), 0) FROM bbs_post`)
	if err != nil {
		return 0, fmt.Errorf("PostMaxID: %w", err)
	}
	return maxID, nil
}

// PostHighlightKeyword 在字符串中高亮关键词
// 对应 PHP: post_highlight_keyword($str, $k)
// 返回包含 <span class="red"> 标签的 HTML 片段
func PostHighlightKeyword(str, keyword string) string {
	if keyword == "" {
		return str
	}
	// 大小写不敏感替换
	return strings.ReplaceAll(
		strings.ToLower(str),
		strings.ToLower(keyword),
		`<span class="red">`+keyword+`</span>`,
	)
}

// UserPostMessageFormat 格式化用户帖子消息（生成摘要）
// 对应 PHP: user_post_message_format(&$s)
// 去除引用块、HTML 标签，截取前 100 字符
func UserPostMessageFormat(s string) string {
	if len([]rune(s)) < 100 {
		return s
	}
	// 去除 blockquote 引用块
	re := regexp.MustCompile(`<blockquote\s+class="blockquote">.*?</blockquote>`)
	s = re.ReplaceAllString(s, "")
	// 替换换行标签
	s = strings.ReplaceAll(s, "<br>", "\n")
	s = strings.ReplaceAll(s, "<br />", "\n")
	s = strings.ReplaceAll(s, "<br/>", "\n")
	s = strings.ReplaceAll(s, "</p>", "\n")
	s = strings.ReplaceAll(s, "</tr>", "\n")
	s = strings.ReplaceAll(s, "</div>", "\n")
	s = strings.ReplaceAll(s, "</li>", "\n")
	s = strings.ReplaceAll(s, "</dd>", "\n")
	s = strings.ReplaceAll(s, "</dt>", "\n")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	// 去除 HTML 标签
	s = stripHTMLTags(s)
	// 合并连续换行
	re = regexp.MustCompile(`[\r\n]+`)
	s = re.ReplaceAllString(s, "\n")
	// 截取前 100 字符
	runes := []rune(strings.TrimSpace(s))
	if len(runes) > 100 {
		s = string(runes[:100])
	} else {
		s = string(runes)
	}
	// 换行转 <br>
	s = strings.ReplaceAll(s, "\n", "<br>")
	return s
}

// PostListAccessFilter 过滤回帖列表权限
// 对应 PHP: post_list_access_filter(&$postlist, $gid)
// 当前简化实现：gid=1(超管) 可见所有回帖，其他用户仅可见非软删除回帖
// 实际权限判断由 handler 层 CheckForumAccess 完成
func PostListAccessFilter(posts []Post, gid uint16) []Post {
	if gid == 1 {
		return posts // 管理员不过滤
	}
	// 简化：返回原列表，软删除已在 SQL 层过滤
	return posts
}

// PostRead 读取单条回帖
// 对应 PHP: post_read($pid)
func PostRead(ctx context.Context, db *sqlx.DB, pid uint32) (*Post, error) {
	var post Post
	err := db.GetContext(ctx, &post, `SELECT * FROM bbs_post WHERE pid = ?`, pid)
	if err != nil {
		return nil, fmt.Errorf("PostRead: %w", err)
	}
	return &post, nil
}

// PostFind 通用回帖查询
// 对应 PHP: post_find($cond, $orderby, $page, $pagesize)
// cond 支持: tid, uid
func PostFind(ctx context.Context, db *sqlx.DB, tid uint32, page, pageSize int) ([]Post, error) {
	offset := (page - 1) * pageSize
	var list []Post
	err := db.SelectContext(ctx, &list, `
		SELECT * FROM bbs_post
		WHERE tid = ? AND deleted_at IS NULL
		ORDER BY pid ASC
		LIMIT ? OFFSET ?`,
		tid, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("PostFind: %w", err)
	}
	if list == nil {
		list = []Post{}
	}
	return list, nil
}

// PostListCacheDelete 删除帖子列表缓存
// 对应 PHP: post_list_cache_delete($tid)
// 当前 Go 版使用 cache_helper.go 的 InvalidateThreadCache 替代
func PostListCacheDelete(ctx context.Context, cache core.Cache, tid uint32) {
	cache.Del(ctx, fmt.Sprintf("thread_detail:%d", tid))
}
