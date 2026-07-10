package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// --- 帖子版务操作（置顶/关闭） ---

// ThreadModReq 版务操作请求体
type ThreadModReq struct {
	Action string `json:"action"` // "top" 或 "close"
	Value  int    `json:"value"`  // top: 0-3, close: 0-1
}

// ThreadModerateHandler POST /api/v1/thread/{tid}/moderate
func ThreadModerateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, _ := strconv.ParseUint(tidStr, 10, 32)
		if tid == 0 {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		var req ThreadModReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}

		if !app.Policy.CanModerateThread(claims.UID, claims.GID) {
			core.JSONError(w, http.StatusForbidden, "权限不足，无法执行版务操作")
			return
		}

		// 事务包裹：版务操作 + 版务日志，保证原子性
		err := app.Tx(func(tx *sqlx.Tx) error {
			if err := model.ModerateThread(r.Context(), tx, uint32(tid), req.Action, req.Value); err != nil {
				return err
			}
			// 在事务内直接写入版务日志
			now := time.Now().Unix()
			_, err := tx.ExecContext(r.Context(), `
				INSERT INTO bbs_modlog (uid, tid, pid, subject, comment, create_date, action)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				claims.UID, uint32(tid), 0, "", req.Action, now, "修改帖子状态")
			return err
		})
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "操作失败")
			return
		}

		// 失效帖子详情缓存（置顶/关闭状态变化）
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(tid))
		core.JSONSuccess(w, nil)
	}
}

// --- 回帖的改/删 ---

// PostUpdateReq 回帖修改请求体
type PostUpdateReq struct {
	Message string `json:"message"`
}

// PostUpdateHandler PUT /api/v1/post/{pid}
func PostUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pidStr := chi.URLParam(r, "pid")
		pid, _ := strconv.ParseUint(pidStr, 10, 32)
		if pid == 0 {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		var req PostUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}
		req.Message = strings.TrimSpace(req.Message)
		if req.Message == "" {
			core.JSONError(w, http.StatusBadRequest, "内容不能为空")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}

		// 查出该 post 的 uid 和 tid 以校验所有权（仅查必需字段，避免 SELECT * 传输大字段）
		var post struct {
			UID int32 `db:"uid"`
			TID int32 `db:"tid"`
		}
		err := app.DB.GetContext(r.Context(), &post,
			`SELECT uid, tid FROM bbs_post WHERE pid = ? AND deleted_at IS NULL`, pid)
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "回帖不存在")
			return
		}

		if !app.Policy.CanManagePost(claims.UID, claims.GID, uint32(post.UID), uint32(post.TID)) {
			core.JSONError(w, http.StatusForbidden, "无权修改该回帖")
			return
		}

		if err := model.UpdatePostContent(r.Context(), app.DB, uint32(pid), req.Message); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "更新失败")
			return
		}
		// 失效帖子详情缓存（回帖内容变化）
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(post.TID))
		core.JSONSuccess(w, nil)
	}
}

// PostDeleteHandler DELETE /api/v1/post/{pid}
func PostDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		pidStr := chi.URLParam(r, "pid")
		pid, _ := strconv.ParseUint(pidStr, 10, 32)
		if pid == 0 {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}

		// 查出该 post 的 uid 和 tid 以校验所有权（仅查必需字段，避免 SELECT * 传输大字段）
		var post struct {
			UID int32 `db:"uid"`
			TID int32 `db:"tid"`
		}
		err := app.DB.GetContext(r.Context(), &post,
			`SELECT uid, tid FROM bbs_post WHERE pid = ? AND deleted_at IS NULL`, pid)
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "回帖不存在")
			return
		}

		if !app.Policy.CanManagePost(claims.UID, claims.GID, uint32(post.UID), uint32(post.TID)) {
			core.JSONError(w, http.StatusForbidden, "无权删除该回帖")
			return
		}

		if err := model.SoftDeletePost(r.Context(), app.DB, uint32(pid)); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "删除失败")
			return
		}

		// 异步计数器扣减（帖子回复数 + 用户回帖数）
		app.Counter.DecrThreadPost(uint32(post.TID))
		app.Counter.DecrUserPost(uint32(post.UID))

		// 记录版务日志
		model.CreateModLog(r.Context(), app.DB, claims.UID, uint32(post.TID), uint32(pid), "", "delete", "软删除回帖")

		// 失效帖子详情缓存（回帖数变化）
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(post.TID))
		core.JSONSuccess(w, nil)
	}
}

// --- 帖子移动 ---

// ThreadMoveReq 帖子移动请求体
type ThreadMoveReq struct {
	NewFid uint32 `json:"new_fid"`
}

// ThreadMoveHandler POST /api/v1/thread/{tid}/move
// 移动帖子到新版块（含版块统计平移）
func ThreadMoveHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, _ := strconv.ParseUint(tidStr, 10, 32)
		if tid == 0 {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		var req ThreadMoveReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.NewFid == 0 {
			core.JSONError(w, http.StatusBadRequest, "目标版块无效")
			return
		}

		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}

		if !app.Policy.CanModerateThread(claims.UID, claims.GID) {
			core.JSONError(w, http.StatusForbidden, "权限不足，无法执行版务操作")
			return
		}

		// 获取当前帖子所在的 oldFid
		thread, err := model.GetThreadDetail(r.Context(), app.DB, uint32(tid))
		if err != nil || thread == nil {
			core.JSONError(w, http.StatusNotFound, "帖子不存在")
			return
		}

		// 目标版块存在性校验
		_, err = model.GetForum(r.Context(), app.DB, req.NewFid)
		if err != nil {
			core.JSONError(w, http.StatusBadRequest, "目标版块不存在")
			return
		}

		// 执行移动事务
		err = app.Tx(func(tx *sqlx.Tx) error {
			return model.MoveThread(r.Context(), tx, uint32(tid), uint32(thread.FID), req.NewFid)
		})
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "移动帖子失败")
			return
		}

		// 记录版务日志
		model.CreateModLog(r.Context(), app.DB, claims.UID, uint32(tid), 0, thread.Subject, "move",
			"移动到版块: "+strconv.Itoa(int(req.NewFid)))

		// 失效帖子详情缓存 + 新旧版块缓存
		model.InvalidateThreadCache(r.Context(), app.Cache, uint32(tid))
		model.InvalidateForumCache(r.Context(), app.Cache, uint32(thread.FID))
		model.InvalidateForumCache(r.Context(), app.Cache, req.NewFid)
		core.JSONSuccess(w, nil)
	}
}
