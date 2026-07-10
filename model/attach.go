package model

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/jmoiron/sqlx"
)

// Attach 对应 bbs_attach 表
type Attach struct {
	AID         int64  `db:"aid" json:"aid"`
	TID         int64  `db:"tid" json:"tid"`
	PID         int64  `db:"pid" json:"pid"`
	UID         int64  `db:"uid" json:"uid"`
	FileSize    int64  `db:"filesize" json:"filesize"`
	Width       int32  `db:"width" json:"width"`
	Height      int32  `db:"height" json:"height"`
	Filename    string `db:"filename" json:"filename"`
	OrgFilename string `db:"orgfilename" json:"orgfilename"`
	FileType    string `db:"filetype" json:"filetype"`
	CreateDate  int64  `db:"create_date" json:"create_date"`
	Comment     string `db:"comment" json:"comment,omitempty"`
	Downloads   int64  `db:"downloads" json:"downloads"`
	Credits     int64  `db:"credits" json:"credits"`
	Golds       int64  `db:"golds" json:"golds"`
	Rmbs        int64  `db:"rmbs" json:"rmbs"`
	IsImage     int32  `db:"isimage" json:"isimage"`
	// 以下字段由 AttachFormat 填充
	CreateDateFmt string `db:"-" json:"create_date_fmt,omitempty"`
	URL           string `db:"-" json:"url,omitempty"`
}

// GetAttach 根据 aid 获取附件记录
func GetAttach(ctx context.Context, db *sqlx.DB, aid uint32) (*Attach, error) {
	att := &Attach{}
	err := db.GetContext(ctx, att, "SELECT * FROM bbs_attach WHERE aid = ?", aid)
	if err != nil {
		return nil, err
	}
	return att, nil
}

// DeleteAttach 删除附件记录（软删除：设置 deleted_at 或直接 DELETE）
// 当前使用硬删除，与 PHP 版行为一致
func DeleteAttach(ctx context.Context, db *sqlx.DB, aid uint32) error {
	_, err := db.ExecContext(ctx, "DELETE FROM bbs_attach WHERE aid = ?", aid)
	return err
}

// IncrAttachDownload 递增附件下载计数
func IncrAttachDownload(ctx context.Context, db *sqlx.DB, aid uint32) error {
	_, err := db.ExecContext(ctx, "UPDATE bbs_attach SET downloads = downloads + 1 WHERE aid = ?", aid)
	return err
}

// CreateAttach 创建附件记录，返回新 aid
func CreateAttach(ctx context.Context, db *sqlx.DB, att *Attach) (int64, error) {
	att.CreateDate = time.Now().Unix()
	result, err := db.ExecContext(ctx, `
		INSERT INTO bbs_attach (tid, pid, uid, filesize, width, height, filename, orgfilename, filetype, create_date, comment, downloads, credits, golds, rmbs, isimage)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		att.TID, att.PID, att.UID, att.FileSize, att.Width, att.Height,
		att.Filename, att.OrgFilename, att.FileType, att.CreateDate,
		att.Comment, att.Downloads, att.Credits, att.Golds, att.Rmbs, att.IsImage,
	)
	if err != nil {
		return 0, err
	}
	return result.LastInsertId()
}

// UpdateAttach 更新附件记录
// 对应 PHP: attach_update($aid, $arr)
func UpdateAttach(ctx context.Context, db *sqlx.DB, aid uint32, att *Attach) error {
	_, err := db.ExecContext(ctx, `
		UPDATE bbs_attach SET tid=?, pid=?, uid=?, filesize=?, width=?, height=?,
		filename=?, orgfilename=?, filetype=?, comment=?, downloads=?, credits=?, golds=?, rmbs=?, isimage=?
		WHERE aid=?`,
		att.TID, att.PID, att.UID, att.FileSize, att.Width, att.Height,
		att.Filename, att.OrgFilename, att.FileType, att.Comment,
		att.Downloads, att.Credits, att.Golds, att.Rmbs, att.IsImage,
		aid)
	if err != nil {
		return fmt.Errorf("UpdateAttach: %w", err)
	}
	return nil
}

// --- 附件关联帖子 ---

// attachURLRegex 匹配附件 URL 的正则
// 匹配 /upload/202601/02/150405.123456789.jpg 格式
var attachURLRegex = regexp.MustCompile(`/upload/(\d{6}/\d{2}/[^"'\s]+)`)

// AttachAssocPost 将未关联的附件关联到指定帖子
// 对应 PHP: model/attach.func.php attach_assoc_post()
//
// 逻辑：
//  1. 扫描 message 中的附件 URL
//  2. 根据文件名匹配当前用户的未关联附件（tid=0）
//  3. 更新附件记录的 tid 和 pid
//  4. 更新 thread 的 images/files 计数
//
// 注意：Go 版简化了 PHP 的 session tmp_files 机制，
// 上传时直接保存到 upload/attach/ 目录并创建附件记录（tid=0, pid=0），
// 发帖时通过扫描 message 中的 URL 来关联。
func AttachAssocPost(ctx context.Context, db *sqlx.DB, tid, pid, uid uint32, message string) {
	// 1. 扫描 message 中的附件文件名
	matches := attachURLRegex.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return
	}

	// 2. 收集文件名
	filenames := make([]string, 0, len(matches))
	for _, m := range matches {
		filenames = append(filenames, m[1])
	}

	// 3. 匹配当前用户的未关联附件
	//    条件：filename IN (?) AND uid = ? AND tid = 0
	query, args, err := sqlx.In(
		`SELECT aid, isimage FROM bbs_attach WHERE filename IN (?) AND uid = ? AND tid = 0`,
		filenames, uid)
	if err != nil {
		log.Printf("[WARN] AttachAssocPost build query: %v", err)
		return
	}

	var attaches []struct {
		AID     int64 `db:"aid"`
		IsImage int32 `db:"isimage"`
	}
	err = db.SelectContext(ctx, &attaches, db.Rebind(query), args...)
	if err != nil {
		log.Printf("[WARN] AttachAssocPost query: %v", err)
		return
	}

	if len(attaches) == 0 {
		return
	}

	// 4. 更新附件记录的 tid 和 pid
	var aids []int64
	images := int32(0)
	files := int32(0)
	for _, att := range attaches {
		aids = append(aids, att.AID)
		if att.IsImage == 1 {
			images++
		} else {
			files++
		}
	}

	// 批量更新 tid 和 pid
	updateQuery, updateArgs, err := sqlx.In(
		`UPDATE bbs_attach SET tid = ?, pid = ? WHERE aid IN (?)`,
		tid, pid, aids)
	if err != nil {
		log.Printf("[WARN] AttachAssocPost build update: %v", err)
		return
	}
	_, err = db.ExecContext(ctx, db.Rebind(updateQuery), updateArgs...)
	if err != nil {
		log.Printf("[WARN] AttachAssocPost update: %v", err)
		return
	}

	// 5. 更新 thread 的 images/files 计数
	//    使用 + 操作而非直接赋值，因为可能有多批附件关联到同一主题
	if images > 0 {
		_, err = db.ExecContext(ctx,
			`UPDATE bbs_thread SET images = images + ? WHERE tid = ?`, images, tid)
		if err != nil {
			log.Printf("[WARN] AttachAssocPost update thread images: %v", err)
		}
	}
	if files > 0 {
		_, err = db.ExecContext(ctx,
			`UPDATE bbs_thread SET files = files + ? WHERE tid = ?`, files, tid)
		if err != nil {
			log.Printf("[WARN] AttachAssocPost update thread files: %v", err)
		}
	}

	// 6. 更新 post 的 images/files 计数
	if images > 0 || files > 0 {
		_, err = db.ExecContext(ctx,
			`UPDATE bbs_post SET images = images + ?, files = files + ? WHERE pid = ?`,
			images, files, pid)
		if err != nil {
			log.Printf("[WARN] AttachAssocPost update post images: %v", err)
		}
	}
}

// AttachGC 附件垃圾回收
// 对应 PHP: model/attach.func.php attach_gc()
//
// 清理策略：
//  1. 清理 upload/tmp/ 目录下超过 1 天的临时文件（PHP 兼容）
//  2. 清理未关联附件（tid=0 且 create_date > 1 天）的数据库记录和文件
//     这些附件是用户上传但未发帖产生的垃圾数据
func AttachGC(ctx context.Context, db *sqlx.DB, uploadDir string) {
	// 1. 清理 upload/tmp/ 临时文件
	tmpDir := uploadDir + "/tmp"
	if entries, err := os.ReadDir(tmpDir); err == nil {
		now := time.Now()
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			info, err := entry.Info()
			if err != nil {
				continue
			}
			// 超过 1 天的临时文件
			if now.Sub(info.ModTime()) > 24*time.Hour {
				fullPath := filepath.Join(tmpDir, entry.Name())
				if err := os.Remove(fullPath); err != nil {
					log.Printf("[AttachGC] 删除临时文件失败: %s, %v", fullPath, err)
				}
			}
		}
	}

	// 2. 清理未关联附件（tid=0 且超过 1 天）
	cutoff := time.Now().AddDate(0, 0, -1).Unix()
	var orphanAttaches []Attach
	err := db.SelectContext(ctx, &orphanAttaches,
		`SELECT * FROM bbs_attach WHERE tid = 0 AND create_date < ?`, cutoff)
	if err != nil {
		log.Printf("[AttachGC] 查询未关联附件失败: %v", err)
		return
	}

	for _, att := range orphanAttaches {
		// 删除存储文件
		fullPath := filepath.Join(uploadDir, "attach", att.Filename)
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[AttachGC] 删除附件文件失败: %s, %v", fullPath, err)
		}
		// 删除数据库记录
		if _, err := db.ExecContext(ctx, "DELETE FROM bbs_attach WHERE aid = ?", att.AID); err != nil {
			log.Printf("[AttachGC] 删除附件记录失败 aid=%d: %v", att.AID, err)
		}
	}

	if len(orphanAttaches) > 0 {
		log.Printf("[AttachGC] 清理了 %d 个未关联附件", len(orphanAttaches))
	}
}

// ============================================================
// 以下为 P3 补齐函数，对应 PHP model/attach.func.php
// ============================================================

// AttachFormat 格式化附件数据
// 对应 PHP: attach_format(&$attach)
func AttachFormat(att *Attach, uploadURL string) {
	if att == nil {
		return
	}
	att.CreateDateFmt = dateYMD(att.CreateDate)
	att.URL = uploadURL + "/attach/" + att.Filename
}

// AttachCount 统计附件数量
// 对应 PHP: attach_count($cond)
func AttachCount(ctx context.Context, db *sqlx.DB, cond map[string]interface{}) (int, error) {
	query := `SELECT COUNT(*) FROM bbs_attach`
	args := []interface{}{}
	first := true
	for k, v := range cond {
		if first {
			query += " WHERE " + k + " = ?"
			first = false
		} else {
			query += " AND " + k + " = ?"
		}
		args = append(args, v)
	}
	var count int
	err := db.GetContext(ctx, &count, query, args...)
	return count, err
}

// AttachFindByPID 根据帖子 ID 查找附件
// 对应 PHP: attach_find_by_pid($pid)
// 返回 (attachlist, imagelist, filelist)
func AttachFindByPID(ctx context.Context, db *sqlx.DB, pid uint32) ([]Attach, []Attach, []Attach, error) {
	var attachlist []Attach
	err := db.SelectContext(ctx, &attachlist,
		`SELECT * FROM bbs_attach WHERE pid = ? ORDER BY aid ASC`, pid)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("AttachFindByPID: %w", err)
	}
	if attachlist == nil {
		attachlist = []Attach{}
	}

	var imagelist, filelist []Attach
	for _, att := range attachlist {
		if att.IsImage == 1 {
			imagelist = append(imagelist, att)
		} else {
			filelist = append(filelist, att)
		}
	}
	if imagelist == nil {
		imagelist = []Attach{}
	}
	if filelist == nil {
		filelist = []Attach{}
	}
	return attachlist, imagelist, filelist, nil
}

// AttachDeleteByPID 删除某个帖子下的所有附件（含文件）
// 对应 PHP: attach_delete_by_pid($pid)
func AttachDeleteByPID(ctx context.Context, db *sqlx.DB, pid uint32, uploadDir string) (int, error) {
	attachlist, _, _, err := AttachFindByPID(ctx, db, pid)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, att := range attachlist {
		fullPath := uploadDir + "/attach/" + att.Filename
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[WARN] AttachDeleteByPID 删除文件失败: %s, %v", fullPath, err)
		}
		if _, err := db.ExecContext(ctx, "DELETE FROM bbs_attach WHERE aid = ?", att.AID); err != nil {
			log.Printf("[WARN] AttachDeleteByPID 删除记录失败 aid=%d: %v", att.AID, err)
		}
		count++
	}
	return count, nil
}

// AttachDeleteByUID 删除某个用户的所有附件（含文件）
// 对应 PHP: attach_delete_by_uid($uid)
func AttachDeleteByUID(ctx context.Context, db *sqlx.DB, uid uint32, uploadDir string) error {
	var attachlist []Attach
	err := db.SelectContext(ctx, &attachlist,
		`SELECT * FROM bbs_attach WHERE uid = ?`, uid)
	if err != nil {
		return fmt.Errorf("AttachDeleteByUID: %w", err)
	}
	for _, att := range attachlist {
		fullPath := uploadDir + "/attach/" + att.Filename
		if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
			log.Printf("[WARN] AttachDeleteByUID 删除文件失败: %s, %v", fullPath, err)
		}
		if _, err := db.ExecContext(ctx, "DELETE FROM bbs_attach WHERE aid = ?", att.AID); err != nil {
			log.Printf("[WARN] AttachDeleteByUID 删除记录失败 aid=%d: %v", att.AID, err)
		}
	}
	return nil
}

// AttachFind 查找附件列表
// 对应 PHP: attach_find($cond, $orderby, $page, $pagesize)
func AttachFind(ctx context.Context, db *sqlx.DB, page, pageSize int) ([]Attach, error) {
	offset := (page - 1) * pageSize
	var list []Attach
	err := db.SelectContext(ctx, &list,
		`SELECT * FROM bbs_attach ORDER BY aid DESC LIMIT ? OFFSET ?`, pageSize, offset)
	if err != nil {
		return nil, fmt.Errorf("AttachFind: %w", err)
	}
	if list == nil {
		list = []Attach{}
	}
	return list, nil
}

// AttachType 根据文件扩展名判断附件类型
// 对应 PHP: attach_type($name, $types)
func AttachType(name string, types map[string][]string) string {
	ext := ""
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			ext = name[i+1:]
			break
		}
	}
	if ext == "" {
		return "other"
	}
	for typ, exts := range types {
		if typ == "all" {
			continue
		}
		for _, e := range exts {
			if ext == e {
				return typ
			}
		}
	}
	return "other"
}
