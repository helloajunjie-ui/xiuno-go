package handler

import (
	"encoding/json"
	"net/http"

	"xiuno/core"
	"xiuno/model"
)

// PluginInfo Admin API 返回的插件信息
type PluginInfo struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Version string `json:"version"`
	Desc    string `json:"desc"`
	Active  bool   `json:"active"`
}

// AdminPluginListHandler GET /api/v1/admin/plugin
// 返回所有已注册插件及其启用状态
func AdminPluginListHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		plugins := app.Hook.GetPlugins()
		list := make([]PluginInfo, 0, len(plugins))
		for _, p := range plugins {
			list = append(list, PluginInfo{
				Name:    p.Name(),
				Title:   p.Title(),
				Version: p.Version(),
				Desc:    p.Desc(),
				Active:  app.Hook.IsPluginActive(p.Name()),
			})
		}
		core.JSONSuccess(w, list)
	}
}

// AdminPluginToggleReq 切换插件启用状态请求体
type AdminPluginToggleReq struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}

// AdminPluginToggleHandler PUT /api/v1/admin/plugin
// 切换单个插件的启用/禁用状态，持久化到 bbs_kv 并热刷新
func AdminPluginToggleHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AdminPluginToggleReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数格式错误")
			return
		}

		if req.Name == "" {
			core.JSONError(w, http.StatusBadRequest, "插件名不能为空")
			return
		}

		// 检查插件是否存在
		plugins := app.Hook.GetPlugins()
		if _, ok := plugins[req.Name]; !ok {
			core.JSONError(w, http.StatusNotFound, "插件不存在")
			return
		}

		// 读取当前 active_plugins 配置
		ctx := r.Context()
		kv, err := model.LoadAllKV(ctx, app.DB)
		if err != nil {
			// 表为空，初始化
			kv = make(map[string]string)
		}

		var enabled []string
		raw, ok := kv["active_plugins"]
		if ok && raw != "" {
			if err := json.Unmarshal([]byte(raw), &enabled); err != nil {
				enabled = nil
			}
		}

		// 更新列表
		if req.Active {
			// 启用：确保在列表中
			found := false
			for _, name := range enabled {
				if name == req.Name {
					found = true
					break
				}
			}
			if !found {
				enabled = append(enabled, req.Name)
			}
		} else {
			// 禁用：从列表中移除
			newList := make([]string, 0, len(enabled))
			for _, name := range enabled {
				if name != req.Name {
					newList = append(newList, name)
				}
			}
			enabled = newList
		}

		// 序列化并持久化
		data, _ := json.Marshal(enabled)
		if err := model.SetKV(ctx, app.DB, "active_plugins", string(data)); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "配置持久化失败")
			return
		}

		// 热刷新 Hook 状态
		kv["active_plugins"] = string(data)
		app.Hook.ReloadActivePlugins(kv)

		core.JSONSuccess(w, map[string]interface{}{
			"name":   req.Name,
			"active": req.Active,
		})
	}
}
