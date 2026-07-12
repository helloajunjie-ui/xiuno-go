// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"net/http"
	"strconv"

	"xiuno/core"
	"xiuno/model"
)

// MyProfileHandler 获取当前登录用户的个人中心首页数据
// GET /api/v1/my/profile
// 对应 PHP: my.php?action=空 → 返回当前用户完整信息（含统计）
func MyProfileHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil || claims.UID == 0 {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		uid := claims.UID

		user, err := model.GetUserWithCache(r.Context(), app.Cache, app.DB, uid)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "获取用户信息失败")
			return
		}

		// 获取用户组名
		groupName, _ := model.GroupName(r.Context(), app.DB, uint32(user.GID))

		// 从 Storage 驱动获取上传基础 URL 和路径
		uploadURL := app.Storage.GetURL("")
		// 去掉末尾的 /
		if len(uploadURL) > 0 && uploadURL[len(uploadURL)-1] == '/' {
			uploadURL = uploadURL[:len(uploadURL)-1]
		}
		uploadPath := "upload"

		// 格式化用户数据（填充头像 URL、日期格式等）
		model.UserFormat(user, groupName, uploadURL, uploadPath)

		// 返回安全信息（移除密码等敏感字段）
		safe := model.UserSafeInfo(user)

		core.JSONSuccess(w, safe)
	}
}

// MyThreadListHandler 获取当前登录用户参与的主题列表
// GET /api/v1/my/thread?page=1
// 对应 PHP: my.php?action=thread → mythread_find_by_uid()
func MyThreadListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil || claims.UID == 0 {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		uid := claims.UID

		pageStr := r.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		pageSize := 20

		list, err := model.MyThreadFindByUID(r.Context(), app.DB, uid, page, pageSize)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "查询失败")
			return
		}

		total, err := model.MyThreadCountByUID(r.Context(), app.DB, uid)
		if err != nil {
			total = 0
		}

		core.JSONSuccess(w, map[string]interface{}{
			"list":  list,
			"total": total,
			"page":  page,
		})
	}
}
