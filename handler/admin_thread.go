// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"xiuno/core"
	"xiuno/model"
)

// AdminThreadScanReq 后台主题扫描请求参数
type AdminThreadScanReq struct {
	FID             uint32 `json:"fid"`
	UID             uint32 `json:"uid"`
	Username        string `json:"username"`
	UserIP          string `json:"userip"`
	Keyword         string `json:"keyword"`
	CreateDateStart int64  `json:"create_date_start"`
	CreateDateEnd   int64  `json:"create_date_end"`
	Page            int    `json:"page"`
}

// AdminThreadScanHandler POST /api/v1/admin/thread/scan
// 后台主题扫描：按条件扫描全表，将匹配的 TID 存入队列
// 对应 PHP: admin/route/thread.php action=scan
func AdminThreadScanHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "请先登录")
			return
		}

		var req AdminThreadScanReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "数据格式异常")
			return
		}

		// 如果传了 username 但没有传 uid，尝试查找用户
		if req.UID == 0 && req.Username != "" {
			user, err := model.GetUserByAccount(r.Context(), app.DB, req.Username)
			if err == nil && user != nil {
				req.UID = uint32(user.UID)
			}
		}

		// 生成队列 ID（使用当前时间戳）
		queueid := uint32(time.Now().Unix())

		pageSize := 100
		page := req.Page
		if page < 1 {
			page = 1
		}

		// 第1页时销毁旧队列
		if page == 1 {
			_ = model.QueueDestroy(r.Context(), app.DB, queueid)
		}

		// 扫描该页
		threadlist, err := model.ThreadFindByFID(r.Context(), app.DB, req.FID, page, pageSize, "tid")
		if err != nil {
			core.JSONError(w, 500, "扫描主题失败")
			return
		}

		var matchedTIDs []int64
		for _, thread := range threadlist {
			// 逐条件过滤
			if req.FID > 0 && uint32(thread.FID) != req.FID {
				continue
			}
			if req.CreateDateStart > 0 && thread.CreateDate < req.CreateDateStart {
				continue
			}
			if req.CreateDateEnd > 0 && thread.CreateDate > req.CreateDateEnd {
				continue
			}
			if req.UID > 0 && uint32(thread.UID) != req.UID {
				continue
			}
			if req.Keyword != "" && !strings.Contains(thread.Subject, req.Keyword) {
				continue
			}

			// 匹配，加入队列
			if err := model.QueuePush(r.Context(), app.DB, queueid, thread.TID, 86400); err != nil {
				// 单条失败不阻塞整体
				continue
			}
			matchedTIDs = append(matchedTIDs, thread.TID)
		}

		core.JSONSuccess(w, map[string]interface{}{
			"queueid": queueid,
			"tids":    matchedTIDs,
			"page":    page,
		})
	}
}

// AdminThreadOperationReq 后台主题批量操作请求
type AdminThreadOperationReq struct {
	QueueID uint32 `json:"queueid"`
	Action  string `json:"action"` // delete, close, open
	Limit   int    `json:"limit"`  // 本次处理数量，默认 100
}

// AdminThreadOperationHandler POST /api/v1/admin/thread/operation
// 后台主题批量操作：从队列中弹出 TID 并执行操作
// 对应 PHP: admin/route/thread.php action=operation
func AdminThreadOperationHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "请先登录")
			return
		}

		var req AdminThreadOperationReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "数据格式异常")
			return
		}

		if req.QueueID == 0 {
			core.JSONError(w, 400, "队列不存在")
			return
		}

		limit := req.Limit
		if limit <= 0 || limit > 100 {
			limit = 100
		}

		var processedTIDs []int64
		for i := 0; i < limit; i++ {
			v, ok, err := model.QueuePop(r.Context(), app.DB, req.QueueID)
			if err != nil || !ok {
				break
			}
			tid := uint32(v)

			switch req.Action {
			case "delete":
				// 软删除
				var deletedFID uint32
				err = app.Tx(func(tx *sqlx.Tx) error {
					var txErr error
					deletedFID, txErr = model.SoftDeleteThread(r.Context(), tx, tid)
					return txErr
				})
				if err != nil {
					continue
				}
				// 记录版务日志
				_ = model.CreateModLog(r.Context(), app.DB, claims.UID, tid, 0, "批量删除", "delete", "后台批量删除")
				// 失效缓存
				model.InvalidateThreadCache(r.Context(), app.Cache, tid)
				model.InvalidateForumListCache(r.Context(), app.Cache)
				model.InvalidateForumCache(r.Context(), app.Cache, deletedFID)

			case "close":
				err = app.Tx(func(tx *sqlx.Tx) error {
					return model.ModerateThread(r.Context(), tx, tid, "close", 1)
				})
				if err != nil {
					continue
				}
				_ = model.CreateModLog(r.Context(), app.DB, claims.UID, tid, 0, "批量关闭", "close", "后台批量关闭")
				model.InvalidateThreadCache(r.Context(), app.Cache, tid)

			case "open":
				err = app.Tx(func(tx *sqlx.Tx) error {
					return model.ModerateThread(r.Context(), tx, tid, "close", 0)
				})
				if err != nil {
					continue
				}
				_ = model.CreateModLog(r.Context(), app.DB, claims.UID, tid, 0, "批量打开", "open", "后台批量打开")
				model.InvalidateThreadCache(r.Context(), app.Cache, tid)
			}

			processedTIDs = append(processedTIDs, int64(tid))
		}

		core.JSONSuccess(w, map[string]interface{}{
			"tids":  processedTIDs,
			"count": len(processedTIDs),
		})
	}
}

// AdminThreadFoundHandler GET /api/v1/admin/thread/found?queueid=X&page=1
// 后台查看已查找的主题列表
// 对应 PHP: admin/route/thread.php action=found
func AdminThreadFoundHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		queueidStr := r.URL.Query().Get("queueid")
		queueid, err := strconv.ParseUint(queueidStr, 10, 32)
		if err != nil || queueid == 0 {
			core.JSONError(w, 400, "队列不存在")
			return
		}

		pageStr := r.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		pageSize := 100

		// 获取队列总数
		total, err := model.QueueCount(r.Context(), app.DB, uint32(queueid))
		if err != nil {
			core.JSONError(w, 500, "查询队列失败")
			return
		}

		// 获取该页 TID 列表
		tids, err := model.QueueFind(r.Context(), app.DB, uint32(queueid), page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "查询队列失败")
			return
		}

		// 批量查询帖子详情
		threadlist, err := model.ThreadFindByTIDs(r.Context(), app.DB, tids)
		if err != nil {
			core.JSONError(w, 500, "查询主题失败")
			return
		}

		core.JSONSuccess(w, map[string]interface{}{
			"list":       threadlist,
			"total":      total,
			"page":       page,
			"page_size":  pageSize,
			"queueid":    queueid,
			"total_page": (total + pageSize - 1) / pageSize,
		})
	}
}

// AdminThreadDeleteHandler DELETE /api/v1/admin/thread/{tid}
// 后台硬删除主题（级联删除）
// 对应 PHP: admin/route/thread.php 中的 thread_delete
func AdminThreadDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, err := strconv.ParseUint(tidStr, 10, 32)
		if err != nil {
			core.JSONError(w, 400, "无效的帖子 ID")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "请先登录")
			return
		}

		// 获取帖子信息
		thread, err := model.GetThreadDetail(r.Context(), app.DB, uint32(tid))
		if err != nil {
			if err == sql.ErrNoRows {
				core.JSONError(w, 404, "帖子不存在")
			} else {
				core.JSONError(w, 500, "查询帖子失败")
			}
			return
		}

		// 级联硬删除
		err = app.Tx(func(tx *sqlx.Tx) error {
			return model.CascadeDeleteThread(r.Context(), tx, uint32(tid))
		})
		if err != nil {
			core.JSONError(w, 500, "删除失败")
			return
		}

		// 记录版务日志
		_ = model.CreateModLog(r.Context(), app.DB, claims.UID, uint32(tid), 0, thread.Subject, "delete", "后台硬删除主题")

		// 失效缓存
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(tid))
		model.InvalidateForumListCache(r.Context(), app.Cache)
		if thread != nil {
			model.InvalidateForumCache(r.Context(), app.Cache, uint32(thread.FID))
		}

		core.JSONSuccess(w, nil)
	}
}
