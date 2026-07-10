package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"xiuno/core"
	"xiuno/model"
)

// UserGroupUpdateReq 修改用户组请求体
type UserGroupUpdateReq struct {
	GID uint16 `json:"gid"`
}

// AdminUserGroupHandler PUT /api/v1/admin/user/{uid}/group
// 修改用户组：gid=7 封禁，gid=101 解封为普通用户
func AdminUserGroupHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uid, _ := strconv.ParseUint(chi.URLParam(r, "uid"), 10, 32)
		if uid == 0 {
			core.JSONError(w, 400, "用户不存在")
			return
		}

		var req UserGroupUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "参数异常")
			return
		}

		// 防止超管把自己关进小黑屋，或自降权限
		claims := core.GetClaims(r.Context())
		if claims.UID == uint32(uid) {
			core.JSONError(w, 403, "不能对自己的账号进行操作")
			return
		}

		if err := model.UpdateUserGroup(r.Context(), app.DB, uint32(uid), req.GID); err != nil {
			core.JSONError(w, 500, "修改用户组失败")
			return
		}

		// 失效用户缓存
		model.InvalidateUserCache(r.Context(), app.Cache, uint32(uid))
		core.JSONSuccess(w, nil)
	}
}

// AdminUserListHandler GET /api/v1/admin/user?page=1
// 管理员获取用户列表（分页，按 uid 降序）
func AdminUserListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		page := 1
		if pageStr != "" {
			p, err := strconv.Atoi(pageStr)
			if err == nil && p > 0 {
				page = p
			}
		}
		const pageSize = 20

		users, err := model.FindUser(r.Context(), app.DB, page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "获取用户列表失败")
			return
		}

		hasMore := len(users) == pageSize
		core.JSONSuccess(w, map[string]interface{}{
			"users":    users,
			"has_more": hasMore,
			"page":     page,
		})
	}
}
