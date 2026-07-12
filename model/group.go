// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
)

// Group 对应 bbs_group 表
type Group struct {
	GID             int64  `db:"gid" json:"gid"`
	Name            string `db:"name" json:"name"`
	CreditsFrom     int64  `db:"creditsfrom" json:"creditsfrom"`
	CreditsTo       int64  `db:"creditsto" json:"creditsto"`
	AllowRead       int64  `db:"allowread" json:"allowread"`
	AllowThread     int64  `db:"allowthread" json:"allowthread"`
	AllowPost       int64  `db:"allowpost" json:"allowpost"`
	AllowAttach     int64  `db:"allowattach" json:"allowattach"`
	AllowDown       int64  `db:"allowdown" json:"allowdown"`
	AllowTop        int64  `db:"allowtop" json:"allowtop"`
	AllowUpdate     int64  `db:"allowupdate" json:"allowupdate"`
	AllowDelete     int64  `db:"allowdelete" json:"allowdelete"`
	AllowMove       int64  `db:"allowmove" json:"allowmove"`
	AllowBanUser    int64  `db:"allowbanuser" json:"allowbanuser"`
	AllowDeleteUser int64  `db:"allowdeleteuser" json:"allowdeleteuser"`
	AllowViewIP     int64  `db:"allowviewip" json:"allowviewip"`
}

// GetAllGroups 获取所有用户组
// 用于用户组自动升级时的全量遍历
func GetAllGroups(ctx context.Context, db *sqlx.DB) ([]Group, error) {
	var groups []Group
	err := db.SelectContext(ctx, &groups, `SELECT * FROM bbs_group ORDER BY gid ASC`)
	if err != nil {
		return nil, fmt.Errorf("GetAllGroups: %w", err)
	}
	if groups == nil {
		groups = []Group{}
	}
	return groups, nil
}

// GetGroup 获取单个用户组
// 对应 PHP: group_read()
func GetGroup(ctx context.Context, db *sqlx.DB, gid uint32) (*Group, error) {
	var g Group
	err := db.GetContext(ctx, &g, `SELECT * FROM bbs_group WHERE gid = ?`, gid)
	if err != nil {
		return nil, fmt.Errorf("GetGroup: %w", err)
	}
	return &g, nil
}

// CreateGroup 创建用户组
// 对应 PHP: group_create()
// 注意：bbs_group.gid 不是 auto_increment，需要手动计算下一个 gid
func CreateGroup(ctx context.Context, db *sqlx.DB, g *Group) error {
	// 获取当前最大 gid
	maxID, err := GroupMaxID(ctx, db)
	if err != nil {
		return fmt.Errorf("CreateGroup: %w", err)
	}
	nextGID := maxID + 1

	_, err = db.ExecContext(ctx, `
		INSERT INTO bbs_group (gid, name, creditsfrom, creditsto,
			allowread, allowthread, allowpost, allowattach, allowdown,
			allowtop, allowupdate, allowdelete, allowmove,
			allowbanuser, allowdeleteuser, allowviewip)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		nextGID, g.Name, g.CreditsFrom, g.CreditsTo,
		g.AllowRead, g.AllowThread, g.AllowPost, g.AllowAttach, g.AllowDown,
		g.AllowTop, g.AllowUpdate, g.AllowDelete, g.AllowMove,
		g.AllowBanUser, g.AllowDeleteUser, g.AllowViewIP)
	if err != nil {
		return fmt.Errorf("CreateGroup: %w", err)
	}
	g.GID = int64(nextGID)
	return nil
}

// UpdateGroup 更新用户组
// 对应 PHP: group_update()
func UpdateGroup(ctx context.Context, db *sqlx.DB, gid uint32, g *Group) error {
	_, err := db.ExecContext(ctx, `
		UPDATE bbs_group SET
			name = ?, creditsfrom = ?, creditsto = ?,
			allowread = ?, allowthread = ?, allowpost = ?,
			allowattach = ?, allowdown = ?,
			allowtop = ?, allowupdate = ?, allowdelete = ?, allowmove = ?,
			allowbanuser = ?, allowdeleteuser = ?, allowviewip = ?
		WHERE gid = ?`,
		g.Name, g.CreditsFrom, g.CreditsTo,
		g.AllowRead, g.AllowThread, g.AllowPost, g.AllowAttach, g.AllowDown,
		g.AllowTop, g.AllowUpdate, g.AllowDelete, g.AllowMove,
		g.AllowBanUser, g.AllowDeleteUser, g.AllowViewIP,
		gid)
	if err != nil {
		return fmt.Errorf("UpdateGroup: %w", err)
	}
	return nil
}

// DeleteGroup 删除用户组
// 对应 PHP: group_delete()
// 注意：需要确保没有用户属于该组，否则外键约束会失败
func DeleteGroup(ctx context.Context, db *sqlx.DB, gid uint32) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_group WHERE gid = ?`, gid)
	if err != nil {
		return fmt.Errorf("DeleteGroup: %w", err)
	}
	return nil
}

// AutoUpdateUserGroup 根据发帖数自动升级用户组
// 对应 PHP: model/user.func.php user_update_group()
//
// 规则：
//   - 仅对 gid >= 100 的普通用户组生效（管理员/版主等固定组不自动升降）
//   - 根据 posts + threads 发帖总数，匹配 creditsfrom ~ creditsto 区间
//   - 如果当前 gid 已匹配则跳过，避免无谓的 UPDATE
func AutoUpdateUserGroup(ctx context.Context, db *sqlx.DB, uid uint32) {
	// 1. 查出用户当前信息
	var user struct {
		GID     int64 `db:"gid"`
		Threads int64 `db:"threads"`
		Posts   int64 `db:"posts"`
	}
	err := db.GetContext(ctx, &user, `SELECT gid, threads, posts FROM bbs_user WHERE uid = ?`, uid)
	if err != nil {
		log.Printf("[WARN] UpdateUserGroup read user %d: %v", uid, err)
		return
	}

	// 2. 仅对普通用户组（gid >= 100）执行自动升级
	if user.GID < 100 {
		return
	}

	// 3. 查出所有用户组
	groups, err := GetAllGroups(ctx, db)
	if err != nil {
		log.Printf("[WARN] UpdateUserGroup get groups: %v", err)
		return
	}

	// 4. 计算发帖总数
	total := user.Threads + user.Posts

	// 5. 遍历匹配
	for _, g := range groups {
		if g.GID < 100 {
			continue // 跳过管理组
		}
		if total > g.CreditsFrom && total < g.CreditsTo {
			if user.GID != g.GID {
				_, err := db.ExecContext(ctx, `UPDATE bbs_user SET gid = ? WHERE uid = ?`, g.GID, uid)
				if err != nil {
					log.Printf("[WARN] UpdateUserGroup update uid=%d to gid=%d: %v", uid, g.GID, err)
				} else {
					log.Printf("[INFO] 用户 %d 自动升级到用户组 %d (发帖数: %d)", uid, g.GID, total)
				}
			}
			return
		}
	}
}

// GroupFormat 格式化用户组（目前仅占位，原版无实质格式化逻辑）
// 对应 PHP: group_format()
func GroupFormat(group *Group) {
	// 原版 PHP 中 group_format() 为空操作
}

// GroupName 根据 gid 获取用户组名称
// 对应 PHP: group_name()
func GroupName(ctx context.Context, db *sqlx.DB, gid uint32) (string, error) {
	var name string
	err := db.GetContext(ctx, &name, `SELECT name FROM bbs_group WHERE gid = ?`, gid)
	if err != nil {
		return "", fmt.Errorf("GroupName: %w", err)
	}
	return name, nil
}

// GroupCount 统计用户组数量
// 对应 PHP: group_count()
func GroupCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_group`)
	if err != nil {
		return 0, fmt.Errorf("GroupCount: %w", err)
	}
	return count, nil
}

// GroupListCache 获取用户组列表缓存（全量）
// 对应 PHP: group_list_cache()
// 返回 map[gid]*Group，方便前端快速查找
func GroupListCache(ctx context.Context, db *sqlx.DB) (map[int64]*Group, error) {
	groups, err := GetAllGroups(ctx, db)
	if err != nil {
		return nil, err
	}
	result := make(map[int64]*Group, len(groups))
	for i := range groups {
		result[groups[i].GID] = &groups[i]
	}
	return result, nil
}

// GroupListCacheDelete 清除用户组列表缓存
// 对应 PHP: group_list_cache_delete()
// 当前无独立缓存层，直接调用 GetAllGroups 重新查询
func GroupListCacheDelete() {
	// 无缓存需要清除，保留作为接口兼容
}

// GroupMaxID 获取最大用户组 ID
// 对应 PHP: group_maxid()
func GroupMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(gid), 0) FROM bbs_group`)
	if err != nil {
		return 0, fmt.Errorf("GroupMaxID: %w", err)
	}
	return maxID, nil
}
