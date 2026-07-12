// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"xiuno/core"
	"xiuno/model"
)

// AdminGroupListHandler GET /api/v1/admin/group
// 获取所有用户组列表（带缓存）
func AdminGroupListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groups, err := model.GetGroupListWithCache(r.Context(), app.Cache, app.DB)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "获取用户组列表失败")
			return
		}
		core.JSONSuccess(w, groups)
	}
}

// AdminGroupReadHandler GET /api/v1/admin/group/{gid}
// 获取单个用户组详情（带缓存）
func AdminGroupReadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gid, _ := strconv.ParseUint(chi.URLParam(r, "gid"), 10, 32)
		if gid == 0 {
			core.JSONError(w, 400, "参数异常")
			return
		}
		g, err := model.GetGroupWithCache(r.Context(), app.Cache, app.DB, uint32(gid))
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "用户组不存在")
			return
		}
		core.JSONSuccess(w, g)
	}
}

// AdminGroupCreateHandler POST /api/v1/admin/group
// 创建用户组
func AdminGroupCreateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var g model.Group
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			core.JSONError(w, 400, "参数异常")
			return
		}
		if g.Name == "" {
			core.JSONError(w, 400, "用户组名称不能为空")
			return
		}
		if err := model.CreateGroup(r.Context(), app.DB, &g); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "创建失败")
			return
		}
		// 失效用户组列表缓存
		model.InvalidateGroupListCache(r.Context(), app.Cache)
		core.JSONSuccess(w, nil)
	}
}

// AdminGroupUpdateHandler PUT /api/v1/admin/group/{gid}
// 更新用户组
func AdminGroupUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gid, _ := strconv.ParseUint(chi.URLParam(r, "gid"), 10, 32)
		if gid == 0 {
			core.JSONError(w, 400, "参数异常")
			return
		}

		var g model.Group
		if err := json.NewDecoder(r.Body).Decode(&g); err != nil {
			core.JSONError(w, 400, "参数异常")
			return
		}
		if g.Name == "" {
			core.JSONError(w, 400, "用户组名称不能为空")
			return
		}
		if err := model.UpdateGroup(r.Context(), app.DB, uint32(gid), &g); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "更新失败")
			return
		}
		// 失效用户组缓存
		model.InvalidateGroupRelated(r.Context(), app.Cache, uint32(gid))
		core.JSONSuccess(w, nil)
	}
}

// AdminGroupDeleteHandler DELETE /api/v1/admin/group/{gid}
// 删除用户组
func AdminGroupDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gid, _ := strconv.ParseUint(chi.URLParam(r, "gid"), 10, 32)
		if gid == 0 {
			core.JSONError(w, 400, "参数异常")
			return
		}
		// 保护系统组不被删除
		if gid < 100 {
			core.JSONError(w, 403, "系统内置用户组不可删除")
			return
		}
		if err := model.DeleteGroup(r.Context(), app.DB, uint32(gid)); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "删除失败")
			return
		}
		// 失效用户组缓存
		model.InvalidateGroupRelated(r.Context(), app.Cache, uint32(gid))
		core.JSONSuccess(w, nil)
	}
}
