// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"encoding/json"
	"net/http"

	"xiuno/core"
	"xiuno/model"
)

// ThemeHandler GET /api/v1/theme
// 从内存缓存中读取站点主题配置，0 DB 压力
// 公开接口，无需认证
func ThemeHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		theme := getSiteTheme(app, r.Context())
		core.JSONSuccess(w, theme)
	}
}

// AdminThemeUpdateReq 管理员更新主题配置请求体
type AdminThemeUpdateReq struct {
	PrimaryColor string `json:"primary_color"`
	BgColor      string `json:"bg_color"`
	CardRadius   string `json:"card_radius"`
	ListLayout   string `json:"list_layout"`
	ThemeMode    string `json:"theme_mode"`
	CustomCSS    string `json:"custom_css"`
}

// AdminThemeUpdateHandler PUT /api/v1/admin/theme
// 热更新站点主题：写入 DB + 刷新内存缓存
// 仅 GID=1（管理员组）可操作
func AdminThemeUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		if claims.GID != 1 {
			core.JSONError(w, http.StatusForbidden, "仅管理员可修改站点主题")
			return
		}

		var req AdminThemeUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数格式错误")
			return
		}

		// 构造新的 SiteTheme
		theme := &model.SiteTheme{
			PrimaryColor: req.PrimaryColor,
			BgColor:      req.BgColor,
			CardRadius:   req.CardRadius,
			ListLayout:   req.ListLayout,
			ThemeMode:    req.ThemeMode,
			CustomCSS:    req.CustomCSS,
		}

		// 序列化为 JSON 字符串
		raw, err := json.Marshal(theme)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "主题序列化失败")
			return
		}

		// 写入 DB（单条 KV，key = "site_theme"）
		if err := model.KVCacheSet(r.Context(), app.Cache, app.DB, "site_theme", string(raw)); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "主题持久化失败")
			return
		}

		// 刷新内存缓存
		setSiteTheme(app, r.Context(), theme)

		core.JSONSuccess(w, theme)
	}
}
