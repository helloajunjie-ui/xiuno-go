// xiuno-go v2.1.0-beta 尼克修改版
package model

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/jmoiron/sqlx"

	"xiuno/core"
)

// KV 对应 bbs_kv 表
type KV struct {
	K      string `db:"k" json:"k"`
	V      string `db:"v" json:"v"`
	Expiry uint32 `db:"expiry" json:"expiry"`
}

// LoadAllKV 全量加载 bbs_kv 到 map，供启动时缓存预热
func LoadAllKV(ctx context.Context, db *sqlx.DB) (map[string]string, error) {
	rows, err := db.QueryContext(ctx, `SELECT k, v FROM bbs_kv`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	kv := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		kv[k] = v
	}
	return kv, rows.Err()
}

// SetKV 写入/更新单条 KV 配置
func SetKV(ctx context.Context, db *sqlx.DB, key, value string) error {
	_, err := db.ExecContext(ctx,
		`INSERT INTO bbs_kv (k, v, expiry) VALUES (?, ?, 0) ON DUPLICATE KEY UPDATE v = ?`,
		key, value, value)
	return err
}

// DeleteKV 删除单条 KV 配置
func DeleteKV(ctx context.Context, db *sqlx.DB, key string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM bbs_kv WHERE k = ?`, key)
	return err
}

// ==================== KV 缓存层 ====================

// cacheKeyPrefix KV 缓存 key 前缀
const cacheKeyPrefix = "kv:"

// KVCacheGet 从缓存读取 KV 值，缓存未命中则查 DB 并回填
// 返回 value 和是否存在的标志
func KVCacheGet(ctx context.Context, cache core.Cache, db *sqlx.DB, key string) (string, bool, error) {
	cacheKey := cacheKeyPrefix + key

	// 1. 尝试从缓存读取
	data, ok := cache.Get(ctx, cacheKey)
	if ok {
		return string(data), true, nil
	}

	// 2. 缓存未命中，查 DB
	var v string
	err := db.GetContext(ctx, &v, `SELECT v FROM bbs_kv WHERE k = ?`, key)
	if err != nil {
		return "", false, nil // key 不存在
	}

	// 3. 回填缓存（TTL=0 表示永不过期，由写入/删除时主动失效）
	cache.Set(ctx, cacheKey, []byte(v), 0)
	return v, true, nil
}

// KVCacheSet 写入 KV 到 DB 并更新缓存
func KVCacheSet(ctx context.Context, cache core.Cache, db *sqlx.DB, key, value string) error {
	// 1. 写入 DB
	if err := SetKV(ctx, db, key, value); err != nil {
		return err
	}
	// 2. 更新缓存
	cache.Set(ctx, cacheKeyPrefix+key, []byte(value), 0)
	return nil
}

// KVCacheDelete 删除 KV 并清除缓存
func KVCacheDelete(ctx context.Context, cache core.Cache, db *sqlx.DB, key string) error {
	// 1. 删除 DB
	if err := DeleteKV(ctx, db, key); err != nil {
		return err
	}
	// 2. 清除缓存
	cache.Del(ctx, cacheKeyPrefix+key)
	return nil
}

// KVCacheWarmUp 启动时预热：全量加载 bbs_kv 到缓存
func KVCacheWarmUp(ctx context.Context, cache core.Cache, db *sqlx.DB) error {
	kvMap, err := LoadAllKV(ctx, db)
	if err != nil {
		return err
	}
	for k, v := range kvMap {
		cache.Set(ctx, cacheKeyPrefix+k, []byte(v), 0)
	}
	return nil
}

// SiteConf 站点配置结构体（从 bbs_kv 解析）
// 前端通过 GET /api/v1/config 获取，0 DB 压力
type SiteConf struct {
	SiteName     string `json:"site_name"`
	SiteBrief    string `json:"site_brief"`
	SiteURL      string `json:"site_url"`
	PageSize     int    `json:"page_size"`      // 每页主题数
	PostPageSize int    `json:"post_page_size"` // 每页回帖数
	CloseReason  string `json:"close_reason"`
}

// DefaultSiteConf 返回默认站点配置
func DefaultSiteConf() *SiteConf {
	return &SiteConf{
		SiteName:     "Xiuno Go",
		SiteBrief:    "Powered by Xiuno Go",
		PageSize:     20,
		PostPageSize: 50,
	}
}

// ParseSiteConf 从 kv map 解析站点配置
func ParseSiteConf(kv map[string]string) *SiteConf {
	conf := DefaultSiteConf()
	if v, ok := kv["site_name"]; ok && v != "" {
		conf.SiteName = v
	}
	if v, ok := kv["site_brief"]; ok && v != "" {
		conf.SiteBrief = v
	}
	if v, ok := kv["site_url"]; ok && v != "" {
		conf.SiteURL = v
	}
	if v, ok := kv["page_size"]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			conf.PageSize = n
		}
	}
	if v, ok := kv["post_page_size"]; ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			conf.PostPageSize = n
		}
	}
	if v, ok := kv["close_reason"]; ok {
		conf.CloseReason = v
	}
	return conf
}

// SiteConfToMap 将 SiteConf 序列化为 kv map（用于持久化）
func SiteConfToMap(conf *SiteConf) map[string]string {
	return map[string]string{
		"site_name":      conf.SiteName,
		"site_brief":     conf.SiteBrief,
		"site_url":       conf.SiteURL,
		"page_size":      strconv.Itoa(conf.PageSize),
		"post_page_size": strconv.Itoa(conf.PostPageSize),
		"close_reason":   conf.CloseReason,
	}
}

// ==================== SiteTheme（主题即数据） ====================

// SiteTheme 站点主题配置（存储在 bbs_kv 表，key = "site_theme"）
// 前端通过 GET /api/v1/theme 获取，0 DB 压力
type SiteTheme struct {
	PrimaryColor string `json:"primary_color"` // 主色，如 "#4f46e5"
	BgColor      string `json:"bg_color"`      // 背景色，如 "#f9fafb"
	CardRadius   string `json:"card_radius"`   // 卡片圆角，如 "0.75rem"
	ListLayout   string `json:"list_layout"`   // 列表布局：classic | waterfall
	ThemeMode    string `json:"theme_mode"`    // 主题模式：light | dark
	CustomCSS    string `json:"custom_css"`    // 用户自定义 CSS 片段
}

// DefaultSiteTheme 返回默认主题配置
func DefaultSiteTheme() *SiteTheme {
	return &SiteTheme{
		PrimaryColor: "#4f46e5",
		BgColor:      "#f9fafb",
		CardRadius:   "0.75rem",
		ListLayout:   "classic",
		ThemeMode:    "light",
		CustomCSS:    "",
	}
}

// ParseSiteTheme 从 kv map 解析主题配置
// 如果 key "site_theme" 不存在或解析失败，返回默认配置
func ParseSiteTheme(kv map[string]string) *SiteTheme {
	raw, ok := kv["site_theme"]
	if !ok || raw == "" {
		return DefaultSiteTheme()
	}
	var theme SiteTheme
	if err := json.Unmarshal([]byte(raw), &theme); err != nil {
		return DefaultSiteTheme()
	}
	// 字段级兜底
	if theme.PrimaryColor == "" {
		theme.PrimaryColor = "#4f46e5"
	}
	if theme.BgColor == "" {
		theme.BgColor = "#f9fafb"
	}
	if theme.CardRadius == "" {
		theme.CardRadius = "0.75rem"
	}
	if theme.ListLayout == "" {
		theme.ListLayout = "classic"
	}
	if theme.ThemeMode == "" {
		theme.ThemeMode = "light"
	}
	return &theme
}
