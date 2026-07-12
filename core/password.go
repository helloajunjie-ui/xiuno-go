// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"crypto/md5"
	"fmt"
	"log"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
)

// XiunoMD5 原版 Xiuno 密码哈希: md5(password + salt)
// 来源: route/user.php:85, route/user.php:158
func XiunoMD5(password, salt string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(password+salt)))
}

// CheckPassword 检查密码，支持 bcrypt（新）和 MD5（旧）两种格式
// 旧格式验证通过后静默升级为 bcrypt
func CheckPassword(db *sqlx.DB, uid int64, inputPwd, storedHash, storedSalt string) bool {
	// 1. 先尝试 bcrypt（新格式）
	if isBcryptHash(storedHash) {
		err := bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(inputPwd))
		return err == nil
	}

	// 2. 回退 MD5（旧格式）: md5(password + salt)
	oldHash := XiunoMD5(inputPwd, storedSalt)
	if storedHash == oldHash {
		// 验证通过，静默升级为 bcrypt
		go silentUpgrade(db, uid, inputPwd)
		return true
	}

	return false
}

// HashPassword bcrypt 哈希密码
func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// isBcryptHash 判断是否为 bcrypt hash（以 $2a$ 或 $2b$ 开头）
func isBcryptHash(hash string) bool {
	if len(hash) < 4 {
		return false
	}
	return hash[:4] == "$2a$" || hash[:4] == "$2b$"
}

// silentUpgrade 静默升级密码为 bcrypt
func silentUpgrade(db *sqlx.DB, uid int64, pwd string) {
	newHash, err := HashPassword(pwd)
	if err != nil {
		log.Printf("[WARN] 密码静默升级失败 uid=%d: %v", uid, err)
		return
	}
	_, err = db.Exec("UPDATE bbs_user SET password = ?, salt = '' WHERE uid = ?", newHash, uid)
	if err != nil {
		log.Printf("[WARN] 密码静默升级写入失败 uid=%d: %v", uid, err)
	}
}
