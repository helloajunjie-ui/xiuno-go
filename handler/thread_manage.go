// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"xiuno/core"
	"xiuno/model"
)

// ThreadUpdateReq 修改帖子请求体
type ThreadUpdateReq struct {
	Subject string `json:"subject"`
	Message string `json:"message"`
}

// ThreadUpdateHandler PUT /api/v1/thread/{tid}
// 修改主帖标题和首帖内容
func ThreadUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, err := strconv.ParseUint(tidStr, 10, 32)
		if err != nil {
			core.JSONError(w, 400, "无效的帖子 ID")
			return
		}

		var req ThreadUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "数据格式异常")
			return
		}

		req.Subject = strings.TrimSpace(req.Subject)
		req.Message = strings.TrimSpace(req.Message)
		if req.Subject == "" || req.Message == "" {
			core.JSONError(w, 400, "标题和内容不能为空")
			return
		}

		// 1. 获取当前用户和目标帖子
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, 401, "请先登录")
			return
		}

		thread, err := model.GetThreadDetail(r.Context(), app.DB, uint32(tid))
		if err != nil {
			if err == sql.ErrNoRows {
				core.JSONError(w, 404, "帖子不存在")
			} else {
				core.JSONError(w, 500, "查询帖子失败")
			}
			return
		}

		// 2. Policy 权限校验
		if !app.Policy.CanManageThread(claims.UID, claims.GID, uint32(thread.UID), uint32(thread.FID)) {
			core.JSONError(w, 403, "你没有权限修改此贴")
			return
		}

		// 3. 执行修改事务
		err = app.Tx(func(tx *sqlx.Tx) error {
			return model.UpdateThreadContent(r.Context(), tx, uint32(tid), uint32(thread.FirstPID), req.Subject, req.Message)
		})
		if err != nil {
			core.JSONError(w, 500, "修改失败")
			return
		}

		// 失效帖子详情缓存
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(tid))
		core.JSONSuccess(w, nil)
	}
}

// ThreadDeleteHandler DELETE /api/v1/thread/{tid}
// 软删除主帖及旗下所有回复
func ThreadDeleteHandler(app *core.AppCtx) http.HandlerFunc {
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

		// 获取帖子基本信息（只需要 UID 和 FID 做权限校验）
		thread, err := model.GetThreadDetail(r.Context(), app.DB, uint32(tid))
		if err != nil {
			if err == sql.ErrNoRows {
				core.JSONError(w, 404, "帖子不存在")
			} else {
				core.JSONError(w, 500, "查询帖子失败")
			}
			return
		}

		// Policy 权限校验
		if !app.Policy.CanManageThread(claims.UID, claims.GID, uint32(thread.UID), uint32(thread.FID)) {
			core.JSONError(w, 403, "无权删除此贴")
			return
		}

		// 软删除事务
		var deletedFID uint32
		err = app.Tx(func(tx *sqlx.Tx) error {
			var txErr error
			deletedFID, txErr = model.SoftDeleteThread(r.Context(), tx, uint32(tid))
			return txErr
		})
		if err != nil {
			core.JSONError(w, 500, "删除失败")
			return
		}

		// 记录版务日志（先记录日志，再扣减计数器，确保日志写入失败时计数器不受影响）
		if logErr := model.CreateModLog(r.Context(), app.DB, claims.UID, uint32(tid), 0, thread.Subject, "delete", "软删除主题"); logErr != nil {
			core.JSONError(w, 500, "删除失败")
			return
		}

		// 异步扣减统计数（GREATEST 防 unsigned 溢出）
		// 注意：版块统计已在 SoftDeleteThread 事务中直接更新，此处不再重复扣减
		// 用户发帖数和帖子回复数仍通过异步计数器扣减
		app.Counter.DecrUserThread(uint32(thread.UID))
		// 帖子下的所有回帖也被软删除了，扣减帖子回复数
		app.Counter.DecrThreadPost(uint32(tid))

		// 失效帖子详情缓存
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(tid))
		// 失效版块缓存（使版块列表的计数立即更新）
		model.InvalidateForumListCache(r.Context(), app.Cache)
		model.InvalidateForumCache(r.Context(), app.Cache, deletedFID)
		core.JSONSuccess(w, nil)
	}
}
