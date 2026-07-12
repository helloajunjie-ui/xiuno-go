// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"

	"xiuno/core"
	"xiuno/model"
)

// AdminForumListHandler GET /api/v1/admin/forum
// 返回所有版块列表（不做权限过滤），供后台管理使用
// 区别于公开 GET /api/v1/forum（会按用户权限过滤不可读版块）
func AdminForumListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		forums, err := model.ForumFind(r.Context(), app.DB)
		if err != nil {
			core.JSONErrorLog(w, 500, "获取版块列表失败", err)
			return
		}
		if forums == nil {
			forums = []model.Forum{}
		}
		core.JSONSuccess(w, forums)
	}
}

// AdminForumCreateHandler POST /api/v1/admin/forum
func AdminForumCreateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var f model.Forum
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil || f.Name == "" {
			core.JSONError(w, 400, "版块名称不能为空")
			return
		}
		f.CreateDate = time.Now().Unix()

		fid, err := model.CreateForum(r.Context(), app.DB, &f)
		if err != nil {
			core.JSONErrorLog(w, 500, "创建版块失败", err)
			return
		}
		// 失效版块列表缓存
		model.InvalidateForumListCache(r.Context(), app.Cache)
		core.JSONSuccess(w, map[string]uint32{"fid": fid})
	}
}

// AdminForumUpdateHandler PUT /api/v1/admin/forum/{fid}
func AdminForumUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid, _ := strconv.ParseUint(chi.URLParam(r, "fid"), 10, 32)
		if fid == 0 {
			core.JSONError(w, 400, "版块不存在")
			return
		}

		var f model.Forum
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil || f.Name == "" {
			core.JSONError(w, 400, "版块名称不能为空")
			return
		}

		if err := model.UpdateForum(r.Context(), app.DB, uint32(fid), &f); err != nil {
			core.JSONError(w, 500, "更新版块失败")
			return
		}
		// 失效版块缓存
		model.InvalidateForumRelated(r.Context(), app.Cache, uint32(fid))
		core.JSONSuccess(w, nil)
	}
}

// AdminForumDeleteHandler DELETE /api/v1/admin/forum/{fid}
func AdminForumDeleteHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid, _ := strconv.ParseUint(chi.URLParam(r, "fid"), 10, 32)
		if fid == 0 {
			core.JSONError(w, 400, "版块不存在")
			return
		}

		// 架构高标拦截：如果版块里还有帖子，绝对不许删
		forum, err := model.GetForum(r.Context(), app.DB, uint32(fid))
		if err != nil {
			core.JSONError(w, 404, "版块不存在")
			return
		}
		if forum.Threads > 0 {
			core.JSONError(w, 403, "该版块下还有帖子，请先转移或清理帖子后再删除")
			return
		}

		if err := model.DeleteForum(r.Context(), app.DB, uint32(fid)); err != nil {
			core.JSONError(w, 500, "删除失败")
			return
		}
		// 失效版块缓存
		model.InvalidateForumRelated(r.Context(), app.Cache, uint32(fid))
		core.JSONSuccess(w, nil)
	}
}

// ==================== 版块权限管理 ====================

// AdminForumAccessListHandler GET /api/v1/admin/forum/{fid}/access
// 返回该版块的所有权限配置列表
func AdminForumAccessListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid, _ := strconv.ParseUint(chi.URLParam(r, "fid"), 10, 32)
		if fid == 0 {
			core.JSONError(w, 400, "版块不存在")
			return
		}

		accessList, err := model.ForumAccessFindByFID(r.Context(), app.DB, uint32(fid))
		if err != nil {
			core.JSONError(w, 500, "查询版块权限失败")
			return
		}
		if accessList == nil {
			accessList = []model.ForumAccess{}
		}

		// 同时返回所有用户组列表（带缓存），供前端选择
		groups, err := model.GetGroupListWithCache(r.Context(), app.Cache, app.DB)
		if err != nil {
			groups = []model.Group{}
		}

		core.JSONSuccess(w, map[string]interface{}{
			"access_list": accessList,
			"groups":      groups,
		})
	}
}

// AdminForumAccessUpdateHandler PUT /api/v1/admin/forum/{fid}/access
// 批量更新版块权限配置
func AdminForumAccessUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fid, _ := strconv.ParseUint(chi.URLParam(r, "fid"), 10, 32)
		if fid == 0 {
			core.JSONError(w, 400, "版块不存在")
			return
		}

		var req struct {
			AccessOn   int                 `json:"accesson"`
			AccessList []model.ForumAccess `json:"access_list"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "请求格式错误")
			return
		}

		// 开启事务
		err := app.Tx(func(tx *sqlx.Tx) error {
			// 1. 更新版块的 accesson 开关
			_, err := tx.ExecContext(r.Context(),
				`UPDATE bbs_forum SET accesson=? WHERE fid=?`, req.AccessOn, fid)
			if err != nil {
				return err
			}

			// 2. 删除该版块所有旧权限记录
			_, err = tx.ExecContext(r.Context(),
				`DELETE FROM bbs_forum_access WHERE fid=?`, fid)
			if err != nil {
				return err
			}

			// 3. 批量插入新权限记录
			if req.AccessOn == 1 && len(req.AccessList) > 0 {
				for _, a := range req.AccessList {
					_, err = tx.ExecContext(r.Context(),
						`INSERT INTO bbs_forum_access (fid, gid, allowread, allowthread, allowpost, allowattach, allowdown)
						 VALUES (?, ?, ?, ?, ?, ?, ?)`,
						fid, a.GID, a.AllowRead, a.AllowThread, a.AllowPost, a.AllowAttach, a.AllowDown)
					if err != nil {
						return err
					}
				}
			}
			return nil
		})

		if err != nil {
			core.JSONError(w, 500, "更新版块权限失败")
			return
		}
		// 失效该版块的所有权限缓存
		model.InvalidateAccessCacheByFID(r.Context(), app.Cache, uint32(fid))
		core.JSONSuccess(w, nil)
	}
}
