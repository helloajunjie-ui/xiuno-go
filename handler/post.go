// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"html"
	"net/http"
	"strconv"
	"strings"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ReplyCreateReq 回帖请求体
type ReplyCreateReq struct {
	Message  string `json:"message"`
	QuotePid uint32 `json:"quotepid"` // 引用的楼层 PID，0 为不引用
	DocType  int    `json:"doctype"`  // 0:HTML, 1:TXT, 2:Markdown（默认）
}

// ReplyCreateHandler POST /api/v1/thread/{tid}/post
func ReplyCreateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, _ := strconv.ParseUint(tidStr, 10, 32)
		if tid == 0 {
			core.JSONError(w, 400, "帖子跑丢了")
			return
		}

		// 从中间件取 UID
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		uid := claims.UID

		// 查出帖子版块 fid 和 closed 状态（一次查询）
		var threadInfo struct {
			FID    int32 `db:"fid"`
			Closed int32 `db:"closed"`
		}
		err := app.DB.GetContext(r.Context(), &threadInfo,
			`SELECT fid, closed FROM bbs_thread WHERE tid = ? AND deleted_at IS NULL`, tid)
		if err != nil {
			core.JSONError(w, http.StatusNotFound, "帖子已失联")
			return
		}
		if threadInfo.Closed == 1 {
			core.JSONError(w, http.StatusForbidden, "该帖子已关闭，无法回复")
			return
		}

		// 版块回帖权限校验（带缓存）
		if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, uid, claims.GID, uint32(threadInfo.FID), "post") {
			core.JSONError(w, 403, "你所在的用户组不允许在此回复")
			return
		}

		var req ReplyCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "格式异常")
			return
		}

		req.Message = strings.TrimSpace(req.Message)
		if len(req.Message) == 0 {
			core.JSONError(w, 400, "回帖内容不能为空")
			return
		}

		userIP := parseUserIP(r)

		// DocType 越权校验与 message_fmt 生成
		doctype := int32(req.DocType)
		if doctype < 0 || doctype > 2 {
			doctype = 2
		}
		if doctype == 0 && claims.GID != 1 {
			doctype = 2 // 非超管强制降级为 Markdown
		}
		var messageFmt string
		if doctype == 0 {
			messageFmt = req.Message // 超管 HTML 直接放行
		} else {
			messageFmt = html.EscapeString(req.Message) // Markdown/纯文本转义后存 message_fmt
		}

		// 插件 Filter 锚点：回帖前内容过滤（敏感词、防灌水等）
		//    如果插件拦截（如敏感词命中），直接返回 400 错误给前端
		filteredData, filterErr := app.Hook.ApplyFilters(r.Context(), "post_create_before", req.Message)
		if filterErr != nil {
			core.JSONError(w, 400, filterErr.Error())
			return
		}
		filteredMessage := filteredData.(string)

		var newPid uint32

		// 执行强一致性回帖事务
		err = app.Tx(func(tx *sqlx.Tx) error {
			var txErr error
			newPid, txErr = model.CreateReply(r.Context(), tx, uint32(tid), uid, model.IP2Long(userIP), filteredMessage, req.QuotePid, doctype, messageFmt)
			return txErr
		})
		if err != nil {
			core.JSONError(w, 500, "回复失败，请稍后再试")
			return
		}

		// 异步计数器更新（无锁狂奔）
		app.Counter.IncrThreadPost(uint32(tid))
		app.Counter.IncrUserPost(uid)

		// 用户组自动升级（根据发帖数）
		model.AutoUpdateUserGroup(r.Context(), app.DB, uid)

		// 附件关联帖子（扫描 message 中的附件 URL，关联到回帖）
		model.AttachAssocPost(r.Context(), app.DB, uint32(tid), newPid, uid, req.Message)

		// 插件 Action 锚点：回帖后旁路动作（积分赠送、通知等）
		app.Hook.DoAction(r.Context(), "post_create_after", map[string]interface{}{
			"pid": newPid,
			"tid": uint32(tid),
			"uid": uid,
		})

		// 返回新楼层的 PID，前端可以利用这个 PID 定位锚点平滑滚动
		core.JSONSuccess(w, map[string]uint32{"pid": newPid})
	}
}

// ReplyListHandler GET /api/v1/thread/{tid}/post?page=1
func ReplyListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, _ := strconv.ParseUint(tidStr, 10, 32)
		if tid == 0 {
			core.JSONError(w, 400, "无效帖子 tid")
			return
		}

		// 先查出帖子归属版块 fid，校验版块访问权限
		var fid uint32
		err := app.DB.GetContext(r.Context(), &fid,
			`SELECT fid FROM bbs_thread WHERE tid = ? AND deleted_at IS NULL`, tid)
		if err != nil {
			core.JSONError(w, 404, "帖子不存在")
			return
		}

		claims := core.GetClaims(r.Context())
		if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, claims.UID, claims.GID, fid, "read") {
			core.JSONError(w, 403, "你没有权限查看该版块的回帖")
			return
		}

		pageStr := r.URL.Query().Get("page")
		page, _ := strconv.Atoi(pageStr)
		if page < 1 {
			page = 1
		}
		const pageSize = 20

		list, err := model.GetPostList(r.Context(), app.DB, uint32(tid), page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "获取回复失败")
			return
		}

		hasMore := len(list) == pageSize

		core.JSONSuccess(w, map[string]interface{}{
			"list":     list,
			"has_more": hasMore,
			"page":     page,
		})
	}
}
