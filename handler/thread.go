// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"html"
	"log"
	"net"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"

	"xiuno/core"
	"xiuno/model"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

// ThreadCreateReq 发帖请求体
type ThreadCreateReq struct {
	Fid     uint32 `json:"fid"`
	Subject string `json:"subject"`
	Message string `json:"message"`
	DocType int    `json:"doctype"` // 0:HTML, 1:TXT, 2:Markdown（默认）
	Tags    string `json:"tags"`    // 逗号分隔的标签名，如 "golang,redis"
}

// ThreadCreateHandler 发帖端点（需挂载 AuthMiddleware）
func ThreadCreateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rcv := recover(); rcv != nil {
				log.Printf("[PANIC] ThreadCreateHandler: %v\n%s", rcv, debug.Stack())
				core.JSONError(w, 500, "发帖失败")
			}
		}()

		// 1. 从 context 提取已登录用户 UID（由 AuthMiddleware 注入）
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		uid := claims.UID

		var req ThreadCreateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "请求格式错误")
			return
		}

		req.Subject = strings.TrimSpace(req.Subject)
		req.Message = strings.TrimSpace(req.Message)
		if len(req.Subject) == 0 || len(req.Message) == 0 {
			core.JSONError(w, 400, "标题和内容不能为空")
			return
		}

		// 1.5 版块发帖权限校验（带缓存）
		if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, uid, claims.GID, req.Fid, "thread") {
			core.JSONError(w, 403, "你所在的用户组不允许在此版块发帖")
			return
		}

		// 2. 解析用户 IP
		userIP := parseUserIP(r)

		// 3. DocType 越权校验与 message_fmt 生成
		//    默认强制 Markdown；非超管传 HTML 则降级为 Markdown
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

		// 4. 插件 Filter 锚点：发帖前内容过滤（敏感词、防灌水等）
		//    如果插件拦截（如敏感词命中），直接返回 400 错误给前端
		filteredData, err := app.Hook.ApplyFilters(r.Context(), "thread_create_before", req.Message)
		if err != nil {
			core.JSONError(w, 400, err.Error())
			return
		}
		filteredMessage := filteredData.(string)

		// 5. 强一致性事务：创建主帖 + 首帖 + 反写 firstpid
		var newTid uint32
		err = app.Tx(func(tx *sqlx.Tx) error {
			var txErr error
			newTid, txErr = model.CreateThreadAndFirstPost(
				r.Context(), tx, req.Fid, uid, model.IP2Long(userIP), req.Subject, filteredMessage, doctype, messageFmt)
			return txErr
		})
		if err != nil {
			log.Printf("[ERROR] ThreadCreateHandler 事务失败: %v", err)
			core.JSONError(w, 500, "发帖失败")
			return
		}

		// 5.5 失效版块缓存（使版块列表和单个版块的计数立即更新）
		model.InvalidateForumListCache(r.Context(), app.Cache)
		model.InvalidateForumCache(r.Context(), app.Cache, req.Fid)

		// 6. 异步计数器更新（移出事务，消除行锁热点）
		// 注意：版块计数已在 CreateThreadAndFirstPost 事务中直接更新 DB，此处不再重复计数
		// 用户发帖数仍通过异步计数器更新（非关键路径，允许短暂不一致）
		app.Counter.IncrUserThread(uid)

		// 6.5 用户组自动升级（根据发帖数）
		model.AutoUpdateUserGroup(r.Context(), app.DB, uid)

		// 6.6 附件关联帖子（扫描 message 中的附件 URL，关联到新帖）
		model.AttachAssocPost(r.Context(), app.DB, newTid, 0, uid, req.Message)

		// 6.7 标签处理（逗号分隔的标签名）
		if req.Tags != "" {
			if err := model.TagSetThreadTags(r.Context(), app.DB, newTid, req.Tags); err != nil {
				log.Printf("[WARN] ThreadCreateHandler 标签处理失败: %v", err)
			}
		}

		// 7. 插件 Action 锚点：发帖后旁路动作（积分赠送、通知等）
		app.Hook.DoAction(r.Context(), "thread_create_after", map[string]interface{}{
			"tid":     newTid,
			"fid":     req.Fid,
			"uid":     uid,
			"subject": req.Subject,
		})

		// 8. 返回新帖 TID
		core.JSONSuccess(w, map[string]uint32{"tid": newTid})
	}
}

// ThreadListHandler 帖子列表 GET /api/v1/thread?fid=1&page=1
// fid 为空时返回全站帖子列表（首页模式）
func ThreadListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fidStr := r.URL.Query().Get("fid")
		pageStr := r.URL.Query().Get("page")

		page := 1
		if pageStr != "" {
			p, err := strconv.Atoi(pageStr)
			if err == nil && p > 0 {
				page = p
			}
		}
		const pageSize = 20

		claims := core.GetClaims(r.Context())

		// fid 为空 → 全站模式，跳过版块权限校验
		if fidStr != "" && fidStr != "0" {
			fid, err := strconv.ParseUint(fidStr, 10, 32)
			if err != nil || fid == 0 {
				core.JSONError(w, 400, "无效版块 fid")
				return
			}

			// 版块读取权限校验（带缓存）
			if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, claims.UID, claims.GID, uint32(fid), "read") {
				core.JSONError(w, 403, "你所在的用户组没有权限访问该版块")
				return
			}

			threads, err := model.GetThreadList(r.Context(), app.DB, uint32(fid), page, pageSize)
			if err != nil {
				core.JSONError(w, 500, "获取帖子列表失败")
				return
			}

			hasMore := len(threads) == pageSize
			core.JSONSuccess(w, map[string]interface{}{
				"threads":  threads,
				"has_more": hasMore,
				"page":     page,
			})
			return
		}

		// 全站模式：fid=0，不校验版块权限
		threads, err := model.GetThreadList(r.Context(), app.DB, 0, page, pageSize)
		if err != nil {
			core.JSONError(w, 500, "获取帖子列表失败")
			return
		}

		hasMore := len(threads) == pageSize
		core.JSONSuccess(w, map[string]interface{}{
			"threads":  threads,
			"has_more": hasMore,
			"page":     page,
		})
	}
}

// parseUserIP 从请求中提取用户 IP，保证返回非 nil 的 net.IP
func parseUserIP(r *http.Request) net.IP {
	ip := net.ParseIP(r.Header.Get("X-Real-IP"))
	if ip != nil {
		return ip
	}
	ip = net.ParseIP(strings.Split(r.RemoteAddr, ":")[0])
	if ip != nil {
		return ip
	}
	return net.IPv4zero
}

// ThreadReadHandler 帖子详情 GET /api/v1/thread/{tid}
// 帖子详情已通过 model.GetThreadDetailWithCache 缓存
func ThreadReadHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tidStr := chi.URLParam(r, "tid")
		tid, err := strconv.ParseUint(tidStr, 10, 32)
		if err != nil || tid == 0 {
			core.JSONError(w, 400, "无效帖子 tid")
			return
		}

		detail, err := model.GetThreadDetailWithCache(r.Context(), app.Cache, app.DB, uint32(tid))
		if err != nil {
			core.JSONError(w, 404, "帖子不存在")
			return
		}

		// 版块读取权限校验（带缓存）
		claims := core.GetClaims(r.Context())
		if !model.CheckForumAccessWithCache(r.Context(), app.Cache, app.DB, claims.UID, claims.GID, uint32(detail.FID), "read") {
			core.JSONError(w, 403, "你所在的用户组没有权限访问该版块")
			return
		}

		// 异步增加浏览数（非阻塞，不等待 DB 写入）
		app.Counter.IncrThreadView(uint32(tid))

		core.JSONSuccess(w, detail)
	}
}
