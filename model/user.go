// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"

	"xiuno/core"
)

// User 对应 bbs_user 表
type User struct {
	UID      uint32 `db:"uid" json:"uid"`
	GID      uint16 `db:"gid" json:"gid"`
	Email    string `db:"email" json:"email"`
	Username string `db:"username" json:"username"`
	Realname string `db:"realname" json:"realname,omitempty"`
	Avatar   uint32 `db:"avatar" json:"avatar"`
	Threads  uint32 `db:"threads" json:"threads"`
	Posts    uint32 `db:"posts" json:"posts"`
	Credits  uint32 `db:"credits" json:"credits"`

	// 敏感字段，禁止 JSON 序列化
	Password    string `db:"password" json:"-"`
	Salt        string `db:"salt" json:"-"`
	PasswordSms string `db:"password_sms" json:"-"`
	IdNumber    string `db:"idnumber" json:"-"`
	Mobile      string `db:"mobile" json:"-"`
	QQ          string `db:"qq" json:"-"`

	// 虚拟货币（不暴露 JSON）
	Golds uint32 `db:"golds" json:"-"`
	Rmbs  uint32 `db:"rmbs" json:"-"`

	// 时间戳与统计
	CreateIP   uint32    `db:"create_ip" json:"-"`
	CreateDate int64     `db:"create_date" json:"create_date"`
	LoginIP    uint32    `db:"login_ip" json:"-"`
	LoginDate  int64     `db:"login_date" json:"login_date"`
	Logins     uint32    `db:"logins" json:"logins"`
	CreatedAt  time.Time `db:"created_at" json:"-"`
	UpdatedAt  time.Time `db:"updated_at" json:"-"`

	// 格式化显示字段（不存库）
	CreateIPFmt   string `db:"-" json:"create_ip_fmt,omitempty"`
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
	LoginIPFmt    string `db:"-" json:"login_ip_fmt,omitempty"`
	LoginDateFmt  string `db:"-" json:"login_date_fmt,omitempty"`
	GroupName     string `db:"-" json:"group_name,omitempty"`
	AvatarURL     string `db:"-" json:"avatar_url,omitempty"`
	AvatarPath    string `db:"-" json:"avatar_path,omitempty"`
}

// UserBrief 用户简要信息，用于列表展示
type UserBrief struct {
	UID       uint32 `json:"uid"`
	Username  string `json:"username"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

// GetUserByEmail 通过邮箱获取用户
func GetUserByEmail(ctx context.Context, db *sqlx.DB, email string) (*User, error) {
	var user User
	err := db.GetContext(ctx, &user, `SELECT * FROM bbs_user WHERE email = ? LIMIT 1`, email)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByAccount 支持用户名或邮箱登录
// 利用 OR 兼顾邮箱和用户名，命中索引
func GetUserByAccount(ctx context.Context, db *sqlx.DB, account string) (*User, error) {
	var user User
	query := `SELECT * FROM bbs_user WHERE username = ? OR email = ? LIMIT 1`
	err := db.GetContext(ctx, &user, query, account, account)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUID 通过 UID 获取用户
func GetUserByUID(ctx context.Context, db *sqlx.DB, uid uint32) (*User, error) {
	var user User
	err := db.GetContext(ctx, &user, `SELECT * FROM bbs_user WHERE uid = ?`, uid)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByID 通过 UID 获取用户（语义别名，与 GetUserByUID 等价）
func GetUserByID(ctx context.Context, db *sqlx.DB, uid uint32) (*User, error) {
	return GetUserByUID(ctx, db, uid)
}

// GetAvatarPath 根据 UID 计算头像存储路径
// Xiuno 极客设计：UID 补齐 9 位，按 3 位一层切分目录
// 如 UID=123 → "avatar/000/000/123.png"
// 千万级用户量下防单目录 Inode 爆炸的教科书级算法
func GetAvatarPath(uid uint32) string {
	s := fmt.Sprintf("%09d", uid)
	return fmt.Sprintf("avatar/%s/%s/%s.png", s[0:3], s[3:6], s[6:9])
}

// EnsureDefaultAvatar 确保默认头像文件存在，不存在则生成一个 128x128 灰色占位 PNG
// 避免前端 <img> 加载默认头像时 404 导致 @error 回退链断裂
// uploadDir 是上传根目录（如 "upload"），默认头像路径为 uploadDir + "/avatar/0.png"
func EnsureDefaultAvatar(uploadDir string) error {
	path := filepath.Join(uploadDir, "avatar", "0.png")
	if _, err := os.Stat(path); err == nil {
		return nil // 已存在
	}

	// 创建目录
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("创建默认头像目录失败: %w", err)
	}

	// 生成 128x128 灰色头像：浅灰背景 + 深灰人形轮廓
	img := image.NewRGBA(image.Rect(0, 0, 128, 128))

	// 填充浅灰色背景 (#E5E7EB)
	bg := color.RGBA{229, 231, 235, 255}
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			img.Set(x, y, bg)
		}
	}

	// 画一个简单的圆形人头轮廓（深灰色 #9CA3AF）
	headColor := color.RGBA{156, 163, 175, 255}
	cx, cy, r := 64, 50, 28 // 圆心 (64,50)，半径 28
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				img.Set(x, y, headColor)
			}
		}
	}

	// 画一个简单的身体轮廓（深灰色 #9CA3AF）
	bodyColor := color.RGBA{156, 163, 175, 255}
	for y := 82; y < 128; y++ {
		halfWidth := 35 - (y-82)*3/10 // 上宽下窄的梯形
		if halfWidth < 10 {
			halfWidth = 10
		}
		for x := 64 - halfWidth; x <= 64+halfWidth; x++ {
			img.Set(x, y, bodyColor)
		}
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fmt.Errorf("编码默认头像 PNG 失败: %w", err)
	}

	// 写入文件
	if err := os.WriteFile(path, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("写入默认头像文件失败: %w", err)
	}

	return nil
}

// UpdateUserAvatar 更新用户头像时间戳
// 数据库只存时间戳，前端拼接 URL 时加 ?t=timestamp 解决缓存更新
func UpdateUserAvatar(ctx context.Context, db *sqlx.DB, uid uint32, timestamp int64) error {
	_, err := db.ExecContext(ctx, `UPDATE bbs_user SET avatar = ? WHERE uid = ?`, timestamp, uid)
	return err
}

// VerifyPassword 密码校验与静默升级策略
// 返回 (是否通过, 是否需要更新密码hash)
func (u *User) VerifyPassword(ctx context.Context, db *sqlx.DB, plainPassword string) (bool, bool) {
	// 1. 如果已经是 bcrypt
	if len(u.Password) > 10 && (u.Password[:4] == "$2a$" || u.Password[:4] == "$2b$") {
		err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(plainPassword))
		return err == nil, false
	}

	// 2. 兼容旧版 Xiuno MD5: md5(password + salt)
	// 来源: route/user.php:85
	oldHash := core.XiunoMD5(plainPassword, u.Salt)
	if u.Password == oldHash {
		return true, true // 验证通过，需要静默升级
	}

	return false, false
}

// UpgradePassword 静默升级密码为 bcrypt
func (u *User) UpgradePassword(ctx context.Context, db *sqlx.DB, plainPassword string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.Password = string(hash)
	u.Salt = ""
	_, err = db.ExecContext(ctx, `UPDATE bbs_user SET password = ?, salt = '' WHERE uid = ?`, u.Password, u.UID)
	return err
}

// CreateUser 创建用户（注册）
func CreateUser(ctx context.Context, db *sqlx.DB, tx *sqlx.Tx, username, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 检测是否为第一个真实注册用户（排除空密码的旧版预置账号）
	// 第一个注册用户自动成为管理员（gid=1）
	var userCount int
	_ = db.GetContext(ctx, &userCount, "SELECT COUNT(*) FROM bbs_user WHERE password != '' AND password IS NOT NULL")
	defaultGID := uint16(101) // 默认一级用户组
	if userCount == 0 {
		defaultGID = 1 // 第一个用户为管理员
	}

	now := time.Now().Unix()
	user := &User{
		GID:        defaultGID,
		Email:      email,
		Username:   username,
		Password:   string(hash),
		Salt:       "", // bcrypt 不需要 salt
		CreateDate: now,
		LoginDate:  now,
		Logins:     1,
	}

	var execer sqlx.ExecerContext = db
	if tx != nil {
		execer = tx
	}

	result, err := execer.ExecContext(ctx,
		`INSERT INTO bbs_user (gid, email, username, password, salt, create_date, login_date, logins)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		user.GID, user.Email, user.Username, user.Password, user.Salt,
		user.CreateDate, user.LoginDate, user.Logins)
	if err != nil {
		return nil, err
	}

	lastID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}
	user.UID = uint32(lastID)

	return user, nil
}

// CheckUserExists 检查用户名或邮箱是否已存在
func CheckUserExists(ctx context.Context, db *sqlx.DB, username, email string) (string, error) {
	var uid uint32
	err := db.GetContext(ctx, &uid, `SELECT uid FROM bbs_user WHERE username = ? OR email = ? LIMIT 1`, username, email)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return "用户名或邮箱已被注册", nil
}

// UpdateUserGroup 修改用户组
// gid=7 为禁止用户组（小黑屋），gid=1 为超管，gid=101 为普通用户
func UpdateUserGroup(ctx context.Context, db *sqlx.DB, uid uint32, gid uint16) error {
	_, err := db.ExecContext(ctx, `UPDATE bbs_user SET gid = ? WHERE uid = ?`, gid, uid)
	return err
}

// UpdatePassword 彻底更新用户密码（抹除旧版 salt，全面拥抱 bcrypt）
func UpdatePassword(ctx context.Context, db *sqlx.DB, uid uint32, plainPassword string) error {
	hashText, err := bcrypt.GenerateFromPassword([]byte(plainPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// 写入新 hash 的同时，把历史遗留的 salt 清空
	_, err = db.ExecContext(ctx, `UPDATE bbs_user SET password = ?, salt = '' WHERE uid = ?`, string(hashText), uid)
	return err
}

// FindUser 用户列表查询（分页，按 uid 降序）
// 对应原版 user_find()
func FindUser(ctx context.Context, db *sqlx.DB, page, pageSize int) ([]User, error) {
	offset := (page - 1) * pageSize
	var users []User
	err := db.SelectContext(ctx, &users,
		`SELECT uid, gid, email, username, avatar, threads, posts, credits, create_date, login_date, logins
		 FROM bbs_user ORDER BY uid DESC LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("FindUser: %w", err)
	}
	if users == nil {
		users = []User{}
	}
	return users, nil
}

// UserFormat 格式化用户数据（填充显示字段）
// 对应 PHP: user_format()
func UserFormat(user *User, groupName string, uploadURL, uploadPath string) {
	if user == nil {
		return
	}
	// IP 格式化（uint32 存储，仅当非 0 时格式化）
	if user.CreateIP != 0 {
		user.CreateIPFmt = modelLong2IP(user.CreateIP)
	}
	if user.LoginIP != 0 {
		user.LoginIPFmt = modelLong2IP(user.LoginIP)
	}
	// 日期格式化
	user.CreateDateFmt = humandate(user.CreateDate)
	user.LoginDateFmt = humandate(user.LoginDate)
	// 用户组名
	user.GroupName = groupName
	// 头像路径
	user.AvatarPath = GetAvatarPath(user.UID)
	if uploadURL != "" {
		user.AvatarURL = uploadURL + "/" + user.AvatarPath
	} else {
		user.AvatarURL = user.AvatarPath
	}
}

// UserSafeInfo 返回用户安全信息（移除敏感字段）
// 对应 PHP: user_safe_info()
func UserSafeInfo(user *User) *User {
	if user == nil {
		return nil
	}
	// 返回副本，不修改原对象
	safe := *user
	safe.Password = ""
	safe.Salt = ""
	safe.PasswordSms = ""
	safe.IdNumber = ""
	safe.Mobile = ""
	safe.CreateIP = 0
	safe.LoginIP = 0
	return &safe
}

// UserGuest 返回访客用户信息
// 对应 PHP: user_guest()
func UserGuest() *User {
	return &User{
		GID:       0,
		Username:  "guest",
		GroupName: "游客",
	}
}

// UserReadByUsername 通过用户名获取用户
// 对应 PHP: user_read_by_username()
func UserReadByUsername(ctx context.Context, db *sqlx.DB, username string) (*User, error) {
	var user User
	err := db.GetContext(ctx, &user, `SELECT * FROM bbs_user WHERE username = ? LIMIT 1`, username)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UserCount 统计用户数量
// 对应 PHP: user_count()
func UserCount(ctx context.Context, db *sqlx.DB) (int, error) {
	var count int
	err := db.GetContext(ctx, &count, `SELECT COUNT(*) FROM bbs_user`)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// UserFindByUIDs 批量通过 UID 获取用户
// 对应 PHP: user_find_by_uids()
// 返回 map[uid]*User，方便前端快速查找
func UserFindByUIDs(ctx context.Context, db *sqlx.DB, uids []uint32) (map[uint32]*User, error) {
	if len(uids) == 0 {
		return map[uint32]*User{}, nil
	}
	query, args, err := sqlx.In(`SELECT * FROM bbs_user WHERE uid IN (?)`, uids)
	if err != nil {
		return nil, fmt.Errorf("UserFindByUIDs: %w", err)
	}
	query = db.Rebind(query)
	var users []User
	err = db.SelectContext(ctx, &users, query, args...)
	if err != nil {
		return nil, fmt.Errorf("UserFindByUIDs: %w", err)
	}
	result := make(map[uint32]*User, len(users))
	for i := range users {
		result[users[i].UID] = &users[i]
	}
	return result, nil
}

// UserDelete 硬删除用户及其所有关联数据
// 对应 PHP: user_delete($uid)
// 级联删除：用户主题 → 回帖 → 附件 → 头像 → 用户记录
// uploadDir 用于删除物理附件文件
func UserDelete(ctx context.Context, db *sqlx.DB, uid uint32, uploadDir string) error {
	// 委托给 CascadeDeleteUser 完成级联删除
	return CascadeDeleteUser(ctx, db, uid, uploadDir)
}

// UserMaxID 获取最大用户 ID
// 对应 PHP: user_maxid()
func UserMaxID(ctx context.Context, db *sqlx.DB) (uint32, error) {
	var maxID uint32
	err := db.GetContext(ctx, &maxID, `SELECT COALESCE(MAX(uid), 0) FROM bbs_user`)
	if err != nil {
		return 0, fmt.Errorf("UserMaxID: %w", err)
	}
	return maxID, nil
}

// UpdateUser 更新用户信息
// 对应 PHP: user_update($uid, $arr)
// 使用 map 动态构建 SET 子句，仅更新非零字段
func UpdateUser(ctx context.Context, ext DBOrTx, uid uint32, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	var setClauses []string
	var args []interface{}
	for k, v := range fields {
		setClauses = append(setClauses, fmt.Sprintf("%s = ?", k))
		args = append(args, v)
	}
	args = append(args, uid)
	query := fmt.Sprintf("UPDATE bbs_user SET %s WHERE uid = ?", joinStrings(setClauses, ", "))
	_, err := ext.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("UpdateUser: %w", err)
	}
	return nil
}

// joinStrings 将字符串切片用分隔符连接
func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for _, p := range parts[1:] {
		result += sep + p
	}
	return result
}
