package model

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// PlatID 第三方平台编号
type PlatID uint8

const (
	PlatLocal  PlatID = 0 // 本站
	PlatQQ     PlatID = 1 // QQ 登录
	PlatWechat PlatID = 2 // 微信登录
	PlatAlipay PlatID = 3 // 支付宝登录
)

// UserOpenPlat 第三方平台用户绑定
// 对应 PHP: bbs_user_open_plat 表
// 用于 SSO/OAuth2 第三方登录
type UserOpenPlat struct {
	UID    uint32 `db:"uid" json:"uid"`       // 用户编号（PK）
	PlatID PlatID `db:"platid" json:"platid"` // 平台编号
	OpenID string `db:"openid" json:"openid"` // 第三方唯一标识
}

// UserOpenPlatBind 第三方绑定信息（含用户信息）
type UserOpenPlatBind struct {
	UserOpenPlat
	Username  string `db:"username" json:"username"`
	AvatarURL string `db:"avatar_url" json:"avatar_url,omitempty"`
}

// UserOpenFindByOpenID 根据 openid 查找绑定记录
// 对应 PHP: db_find_one('user_open_plat', array('openid'=>$openid))
func UserOpenFindByOpenID(ctx context.Context, db *sqlx.DB, platID PlatID, openID string) (*UserOpenPlat, error) {
	var uop UserOpenPlat
	err := db.GetContext(ctx, &uop,
		`SELECT uid, platid, openid FROM bbs_user_open_plat WHERE platid = ? AND openid = ?`,
		platID, openID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &uop, nil
}

// UserOpenFindByUID 根据 uid 查找所有绑定记录
func UserOpenFindByUID(ctx context.Context, db *sqlx.DB, uid uint32) ([]UserOpenPlat, error) {
	var list []UserOpenPlat
	err := db.SelectContext(ctx, &list,
		`SELECT uid, platid, openid FROM bbs_user_open_plat WHERE uid = ?`, uid)
	if err != nil {
		return nil, err
	}
	return list, nil
}

// UserOpenCreate 创建第三方绑定
// 对应 PHP: db_insert('user_open_plat', array('uid'=>$uid, 'platid'=>1, 'openid'=>$openid))
func UserOpenCreate(ctx context.Context, ext DBOrTx, uid uint32, platID PlatID, openID string) error {
	_, err := ext.ExecContext(ctx,
		`INSERT INTO bbs_user_open_plat (uid, platid, openid) VALUES (?, ?, ?)`,
		uid, platID, openID)
	return err
}

// UserOpenDelete 删除第三方绑定
func UserOpenDelete(ctx context.Context, ext DBOrTx, uid uint32, platID PlatID) error {
	_, err := ext.ExecContext(ctx,
		`DELETE FROM bbs_user_open_plat WHERE uid = ? AND platid = ?`,
		uid, platID)
	return err
}

// UserOpenDeleteByUID 删除用户的所有第三方绑定（用户删除时级联）
func UserOpenDeleteByUID(ctx context.Context, ext DBOrTx, uid uint32) error {
	_, err := ext.ExecContext(ctx,
		`DELETE FROM bbs_user_open_plat WHERE uid = ?`, uid)
	return err
}

// SSOProfile 第三方用户信息
type SSOProfile struct {
	OpenID    string `json:"openid"`
	Nickname  string `json:"nickname"`
	AvatarURL string `json:"avatar_url,omitempty"`
	Gender    string `json:"gender,omitempty"`
}

// SSOLoginOrRegister SSO 登录/自动注册
// 对应 PHP: qq_login_read_user_by_openid + qq_login_create_user
// 如果 openid 已绑定，直接返回用户；否则自动创建用户并绑定
func SSOLoginOrRegister(ctx context.Context, db *sqlx.DB, platID PlatID, profile *SSOProfile) (*User, bool, error) {
	// 1. 查找是否已绑定
	bind, err := UserOpenFindByOpenID(ctx, db, platID, profile.OpenID)
	if err != nil {
		return nil, false, err
	}

	// 2. 已绑定 → 返回已有用户
	if bind != nil {
		user, err := GetUserByUID(ctx, db, bind.UID)
		if err != nil {
			// 用户已被删除，清理绑定记录
			UserOpenDeleteByUID(ctx, db, bind.UID)
			return nil, false, nil
		}
		return user, false, nil
	}

	// 3. 未绑定 → 自动创建用户
	now := time.Now().Unix()
	username := SanitizeUsername(profile.Nickname)
	if username == "" {
		username = "user"
	}

	// 检查用户名是否被占用，被占用则加时间戳后缀
	existing, err := GetUserByAccount(ctx, db, username)
	if err != nil && err != sql.ErrNoRows {
		return nil, false, err
	}
	if existing != nil {
		username = SanitizeUsername(profile.Nickname + "_" + formatTimestamp(now))
		if username == "" {
			username = "user_" + formatTimestamp(now)
		}
	}

	// 自动生成邮箱
	email := platID.String() + "_" + formatTimestamp(now) + "@sso.local"

	// 随机密码（用户无法用密码登录，只能通过 SSO）
	password := randomPassword()

	// 创建用户
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()

	user, err := CreateUser(ctx, db, tx, username, email, password)
	if err != nil {
		return nil, false, err
	}

	// 绑定第三方账号
	if err := UserOpenCreate(ctx, tx, user.UID, platID, profile.OpenID); err != nil {
		return nil, false, err
	}

	if err := tx.Commit(); err != nil {
		return nil, false, err
	}

	return user, true, nil
}

func (p PlatID) String() string {
	switch p {
	case PlatQQ:
		return "qq"
	case PlatWechat:
		return "wx"
	case PlatAlipay:
		return "ali"
	default:
		return "local"
	}
}

func formatTimestamp(t int64) string {
	return time.Unix(t, 0).Format("150405")
}

func randomPassword() string {
	return "sso_" + time.Now().Format("150405.000000000")
}
