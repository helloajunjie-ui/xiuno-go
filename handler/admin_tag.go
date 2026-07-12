// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"xiuno/core"
	"xiuno/model"
)

// AdminTagListHandler GET /api/v1/admin/tag?page=1
// 返回所有标签列表（分页，按 thread 数降序）
func AdminTagListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pageStr := r.URL.Query().Get("page")
		page := 1
		if pageStr != "" {
			p, err := strconv.Atoi(pageStr)
			if err == nil && p > 0 {
				page = p
			}
		}
		pageSize := 50

		tags, err := model.TagList(r.Context(), app.DB, page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "获取标签列表失败")
			return
		}
		total, err := model.TagCount(r.Context(), app.DB)
		if err != nil {
			total = 0
		}

		core.JSONSuccess(w, map[string]interface{}{
			"tags":      tags,
			"total":     total,
			"page":      page,
			"page_size": pageSize,
		})
	}
}

// AdminTagCreateHandler POST /api/v1/admin/tag
// 创建标签
func AdminTagCreateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			core.JSONError(w, 400, "标签名称不能为空")
			return
		}

		tagid, err := model.TagCreate(r.Context(), app.DB, req.Name)
		if err != nil {
			core.JSONError(w, 500, "创建标签失败")
			return
		}

		core.JSONSuccess(w, map[string]uint32{"tagid": tagid})
	}
}

// AdminTagUpdateHandler PUT /api/v1/admin/tag/{tagid}
// 更新标签名称
func AdminTagUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagid, _ := strconv.ParseUint(chi.URLParam(r, "tagid"), 10, 32)
		if tagid == 0 {
			core.JSONError(w, 400, "标签不存在")
			return
		}

		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
			core.JSONError(w, 400, "标签名称不能为空")
			return
		}

		// 检查标签是否存在
		tag, err := model.TagRead(r.Context(), app.DB, uint32(tagid))
		if err != nil || tag == nil {
			core.JSONError(w, 404, "标签不存在")
			return
		}

		// 更新名称（直接操作 DB）
		_, err = app.DB.ExecContext(r.Context(),
			`UPDATE bbs_tag SET name=? WHERE tagid=?`, req.Name, tagid)
		if err != nil {
			core.JSONError(w, 500, "更新标签失败")
			return
		}

		core.JSONSuccess(w, nil)
	}
}

// AdminTagDeleteHandler DELETE /api/v1/admin/tag/{tagid}
// 删除标签（同时删除关联关系）
func AdminTagDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagid, _ := strconv.ParseUint(chi.URLParam(r, "tagid"), 10, 32)
		if tagid == 0 {
			core.JSONError(w, 400, "标签不存在")
			return
		}

		// 检查标签是否存在
		tag, err := model.TagRead(r.Context(), app.DB, uint32(tagid))
		if err != nil || tag == nil {
			core.JSONError(w, 404, "标签不存在")
			return
		}

		// 事务：删除关联 + 删除标签
		err = app.Tx(func(tx *sqlx.Tx) error {
			// 删除 thread_tag 关联
			_, err := tx.ExecContext(r.Context(),
				`DELETE FROM bbs_thread_tag WHERE tagid=?`, tagid)
			if err != nil {
				return err
			}
			// 删除标签本身
			_, err = tx.ExecContext(r.Context(),
				`DELETE FROM bbs_tag WHERE tagid=?`, tagid)
			return err
		})
		if err != nil {
			core.JSONError(w, 500, "删除标签失败")
			return
		}

		core.JSONSuccess(w, nil)
	}
}
