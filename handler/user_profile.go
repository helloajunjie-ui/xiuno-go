// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"xiuno/core"
	"xiuno/model"
)

// UserProfileHandler GET /api/v1/user/{uid}
// 返回用户公开资料（带缓存）
func UserProfileHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr := chi.URLParam(r, "uid")
		uid, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil {
			core.JSONError(w, 400, "无效的用户 ID")
			return
		}

		user, err := model.GetUserWithCache(r.Context(), app.Cache, app.DB, uint32(uid))
		if err != nil {
			core.JSONError(w, 404, "该用户已遁入虚空")
			return
		}

		core.JSONSuccess(w, user)
	}
}

// UserAvatarUploadHandler POST /api/v1/user/avatar
// 上传用户头像，2MB 限制，MIME 嗅探，覆盖写入固定路径
func UserAvatarUploadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(2 << 20); err != nil {
			core.JSONError(w, 400, "图片体积过大")
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			core.JSONError(w, 400, "无法读取图片")
			return
		}
		defer file.Close()

		// MIME 嗅探：读取前 512 字节检测真实类型
		buf := make([]byte, 512)
		n, _ := file.Read(buf)
		mimeType := http.DetectContentType(buf[:n])
		if mimeType != "image/jpeg" && mimeType != "image/png" &&
			mimeType != "image/gif" && mimeType != "image/webp" {
			core.JSONError(w, 415, "只能上传合法格式的图片")
			return
		}
		// 重置文件指针到开头
		file.Seek(0, 0)

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "未登录")
			return
		}
		uid := claims.UID

		// 计算固定头像路径并覆盖写入
		relPath := model.GetAvatarPath(uid)
		if err := app.Storage.PutFixedPath(file, relPath); err != nil {
			core.JSONError(w, 500, "头像保存失败")
			return
		}

		// 更新时间戳（前端拼接 ?t=timestamp 解决缓存）
		now := time.Now().Unix()
		_ = model.UpdateUserAvatar(r.Context(), app.DB, uid, now)

		// 失效用户缓存
		model.InvalidateUserCache(r.Context(), app.Cache, uid)

		core.JSONSuccess(w, map[string]interface{}{
			"avatar": now,
			"url":    app.Storage.GetURL(relPath) + "?t=" + strconv.FormatInt(now, 10),
		})
	}
}

// UserThreadListHandler GET /api/v1/user/{uid}/thread
// 获取指定用户的帖子列表
func UserThreadListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr := chi.URLParam(r, "uid")
		uid, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil {
			core.JSONError(w, 400, "无效的用户 ID")
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}

		list, err := model.GetUserThreadList(r.Context(), app.DB, uint32(uid), page, 20)
		if err != nil {
			core.JSONError(w, 500, "获取帖子列表失败")
			return
		}

		core.JSONSuccess(w, map[string]interface{}{
			"list":     list,
			"has_more": len(list) == 20,
			"page":     page,
		})
	}
}

// UserPostListHandler GET /api/v1/user/{uid}/post
// 获取指定用户的回帖列表
func UserPostListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr := chi.URLParam(r, "uid")
		uid, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil {
			core.JSONError(w, 400, "无效的用户 ID")
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}

		list, err := model.GetUserPostList(r.Context(), app.DB, uint32(uid), page, 20)
		if err != nil {
			core.JSONError(w, 500, "获取回帖列表失败")
			return
		}

		core.JSONSuccess(w, map[string]interface{}{
			"list":     list,
			"has_more": len(list) == 20,
			"page":     page,
		})
	}
}
