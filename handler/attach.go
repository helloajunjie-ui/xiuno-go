package handler

import (
	"context"
	"log"
	"net/http"
	"path/filepath"
	"strconv"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
)

// 安全图片 MIME 白名单（仅允许已知的图片格式）
// 使用 http.DetectContentType 读取文件头 magic number，而非信任 Content-Type 请求头
var safeImageMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

// UploadHandler 多媒体文件上传处理器
// 安全策略：
//   - 5MB 上传上限
//   - http.DetectContentType MIME 嗅探（基于文件头 magic number）
//   - 仅允许白名单图片格式（jpeg/png/gif/webp）
//   - 文件名由系统生成（纳秒时间戳），拒绝用户原始文件名
func UploadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 限制上传大小（5MB）
		r.Body = http.MaxBytesReader(w, r.Body, 5<<20)
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			core.JSONError(w, http.StatusBadRequest, "文件过大或请求格式错误（最大 5MB）")
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "未找到上传文件")
			return
		}
		defer file.Close()

		// 2. MIME 嗅探：读取前 512 字节检测真实文件类型
		buf := make([]byte, 512)
		if _, err := file.Read(buf); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "文件读取失败")
			return
		}
		mimeType := http.DetectContentType(buf)

		ext, ok := safeImageMimes[mimeType]
		if !ok {
			core.JSONError(w, http.StatusBadRequest, "不支持的文件类型，仅允许 JPEG/PNG/GIF/WebP 图片")
			return
		}

		// 3. 重置文件读取位置到开头（MIME 嗅探消耗了前 512 字节）
		if _, err := file.Seek(0, 0); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "文件重置失败")
			return
		}

		// 4. 获取当前用户
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "未认证")
			return
		}

		// 5. 保存文件到存储后端
		relPath, err := app.Storage.Put(file, ext)
		if err != nil {
			log.Printf("[ERROR] 文件保存失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "文件保存失败")
			return
		}

		// 6. 创建附件数据库记录
		att := &model.Attach{
			UID:         int64(claims.UID),
			FileSize:    header.Size,
			Filename:    filepath.Base(relPath),
			OrgFilename: header.Filename,
			FileType:    mimeType,
			IsImage:     1,
		}
		aid, err := model.CreateAttach(r.Context(), app.DB, att)
		if err != nil {
			log.Printf("[ERROR] 附件记录创建失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "附件记录创建失败")
			return
		}

		// 7. 返回结果
		core.JSONSuccess(w, map[string]interface{}{
			"aid": aid,
			"url": app.Storage.GetURL(relPath),
		})
	}
}

// AttachDownloadHandler 附件下载处理器
// 权限检查：如果附件关联了主题，需校验用户对该版块有 read 权限
// 下载计数：异步递增（不阻塞响应）
func AttachDownloadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aid, err := strconv.ParseUint(chi.URLParam(r, "aid"), 10, 32)
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "无效的附件 ID")
			return
		}

		att, err := model.GetAttach(r.Context(), app.DB, uint32(aid))
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "附件不存在")
			return
		}

		// 如果附件关联了主题，校验版块读权限
		if att.TID > 0 {
			var fid uint32
			err = app.DB.GetContext(r.Context(), &fid, "SELECT fid FROM bbs_thread WHERE tid = ?", att.TID)
			if err == nil {
				claims := core.GetClaims(r.Context())
				if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, claims.UID, claims.GID, fid, "read") {
					core.JSONError(w, http.StatusForbidden, "你没有权限下载该版块的附件")
					return
				}
			}
		}

		// 异步递增下载计数
		go model.IncrAttachDownload(context.Background(), app.DB, uint32(aid))

		app.Storage.ServeDownload(w, r, att.Filename, att.OrgFilename)
	}
}

// AttachDeleteHandler 附件删除处理器
// 权限：附件所有者（UID 匹配）或超级管理员（GID=1）
func AttachDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		aid, err := strconv.ParseUint(chi.URLParam(r, "aid"), 10, 32)
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "无效的附件 ID")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "未认证")
			return
		}

		att, err := model.GetAttach(r.Context(), app.DB, uint32(aid))
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "附件不存在")
			return
		}

		// 校验权限：本人或管理员
		if claims.UID != uint32(att.UID) && claims.GID != 1 {
			core.JSONError(w, http.StatusForbidden, "无权删除该附件")
			return
		}

		// 删除数据库记录
		if err := model.DeleteAttach(r.Context(), app.DB, uint32(aid)); err != nil {
			log.Printf("[ERROR] 附件记录删除失败 aid=%d: %v", aid, err)
			core.JSONError(w, http.StatusInternalServerError, "附件删除失败")
			return
		}

		// 删除存储文件（忽略文件不存在的错误）
		_ = app.Storage.Delete(att.Filename)

		core.JSONSuccess(w, nil)
	}
}
