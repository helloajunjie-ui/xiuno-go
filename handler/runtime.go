// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"net/http"

	"xiuno/core"
	"xiuno/model"
)

// RuntimeHandler GET /api/v1/runtime
// 返回站点运行时统计（用户数、帖子数、今日统计等）
func RuntimeHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rt, err := model.GetRuntime(r.Context(), app.DB, app.Cache)
		if err != nil {
			core.JSONError(w, 500, "获取统计失败")
			return
		}
		core.JSONSuccess(w, rt)
	}
}
