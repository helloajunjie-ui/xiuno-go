// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"net/http"
	"strconv"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
)

// TagListHandler 标签列表 GET /api/v1/tag
func TagListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize := 50

		tags, err := model.TagList(r.Context(), app.DB, page, pageSize)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "获取标签列表失败")
			return
		}
		if tags == nil {
			tags = []model.Tag{}
		}

		total, err := model.TagCount(r.Context(), app.DB)
		if err != nil {
			total = 0
		}

		core.JSONSuccess(w, map[string]interface{}{
			"tags":  tags,
			"total": total,
			"page":  page,
		})
	}
}

// TagReadHandler 读取单个标签 GET /api/v1/tag/{tagid}
func TagReadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagidStr := chi.URLParam(r, "tagid")
		tagid, err := strconv.ParseUint(tagidStr, 10, 32)
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		tag, err := model.TagRead(r.Context(), app.DB, uint32(tagid))
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "标签不存在")
			return
		}

		core.JSONSuccess(w, tag)
	}
}

// TagThreadListHandler 标签下的主题列表 GET /api/v1/tag/{tagid}/thread
func TagThreadListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tagidStr := chi.URLParam(r, "tagid")
		tagid, err := strconv.ParseUint(tagidStr, 10, 32)
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		if page < 1 {
			page = 1
		}
		pageSize := 20

		threads, err := model.TagFindThreads(r.Context(), app.DB, uint32(tagid), page, pageSize)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "获取主题列表失败")
			return
		}
		if threads == nil {
			threads = []model.TagThreadItem{}
		}

		total, err := model.TagThreadCount(r.Context(), app.DB, uint32(tagid))
		if err != nil {
			total = 0
		}

		core.JSONSuccess(w, map[string]interface{}{
			"threads": threads,
			"total":   total,
			"page":    page,
		})
	}
}
