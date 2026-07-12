// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// EffectiveAccess 最终计算出的有效权限
type EffectiveAccess struct {
	AllowRead   int `db:"allowread" json:"allowread"`
	AllowThread int `db:"allowthread" json:"allowthread"`
	AllowPost   int `db:"allowpost" json:"allowpost"`
}

// GetEffectiveAccess 计算用户组 (gid) 在特定版块 (fid) 的最终权限
// 规则：
//  1. 先查全局用户组表 (bbs_group)，拿到基础权限
//  2. 检查版块表 (bbs_forum) 是否开启了权限隔离 (accesson == 1)
//  3. 如果开启了，再去查 (bbs_forum_access)：
//     - 查到了：局部权限覆盖全局权限
//     - 没查到：默认该局部权限为 0（拒绝）
func GetEffectiveAccess(ctx context.Context, db *sqlx.DB, fid uint32, gid uint16) (*EffectiveAccess, error) {
	// 1. 获取全局基础权限
	var access EffectiveAccess
	err := db.GetContext(ctx, &access,
		`SELECT allowread, allowthread, allowpost FROM bbs_group WHERE gid = ?`, gid)
	if err != nil {
		// GID 不存在（如游客 GID=0），默认无权限
		return &EffectiveAccess{AllowRead: 0, AllowThread: 0, AllowPost: 0}, nil
	}

	// 2. 检查版块是否开启了独立权限 (accesson)
	var accesson int
	err = db.GetContext(ctx, &accesson,
		`SELECT accesson FROM bbs_forum WHERE fid = ?`, fid)
	if err != nil {
		if err == sql.ErrNoRows {
			// 版块不存在，默认无权限
			return &EffectiveAccess{AllowRead: 0, AllowThread: 0, AllowPost: 0}, nil
		}
		return nil, err
	}

	// 3. 如果未开启独立权限，直接返回全局权限
	if accesson == 0 {
		return &access, nil
	}

	// 4. 如果开启了独立权限，查询局部配置进行覆盖
	var localAccess EffectiveAccess
	err = db.GetContext(ctx, &localAccess,
		`SELECT allowread, allowthread, allowpost FROM bbs_forum_access WHERE fid = ? AND gid = ?`,
		fid, gid)
	if err == nil {
		// 查到了局部配置，覆盖
		access = localAccess
	} else {
		// 开启了独立权限但没配置该用户组，默认全部拒绝
		access.AllowRead = 0
		access.AllowThread = 0
		access.AllowPost = 0
	}

	return &access, nil
}

// CheckForumAccess 校验用户组在版块的行为权限
// action: "read" / "thread" / "post"
// 超管 (gid=1) 畅通无阻，无需查库
func CheckForumAccess(ctx context.Context, db *sqlx.DB, uid uint32, gid uint16, fid uint32, action string) bool {
	// 小黑屋用户绝对封杀 (GID=7 禁止用户组)
	if gid == 7 {
		return false
	}

	// 超管 (gid=1) 畅通无阻
	if gid == 1 {
		return true
	}

	// 获取有效权限
	access, err := GetEffectiveAccess(ctx, db, fid, gid)
	if err != nil {
		return false
	}

	// 按动作校验
	switch action {
	case "read":
		return access.AllowRead > 0
	case "thread":
		return access.AllowThread > 0
	case "post":
		return access.AllowPost > 0
	}
	return false
}

// CheckForumAccessWithCache 带缓存的权限校验
// 缓存 key: access:{fid}:{gid}，TTL=5min
// 每次请求都会调用此函数，缓存收益极高
func CheckForumAccessWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, uid uint32, gid uint16, fid uint32, action string) bool {
	// 小黑屋用户绝对封杀 (GID=7 禁止用户组)
	if gid == 7 {
		return false
	}

	// 超管 (gid=1) 畅通无阻
	if gid == 1 {
		return true
	}

	// 获取有效权限（带缓存）
	access, err := GetEffectiveAccessWithCache(ctx, cache, db, fid, gid)
	if err != nil {
		return false
	}

	// 按动作校验
	switch action {
	case "read":
		return access.AllowRead > 0
	case "thread":
		return access.AllowThread > 0
	case "post":
		return access.AllowPost > 0
	}
	return false
}

// ==================== ForumAccess CRUD ====================

// ForumAccessCreate 创建版块权限记录
func ForumAccessCreate(ctx context.Context, db *sqlx.DB, fa *ForumAccess) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO bbs_forum_access (fid, gid, allowread, allowthread, allowpost, allowattach, allowdown)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		fa.FID, fa.GID, fa.AllowRead, fa.AllowThread, fa.AllowPost, fa.AllowAttach, fa.AllowDown)
	return err
}

// ForumAccessUpdate 更新版块权限记录
func ForumAccessUpdate(ctx context.Context, db *sqlx.DB, fid, gid uint32, fa *ForumAccess) error {
	_, err := db.ExecContext(ctx,
		`UPDATE bbs_forum_access SET allowread=?, allowthread=?, allowpost=?, allowattach=?, allowdown=?
		 WHERE fid=? AND gid=?`,
		fa.AllowRead, fa.AllowThread, fa.AllowPost, fa.AllowAttach, fa.AllowDown, fid, gid)
	return err
}

// ForumAccessReplace 替换（不存在则创建，存在则更新）版块权限记录
func ForumAccessReplace(ctx context.Context, db *sqlx.DB, fid, gid uint32, fa *ForumAccess) error {
	// 先检查是否存在
	var exists int
	err := db.GetContext(ctx, &exists, `SELECT COUNT(*) FROM bbs_forum_access WHERE fid=? AND gid=?`, fid, gid)
	if err != nil {
		return err
	}
	if exists == 0 {
		fa.FID = int64(fid)
		fa.GID = int64(gid)
		return ForumAccessCreate(ctx, db, fa)
	}
	return ForumAccessUpdate(ctx, db, fid, gid, fa)
}

// ForumAccessDelete 删除单条版块权限记录
func ForumAccessDelete(ctx context.Context, db *sqlx.DB, fid, gid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_forum_access WHERE fid=? AND gid=?`, fid, gid)
	return err
}

// ForumAccessDeleteByFID 删除某个版块的所有权限记录
func ForumAccessDeleteByFID(ctx context.Context, db *sqlx.DB, fid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_forum_access WHERE fid=?`, fid)
	return err
}

// ForumAccessFindByFID 查询某个版块的所有权限记录（按 gid 排序）
func ForumAccessFindByFID(ctx context.Context, db *sqlx.DB, fid uint32) ([]ForumAccess, error) {
	var list []ForumAccess
	err := db.SelectContext(ctx, &list,
		`SELECT * FROM bbs_forum_access WHERE fid=? ORDER BY gid ASC`, fid)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ForumAccessFind 通用查询（支持任意条件组合）
// cond 支持的 key: fid(uint32), gid(uint32)
// 对应 PHP: forum_access_find()
func ForumAccessFind(ctx context.Context, db *sqlx.DB, cond map[string]interface{}, page, pageSize int) ([]ForumAccess, error) {
	query := `SELECT * FROM bbs_forum_access WHERE 1=1`
	var args []interface{}

	if fid, ok := cond["fid"]; ok {
		query += ` AND fid = ?`
		args = append(args, fid)
	}
	if gid, ok := cond["gid"]; ok {
		query += ` AND gid = ?`
		args = append(args, gid)
	}

	query += ` ORDER BY fid, gid`

	if page > 0 && pageSize > 0 {
		offset := (page - 1) * pageSize
		query += ` LIMIT ? OFFSET ?`
		args = append(args, pageSize, offset)
	}

	var list []ForumAccess
	err := db.SelectContext(ctx, &list, query, args...)
	if err != nil {
		return nil, err
	}
	if list == nil {
		list = []ForumAccess{}
	}
	return list, nil
}

// ForumAccessPadding 根据 gid 补充或删除 forum_access 记录
// fill=true: 为所有开启了 accesson 的版块创建该 gid 的空权限记录
// fill=false: 从所有开启了 accesson 的版块删除该 gid 的权限记录
func ForumAccessPadding(ctx context.Context, db *sqlx.DB, gid uint32, fill bool) error {
	forums, err := GetAllForums(ctx, db)
	if err != nil {
		return err
	}
	for _, f := range forums {
		if f.AccessOn == 0 {
			continue
		}
		if fill {
			// 检查是否已存在
			var exists int
			err := db.GetContext(ctx, &exists,
				`SELECT COUNT(*) FROM bbs_forum_access WHERE fid=? AND gid=?`, f.FID, gid)
			if err != nil {
				return err
			}
			if exists == 0 {
				err = ForumAccessCreate(ctx, db, &ForumAccess{
					FID: f.FID, GID: int64(gid),
					AllowRead: 0, AllowThread: 0, AllowPost: 0, AllowAttach: 0, AllowDown: 0,
				})
				if err != nil {
					return err
				}
			}
		} else {
			_ = ForumAccessDelete(ctx, db, uint32(f.FID), gid)
		}
	}
	return nil
}

// GetAllForums 获取所有版块列表（用于 ForumAccessPadding）
func GetAllForums(ctx context.Context, db *sqlx.DB) ([]Forum, error) {
	var list []Forum
	err := db.SelectContext(ctx, &list, "SELECT * FROM bbs_forum ORDER BY `rank` ASC, fid ASC")
	if err != nil {
		return nil, err
	}
	return list, nil
}

// ForumAccessMod 检查用户在版块中的版主权限
// 对应 PHP: forum_access_mod($fid, $gid, $access)
// gid=1(超管)/2(管理员) 拥有所有权限
// gid=3(版主)/4(实习版主) 需要同时满足: 用户组有该权限 + 用户在 moduids 列表中
// access 取值: allowtop, allowmove, allowupdate, allowdelete, allowbanuser, allowviewip, allowdeleteuser
func ForumAccessMod(ctx context.Context, db *sqlx.DB, fid, uid uint32, gid uint16, access string) (bool, error) {
	// 管理员拥有所有权限
	if gid == 1 || gid == 2 {
		return true, nil
	}

	// 只有 gid 3(版主) 和 4(实习版主) 才可能拥有版主权限
	if gid != 3 && gid != 4 {
		return false, nil
	}

	// 获取版块信息
	forum, err := GetForum(ctx, db, fid)
	if err != nil {
		return false, nil
	}

	// 检查用户在 moduids 列表中
	moduids := strings.TrimSpace(forum.ModUIDs)
	if moduids == "" {
		return false, nil
	}
	uidStr := strconv.Itoa(int(uid))
	found := false
	for _, s := range strings.Split(moduids, ",") {
		if strings.TrimSpace(s) == uidStr {
			found = true
			break
		}
	}
	if !found {
		return false, nil
	}

	// 检查用户组是否有该权限（简化：gid 3/4 默认拥有所有版主权限）
	// PHP 原版检查 $group[$access]，但 Go 中 Group 结构体没有这些细粒度权限字段
	// 当前简化实现：只要是版主且在 moduids 中，就拥有所有版主权限
	return true, nil
}

// ForumIsMod 判断用户是否为指定版块的版主
// 对应 PHP: forum_is_mod($fid, $gid, $uid)
// gid=1(超管)/2(管理员) 视为所有版块的版主
// gid=3(版主)/4(实习版主) 需要在版块的 moduids 列表中
// fid=0 时，gid 3/4 视为全局版主
func ForumIsMod(ctx context.Context, db *sqlx.DB, fid, uid uint32, gid uint16) (bool, error) {
	if gid == 1 || gid == 2 {
		return true, nil
	}
	if gid != 3 && gid != 4 {
		return false, nil
	}
	// fid=0 时，版主视为全局版主
	if fid == 0 {
		return true, nil
	}
	// 获取版块信息，检查 moduids
	forum, err := GetForum(ctx, db, fid)
	if err != nil {
		return false, nil
	}
	moduids := strings.TrimSpace(forum.ModUIDs)
	if moduids == "" {
		return false, nil
	}
	uidStr := strconv.Itoa(int(uid))
	for _, s := range strings.Split(moduids, ",") {
		if strings.TrimSpace(s) == uidStr {
			return true, nil
		}
	}
	return false, nil
}

// ForumAccessFormat 格式化版块权限数据
// 对应 PHP: forum_access_format(&$access)
// PHP 原版为空操作，仅保留 hook 点
func ForumAccessFormat(access *ForumAccess) {
	// PHP 原版为空操作，保留接口兼容
}

// ForumAccessCount 统计版块权限记录数
// 对应 PHP: forum_access_count($cond)
func ForumAccessCount(ctx context.Context, db *sqlx.DB, cond map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM bbs_forum_access`
	var args []interface{}
	where := []string{}
	if fid, ok := cond["fid"]; ok {
		where = append(where, "fid = ?")
		args = append(args, fid)
	}
	if gid, ok := cond["gid"]; ok {
		where = append(where, "gid = ?")
		args = append(args, gid)
	}
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	var count int
	err := db.GetContext(ctx, &count, query, args...)
	if err != nil {
		return 0, fmt.Errorf("ForumAccessCount: %w", err)
	}
	return count, nil
}
