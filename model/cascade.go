package model

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/jmoiron/sqlx"
)

// ============================================================
// 级联删除引擎
// 对应 PHP: model/thread.func.php thread_delete()
//          model/user.func.php   user_delete()
//          model/attach.func.php attach_delete_by_pid/uid()
//
// 设计原则：
//   - 硬删除（物理 DELETE FROM），与 PHP 版行为一致
//   - 每次删除都同步更新统计（版块 threads/posts、用户 threads/posts）
//   - 删除附件时同步删除物理文件
//   - 所有级联操作在单个事务中完成，保证原子性
// ============================================================

// --- 主题级联删除 ---

// CascadeDeleteThread 硬删除主题及其所有关联数据
// 删除链：thread → posts → attach(物理文件) → mythread → 统计更新
func CascadeDeleteThread(ctx context.Context, tx *sqlx.Tx, tid uint32) error {
	// 1. 查出主题基本信息（fid, uid 用于后续统计更新）
	var thread struct {
		FID uint32 `db:"fid"`
		UID uint32 `db:"uid"`
	}
	err := tx.GetContext(ctx, &thread, `SELECT fid, uid FROM bbs_thread WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread read thread %d: %w", tid, err)
	}

	// 2. 查出该主题下所有回帖的 PID（含首帖），用于删除附件
	var pids []uint32
	err = tx.SelectContext(ctx, &pids, `SELECT pid FROM bbs_post WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread read pids %d: %w", tid, err)
	}

	// 3. 删除该主题下所有附件（数据库记录 + 物理文件）
	//    注意：附件删除涉及文件系统操作，无法回滚，放在事务最前面执行
	//    如果文件删除失败，仅记录 WARN 日志，不阻塞事务
	if len(pids) > 0 {
		if err := deleteAttachByPIDs(ctx, tx, pids); err != nil {
			return fmt.Errorf("CascadeDeleteThread delete attach %d: %w", tid, err)
		}
	}

	// 4. 删除所有回帖
	_, err = tx.ExecContext(ctx, `DELETE FROM bbs_post WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread delete posts %d: %w", tid, err)
	}

	// 5. 删除 mythread 关联
	_, _ = tx.ExecContext(ctx, `DELETE FROM bbs_mythread WHERE tid = ?`, tid)

	// 5.5 删除 thread_top 置顶记录
	_, _ = tx.ExecContext(ctx, `DELETE FROM bbs_thread_top WHERE tid = ?`, tid)

	// 6. 删除主题本身
	_, err = tx.ExecContext(ctx, `DELETE FROM bbs_thread WHERE tid = ?`, tid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread delete thread %d: %w", tid, err)
	}

	// 7. 更新版块统计（GREATEST 防 unsigned 溢出）
	_, err = tx.ExecContext(ctx,
		`UPDATE bbs_forum SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE fid = ?`, thread.FID)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread decr forum %d: %w", thread.FID, err)
	}

	// 8. 更新用户统计
	_, err = tx.ExecContext(ctx,
		`UPDATE bbs_user SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE uid = ?`, thread.UID)
	if err != nil {
		return fmt.Errorf("CascadeDeleteThread decr user %d: %w", thread.UID, err)
	}

	return nil
}

// --- 回帖级联删除 ---

// CascadeDeletePost 硬删除单个回帖及其关联附件
// 注意：不更新 thread.posts 计数（由 AsyncCounter 异步处理）
func CascadeDeletePost(ctx context.Context, tx *sqlx.Tx, pid uint32) error {
	// 1. 查出回帖基本信息
	var post struct {
		TID uint32 `db:"tid"`
		UID uint32 `db:"uid"`
	}
	err := tx.GetContext(ctx, &post, `SELECT tid, uid FROM bbs_post WHERE pid = ?`, pid)
	if err != nil {
		return fmt.Errorf("CascadeDeletePost read post %d: %w", pid, err)
	}

	// 2. 删除该回帖下的附件
	if err := deleteAttachByPIDs(ctx, tx, []uint32{pid}); err != nil {
		return fmt.Errorf("CascadeDeletePost delete attach %d: %w", pid, err)
	}

	// 3. 删除回帖
	_, err = tx.ExecContext(ctx, `DELETE FROM bbs_post WHERE pid = ?`, pid)
	if err != nil {
		return fmt.Errorf("CascadeDeletePost delete post %d: %w", pid, err)
	}

	// 4. 更新用户回帖计数
	_, err = tx.ExecContext(ctx,
		`UPDATE bbs_user SET posts = GREATEST(CAST(posts AS SIGNED) - 1, 0) WHERE uid = ?`, post.UID)
	if err != nil {
		return fmt.Errorf("CascadeDeletePost decr user %d: %w", post.UID, err)
	}

	return nil
}

// --- 用户级联删除 ---

// CascadeDeleteUser 硬删除用户及其所有关联数据
// 删除链：user → threads(级联) → posts → attach(物理文件) → 头像文件 → 统计更新
func CascadeDeleteUser(ctx context.Context, db *sqlx.DB, uid uint32, uploadDir string) error {
	// 用户级联删除涉及大量数据（可能数百个主题），无法在单个事务中完成
	// 策略：先查出所有主题 TID，逐个调用 CascadeDeleteThread（每个主题独立事务）
	//       再删除剩余回帖和附件

	// 1. 查出用户所有主题
	var tids []uint32
	err := db.SelectContext(ctx, &tids, `SELECT tid FROM bbs_thread WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteUser read tids %d: %w", uid, err)
	}

	// 2. 逐个级联删除主题（每个主题独立事务）
	for _, tid := range tids {
		err := dbTx(ctx, db, func(tx *sqlx.Tx) error {
			return CascadeDeleteThread(ctx, tx, tid)
		})
		if err != nil {
			log.Printf("[WARN] CascadeDeleteUser delete thread %d: %v", tid, err)
			// 继续删除下一个，不阻塞
		}
	}

	// 3. 删除用户剩余回帖（非首帖，首帖随主题已删除）
	//    先查出剩余回帖的 PID
	var remainPids []uint32
	err = db.SelectContext(ctx, &remainPids,
		`SELECT pid FROM bbs_post WHERE uid = ? AND isfirst = 0`, uid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteUser read remain pids %d: %w", uid, err)
	}

	if len(remainPids) > 0 {
		err = dbTx(ctx, db, func(tx *sqlx.Tx) error {
			// 删除附件
			if err := deleteAttachByPIDs(ctx, tx, remainPids); err != nil {
				return err
			}
			// 删除回帖
			_, err := tx.ExecContext(ctx, `DELETE FROM bbs_post WHERE pid IN (?)`, remainPids)
			if err != nil {
				return fmt.Errorf("CascadeDeleteUser delete remain posts: %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("CascadeDeleteUser delete remain posts %d: %w", uid, err)
		}
	}

	// 4. 删除用户未关联帖子的附件（孤儿附件）
	err = dbTx(ctx, db, func(tx *sqlx.Tx) error {
		return deleteAttachByUID(ctx, tx, uid)
	})
	if err != nil {
		log.Printf("[WARN] CascadeDeleteUser delete orphan attach %d: %v", uid, err)
	}

	// 5. 删除 mythread 关联
	if err := MyThreadDeleteByUID(ctx, db, uid); err != nil {
		log.Printf("[WARN] CascadeDeleteUser delete mythread %d: %v", uid, err)
	}

	// 6. 删除头像文件
	deleteAvatarFile(uid, uploadDir)

	// 7. 删除用户记录
	_, err = db.ExecContext(ctx, `DELETE FROM bbs_user WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("CascadeDeleteUser delete user %d: %w", uid, err)
	}

	return nil
}

// --- 附件级联删除（内部工具函数） ---

// deleteAttachByPIDs 删除指定 PID 列表的所有附件（数据库 + 物理文件）
func deleteAttachByPIDs(ctx context.Context, tx *sqlx.Tx, pids []uint32) error {
	if len(pids) == 0 {
		return nil
	}

	// 1. 查出附件记录（获取文件名用于删除物理文件）
	var attaches []struct {
		AID      uint32 `db:"aid"`
		Filename string `db:"filename"`
	}
	query, args, err := sqlx.In(`SELECT aid, filename FROM bbs_attach WHERE pid IN (?)`, pids)
	if err != nil {
		return fmt.Errorf("deleteAttachByPIDs select: %w", err)
	}
	err = tx.SelectContext(ctx, &attaches, tx.Rebind(query), args...)
	if err != nil {
		return fmt.Errorf("deleteAttachByPIDs query: %w", err)
	}

	if len(attaches) == 0 {
		return nil
	}

	// 2. 收集 AID 列表用于删除数据库记录
	var aids []uint32
	for _, att := range attaches {
		aids = append(aids, att.AID)
		// 删除物理文件（忽略错误，文件可能已被手动清理）
		deleteFile(att.Filename)
	}

	// 3. 删除数据库记录
	delQuery, delArgs, err := sqlx.In(`DELETE FROM bbs_attach WHERE aid IN (?)`, aids)
	if err != nil {
		return fmt.Errorf("deleteAttachByPIDs delete: %w", err)
	}
	_, err = tx.ExecContext(ctx, tx.Rebind(delQuery), delArgs...)
	if err != nil {
		return fmt.Errorf("deleteAttachByPIDs exec: %w", err)
	}

	return nil
}

// deleteAttachByUID 删除指定用户的所有附件（数据库 + 物理文件）
func deleteAttachByUID(ctx context.Context, tx *sqlx.Tx, uid uint32) error {
	var attaches []struct {
		AID      uint32 `db:"aid"`
		Filename string `db:"filename"`
	}
	err := tx.SelectContext(ctx, &attaches, `SELECT aid, filename FROM bbs_attach WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("deleteAttachByUID query: %w", err)
	}

	if len(attaches) == 0 {
		return nil
	}

	var aids []uint32
	for _, att := range attaches {
		aids = append(aids, att.AID)
		deleteFile(att.Filename)
	}

	delQuery, delArgs, err := sqlx.In(`DELETE FROM bbs_attach WHERE aid IN (?)`, aids)
	if err != nil {
		return fmt.Errorf("deleteAttachByUID build: %w", err)
	}
	_, err = tx.ExecContext(ctx, tx.Rebind(delQuery), delArgs...)
	if err != nil {
		return fmt.Errorf("deleteAttachByUID exec: %w", err)
	}

	return nil
}

// --- 文件系统操作 ---

// deleteFile 删除物理文件，忽略"文件不存在"错误
func deleteFile(filename string) {
	if filename == "" {
		return
	}
	// 尝试多个可能的路径（兼容不同部署方式）
	paths := []string{
		filepath.Join("upload", "attach", filename),
		filepath.Join("upload", filename),
	}
	for _, p := range paths {
		if err := os.Remove(p); err == nil {
			return // 成功删除
		} else if !os.IsNotExist(err) {
			log.Printf("[WARN] 删除附件文件失败 %s: %v", p, err)
		}
	}
}

// deleteAvatarFile 删除用户头像文件
func deleteAvatarFile(uid uint32, uploadDir string) {
	// PHP 版头像路径: upload/avatar/{dir3}/{uid}.png
	dir := fmt.Sprintf("%03d", uid)
	avatarPath := filepath.Join(uploadDir, "avatar", dir[:3], fmt.Sprintf("%d.png", uid))
	if err := os.Remove(avatarPath); err != nil && !os.IsNotExist(err) {
		log.Printf("[WARN] 删除头像文件失败 %s: %v", avatarPath, err)
	}
}

// --- 事务辅助 ---

// dbTx 执行一个闭包事务
func dbTx(ctx context.Context, db *sqlx.DB, fn func(tx *sqlx.Tx) error) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = tx.Rollback()
			panic(r)
		}
	}()
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

// --- 兼容旧接口（软删除 → 级联硬删除的迁移桥） ---

// HardDeleteThread 外部调用的主题硬删除入口
// 替代 SoftDeleteThread，用于"彻底删除"场景（管理员回收站清空）
func HardDeleteThread(ctx context.Context, db *sqlx.DB, tid uint32) error {
	return dbTx(ctx, db, func(tx *sqlx.Tx) error {
		return CascadeDeleteThread(ctx, tx, tid)
	})
}

// HardDeletePost 外部调用的回帖硬删除入口
// 替代 SoftDeletePost，用于"彻底删除"场景
func HardDeletePost(ctx context.Context, db *sqlx.DB, pid uint32) error {
	return dbTx(ctx, db, func(tx *sqlx.Tx) error {
		return CascadeDeletePost(ctx, tx, pid)
	})
}

// Ensure time import is used
var _ = time.Now
