// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"xiuno/core"
	"xiuno/model"
)

// cacheKeySiteConf 站点配置在 Cache 中的 key
const cacheKeySiteConf = "site_conf"

// cacheKeySiteTheme 站点主题在 Cache 中的 key
const cacheKeySiteTheme = "site_theme"

// ConfigHandler GET /api/v1/config
// 从内存缓存中读取站点配置，0 DB 压力
func ConfigHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		conf := getSiteConf(app, r.Context())
		core.JSONSuccess(w, conf)
	}
}

// AdminConfigUpdateReq 管理员更新配置请求体
type AdminConfigUpdateReq struct {
	SiteName     string `json:"site_name"`
	SiteBrief    string `json:"site_brief"`
	SiteURL      string `json:"site_url"`
	PageSize     int    `json:"page_size"`
	PostPageSize int    `json:"post_page_size"`
	CloseReason  string `json:"close_reason"`
}

// AdminConfigUpdateHandler PUT /api/v1/admin/config
// 热更新站点配置：写入 DB + 刷新内存缓存
// 仅 GID=1（管理员组）可操作
func AdminConfigUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "请先登录")
			return
		}
		if claims.GID != 1 {
			core.JSONError(w, http.StatusForbidden, "仅管理员可修改站点配置")
			return
		}

		var req AdminConfigUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数格式错误")
			return
		}

		// 构造新的 SiteConf
		conf := &model.SiteConf{
			SiteName:     req.SiteName,
			SiteBrief:    req.SiteBrief,
			SiteURL:      req.SiteURL,
			PageSize:     req.PageSize,
			PostPageSize: req.PostPageSize,
			CloseReason:  req.CloseReason,
		}

		// 写入 DB（逐条持久化）
		kvMap := model.SiteConfToMap(conf)
		for k, v := range kvMap {
			if err := model.SetKV(r.Context(), app.DB, k, v); err != nil {
				core.JSONError(w, http.StatusInternalServerError, "配置持久化失败")
				return
			}
		}

		// 刷新内存缓存
		setSiteConf(app, r.Context(), conf)

		core.JSONSuccess(w, conf)
	}
}

// getSiteConf 从缓存读取站点配置，缓存未命中则返回默认值
func getSiteConf(app *core.AppCtx, ctx context.Context) *model.SiteConf {
	data, ok := app.Cache.Get(ctx, cacheKeySiteConf)
	if !ok || data == nil {
		return model.DefaultSiteConf()
	}
	var conf model.SiteConf
	if err := json.Unmarshal(data, &conf); err != nil {
		return model.DefaultSiteConf()
	}
	return &conf
}

// setSiteConf 将站点配置序列化后写入缓存（永不过期）
func setSiteConf(app *core.AppCtx, ctx context.Context, conf *model.SiteConf) {
	data, err := json.Marshal(conf)
	if err != nil {
		return
	}
	app.Cache.Set(ctx, cacheKeySiteConf, data, 0) // 0 = 永不过期
}

// getSiteTheme 从缓存读取站点主题，缓存未命中则返回默认值
func getSiteTheme(app *core.AppCtx, ctx context.Context) *model.SiteTheme {
	data, ok := app.Cache.Get(ctx, cacheKeySiteTheme)
	if !ok || data == nil {
		return model.DefaultSiteTheme()
	}
	var theme model.SiteTheme
	if err := json.Unmarshal(data, &theme); err != nil {
		return model.DefaultSiteTheme()
	}
	return &theme
}

// setSiteTheme 将站点主题序列化后写入缓存（永不过期）
func setSiteTheme(app *core.AppCtx, ctx context.Context, theme *model.SiteTheme) {
	data, err := json.Marshal(theme)
	if err != nil {
		return
	}
	app.Cache.Set(ctx, cacheKeySiteTheme, data, 0) // 0 = 永不过期
}

// InitSiteConf 启动时从 DB 全量加载 bbs_kv 并预热缓存
// 在 NewAppCtx 之后、路由注册之前调用
func InitSiteConf(app *core.AppCtx) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. 预热所有 KV 到缓存（KVCacheWarmUp）
	_ = model.KVCacheWarmUp(ctx, app.Cache, app.DB)

	// 2. 从缓存读取站点配置
	kv, err := model.LoadAllKV(ctx, app.DB)
	if err != nil {
		// bbs_kv 表可能为空，使用默认配置
		setSiteConf(app, ctx, model.DefaultSiteConf())
		setSiteTheme(app, ctx, model.DefaultSiteTheme())
		return
	}

	conf := model.ParseSiteConf(kv)
	setSiteConf(app, ctx, conf)

	// 热刷新插件启用状态（从 bbs_kv 读取 active_plugins）
	app.Hook.ReloadActivePlugins(kv)

	// 3. 预热站点主题缓存
	theme := model.ParseSiteTheme(kv)
	setSiteTheme(app, ctx, theme)
}
