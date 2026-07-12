// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"net/http"
	"strconv"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
)

// ForumListHandler 版块列表
// 根据当前用户权限过滤不可读版块
// 版块列表已通过 model.GetForumListWithCache 缓存
func ForumListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forums, err := model.GetForumListWithCache(r.Context(), app.Cache, app.DB)
		if err != nil {
			core.JSONErrorLog(w, 500, "获取版块列表失败", err)
			return
		}

		// 访问过滤：根据当前用户权限过滤不可读版块（带缓存）
		claims := core.GetClaims(r.Context())
		var filteredForums []model.Forum
		for _, f := range forums {
			if model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, claims.UID, claims.GID, uint32(f.FID), "read") {
				filteredForums = append(filteredForums, f)
			}
		}
		if filteredForums == nil {
			filteredForums = []model.Forum{}
		}

		core.JSONSuccess(w, filteredForums)
	}
}

// ForumReadHandler 读取单个版块（带缓存）
func ForumReadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fidStr := chi.URLParam(r, "fid")
		fid, err := strconv.ParseUint(fidStr, 10, 32)
		if err != nil || fid == 0 {
			core.JSONError(w, 400, "无效版块 fid")
			return
		}

		forum, err := model.GetForumWithCache(r.Context(), app.Cache, app.DB, uint32(fid))
		if err != nil {
			core.JSONError(w, 404, "版块不存在")
			return
		}

		core.JSONSuccess(w, forum)
	}
}
