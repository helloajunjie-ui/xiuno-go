package handler

import (
	"net/http"
	"strconv"

	"xiuno/core"
	"xiuno/model"
)

// AdminModLogListHandler GET /api/v1/admin/modlog?action=&page=1
func AdminModLogListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize := 20

		list, total, err := model.FindModLog(r.Context(), app.DB, action, page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "查询版务日志失败")
			return
		}

		core.JSONSuccess(w, map[string]interface{}{
			"list":      list,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}
