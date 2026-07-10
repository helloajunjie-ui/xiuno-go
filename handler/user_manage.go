package handler

import (
	"database/sql"
	"net/http"
	"strconv"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
)

// UserDeleteByModHandler DELETE /api/v1/user/{uid}/delete
// 版主删除用户（级联删除所有数据）
// 对应 PHP: mod.php?action=deleteuser&uid=X
// 权限：仅超管(gid=1)和超版(gid=2)可操作，且不能删除管理组成员(gid<6)
func UserDeleteByModHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uidStr := chi.URLParam(r, "uid")
		targetUID, err := strconv.ParseUint(uidStr, 10, 32)
		if err != nil || targetUID == 0 {
			core.JSONError(w, 400, "无效的用户 ID")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "请先登录")
			return
		}

		// 1. 查出目标用户
		targetUser, err := model.GetUserByUID(r.Context(), app.DB, uint32(targetUID))
		if err != nil {
			if err == sql.ErrNoRows {
				core.JSONError(w, 404, "用户不存在或已被删除")
			} else {
				core.JSONError(w, 500, "查询用户失败")
			}
			return
		}

		// 2. 权限校验
		if !app.Policy.CanDeleteUser(claims.UID, claims.GID, targetUser.GID) {
			core.JSONError(w, 403, "无权删除该用户")
			return
		}

		// 3. 不能删除自己
		if claims.UID == uint32(targetUID) {
			core.JSONError(w, 400, "不能删除自己的账号")
			return
		}

		// 4. 执行级联删除
		// uploadDir 与 main.go 中静态文件服务的目录一致
		if err := model.CascadeDeleteUser(r.Context(), app.DB, uint32(targetUID), "./upload"); err != nil {
			core.JSONError(w, 500, "删除用户失败: "+err.Error())
			return
		}

		// 5. 记录版务日志
		model.CreateModLog(r.Context(), app.DB, claims.UID, 0, 0,
			"delete user: "+targetUser.Username, "deleteuser", "版主删除用户")

		core.JSONSuccess(w, nil)
	}
}
