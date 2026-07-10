package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"time"

	"xiuno/core"
	"xiuno/model"
)

// SSOConfig SSO 第三方登录配置（从 bbs_kv 读取）
type SSOConfig struct {
	QQAppID     string `json:"qq_appid"`
	QQAppKey    string `json:"qq_appkey"`
	WxAppID     string `json:"wx_appid"`
	WxAppSecret string `json:"wx_appsecret"`
}

// SSOConfigHandler 获取 SSO 配置（仅返回已启用的平台列表，不返回密钥）
// GET /api/v1/sso/config
func SSOConfigHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := getSSOConfig(app)
		platforms := make([]map[string]interface{}, 0)
		if cfg.QQAppID != "" {
			platforms = append(platforms, map[string]interface{}{
				"platform": "qq",
				"name":     "QQ 登录",
			})
		}
		if cfg.WxAppID != "" {
			platforms = append(platforms, map[string]interface{}{
				"platform": "wechat",
				"name":     "微信登录",
			})
		}
		core.JSONSuccess(w, map[string]interface{}{
			"platforms": platforms,
		})
	}
}

// SSOQQLoginHandler QQ 登录入口
// GET /api/v1/sso/qq/login -> 302 跳转到 QQ OAuth2 授权页
func SSOQQLoginHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := getSSOConfig(app)
		if cfg.QQAppID == "" {
			core.JSONError(w, http.StatusBadRequest, "QQ 登录未配置")
			return
		}

		redirectURI := fmt.Sprintf("%s/api/v1/sso/qq/callback", getSiteURL(r))
		state := generateState()

		// 保存 state 到 session 用于 CSRF 校验
		saveOAuthState(app, state)

		loginURL := fmt.Sprintf(
			"https://graph.qq.com/oauth2.0/authorize?response_type=code&client_id=%s&redirect_uri=%s&state=%s&scope=get_user_info",
			cfg.QQAppID, url.QueryEscape(redirectURI), state,
		)
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

// SSOQQCallbackHandler QQ OAuth2 回调
// GET /api/v1/sso/qq/callback?code=xxx&state=xxx
func SSOQQCallbackHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" || state == "" {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		// 验证 state（CSRF 防护）
		if !verifyOAuthState(app, state) {
			core.JSONError(w, http.StatusBadRequest, "state 校验失败")
			return
		}

		cfg := getSSOConfig(app)

		// 1. 用 code 换取 access_token
		token, err := qqGetToken(cfg.QQAppID, cfg.QQAppKey, code, fmt.Sprintf("%s/api/v1/sso/qq/callback", getSiteURL(r)))
		if err != nil {
			log.Printf("[SSO] QQ token 换取失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}

		// 2. 用 token 获取 openid
		openID, err := qqGetOpenID(token)
		if err != nil {
			log.Printf("[SSO] QQ openid 获取失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}

		// 3. 获取用户信息
		profile, err := qqGetUserInfo(token, cfg.QQAppID, openID)
		if err != nil {
			log.Printf("[SSO] QQ 用户信息获取失败: %v", err)
			// openid 已获取，可以继续
			profile = &model.SSOProfile{OpenID: openID, Nickname: "QQ用户"}
		}

		// 4. SSO 登录/自动注册
		user, isNew, err := model.SSOLoginOrRegister(r.Context(), app.DB, model.PlatQQ, profile)
		if err != nil {
			log.Printf("[SSO] SSO 登录失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}
		if user == nil {
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}

		// 5. 生成 JWT token
		tokenStr, err := core.SignJWT(user.UID, user.GID, app.Conf.JWT.Secret, app.Conf.JWT.ExpireHour)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "token 生成失败")
			return
		}

		// 6. 重定向到前端，附带 token
		redirectURL := fmt.Sprintf("%s/sso/callback?token=%s&is_new=%v",
			getSiteURL(r), tokenStr, isNew)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// SSOWechatLoginHandler 微信登录入口
// GET /api/v1/sso/wechat/login -> 302 跳转到微信 OAuth2 授权页
func SSOWechatLoginHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := getSSOConfig(app)
		if cfg.WxAppID == "" {
			core.JSONError(w, http.StatusBadRequest, "微信登录未配置")
			return
		}

		redirectURI := fmt.Sprintf("%s/api/v1/sso/wechat/callback", getSiteURL(r))
		state := generateState()
		saveOAuthState(app, state)

		loginURL := fmt.Sprintf(
			"https://open.weixin.qq.com/connect/oauth2/authorize?appid=%s&redirect_uri=%s&response_type=code&scope=snsapi_userinfo&state=%s#wechat_redirect",
			cfg.WxAppID, url.QueryEscape(redirectURI), state,
		)
		http.Redirect(w, r, loginURL, http.StatusFound)
	}
}

// SSOWechatCallbackHandler 微信 OAuth2 回调
// GET /api/v1/sso/wechat/callback?code=xxx&state=xxx
func SSOWechatCallbackHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		if code == "" || state == "" {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		if !verifyOAuthState(app, state) {
			core.JSONError(w, http.StatusBadRequest, "state 校验失败")
			return
		}

		cfg := getSSOConfig(app)

		// 1. 用 code 换取 access_token + openid
		token, openID, err := wechatGetToken(cfg.WxAppID, cfg.WxAppSecret, code)
		if err != nil {
			log.Printf("[SSO] 微信 token 换取失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}

		// 2. 获取用户信息
		profile, err := wechatGetUserInfo(token, openID)
		if err != nil {
			log.Printf("[SSO] 微信用户信息获取失败: %v", err)
			profile = &model.SSOProfile{OpenID: openID, Nickname: "微信用户"}
		}

		// 3. SSO 登录/自动注册
		user, isNew, err := model.SSOLoginOrRegister(r.Context(), app.DB, model.PlatWechat, profile)
		if err != nil {
			log.Printf("[SSO] SSO 登录失败: %v", err)
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}
		if user == nil {
			core.JSONError(w, http.StatusInternalServerError, "登录失败")
			return
		}

		tokenStr, err := core.SignJWT(user.UID, user.GID, app.Conf.JWT.Secret, app.Conf.JWT.ExpireHour)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "token 生成失败")
			return
		}

		redirectURL := fmt.Sprintf("%s/sso/callback?token=%s&is_new=%v",
			getSiteURL(r), tokenStr, isNew)
		http.Redirect(w, r, redirectURL, http.StatusFound)
	}
}

// SSOBindHandler 绑定第三方账号到当前登录用户
// POST /api/v1/sso/bind {platform, code}
func SSOBindHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}
		user, err := model.GetUserByUID(r.Context(), app.DB, claims.UID)
		if err != nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}
		if user == nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}

		var req struct {
			Platform string `json:"platform"`
			Code     string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		var platID model.PlatID
		switch req.Platform {
		case "qq":
			platID = model.PlatQQ
		case "wechat":
			platID = model.PlatWechat
		default:
			core.JSONError(w, http.StatusBadRequest, "不支持的平台")
			return
		}

		// 检查是否已绑定
		binds, err := model.UserOpenFindByUID(r.Context(), app.DB, user.UID)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "查询失败")
			return
		}
		for _, b := range binds {
			if b.PlatID == platID {
				core.JSONError(w, http.StatusBadRequest, "该平台已绑定")
				return
			}
		}

		// 这里简化处理：前端传入 openid 直接绑定
		// 完整流程需要走 OAuth2，此处仅提供绑定接口骨架
		if err := model.UserOpenCreate(r.Context(), app.DB, user.UID, platID, req.Code); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "绑定失败")
			return
		}

		core.JSONSuccess(w, map[string]string{"message": "绑定成功"})
	}
}

// SSOUnbindHandler 解绑第三方账号
// POST /api/v1/sso/unbind {platform}
func SSOUnbindHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := core.GetClaims(r.Context())
		if claims == nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}
		user, err := model.GetUserByUID(r.Context(), app.DB, claims.UID)
		if err != nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}
		if user == nil {
			core.JSONError(w, http.StatusUnauthorized, "未登录")
			return
		}

		var req struct {
			Platform string `json:"platform"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, http.StatusBadRequest, "参数错误")
			return
		}

		var platID model.PlatID
		switch req.Platform {
		case "qq":
			platID = model.PlatQQ
		case "wechat":
			platID = model.PlatWechat
		default:
			core.JSONError(w, http.StatusBadRequest, "不支持的平台")
			return
		}

		if err := model.UserOpenDelete(r.Context(), app.DB, user.UID, platID); err != nil {
			core.JSONError(w, http.StatusInternalServerError, "解绑失败")
			return
		}

		core.JSONSuccess(w, map[string]string{"message": "解绑成功"})
	}
}

// ====== 内部辅助函数 ======

func getSSOConfig(app *core.AppCtx) *SSOConfig {
	cfg := &SSOConfig{}
	val, ok := app.Cache.Get(nil, "sso_config")
	if ok {
		json.Unmarshal(val, cfg)
		return cfg
	}

	// 从 bbs_kv 读取
	kvStr, _, _ := model.KVCacheGet(nil, app.Cache, app.DB, "sso_config")
	if kvStr != "" {
		json.Unmarshal([]byte(kvStr), cfg)
	}
	// 缓存到内存
	if data, err := json.Marshal(cfg); err == nil {
		app.Cache.Set(nil, "sso_config", data, 0)
	}
	return cfg
}

func getSiteURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}

func generateState() string {
	return fmt.Sprintf("sso_%d", time.Now().UnixNano())
}

func saveOAuthState(app *core.AppCtx, state string) {
	app.Cache.Set(nil, "oauth_state:"+state, []byte("1"), 300) // 5分钟过期
}

func verifyOAuthState(app *core.AppCtx, state string) bool {
	_, ok := app.Cache.Get(nil, "oauth_state:"+state)
	if ok {
		app.Cache.Del(nil, "oauth_state:"+state)
	}
	return ok
}

// ====== QQ API 调用 ======

func qqGetToken(appID, appKey, code, redirectURI string) (string, error) {
	urlStr := fmt.Sprintf(
		"https://graph.qq.com/oauth2.0/token?grant_type=authorization_code&client_id=%s&client_secret=%s&code=%s&redirect_uri=%s",
		appID, appKey, code, url.QueryEscape(redirectURI),
	)
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// QQ 返回格式: access_token=xxx&expires_in=xxx
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return "", err
	}

	token := values.Get("access_token")
	if token == "" {
		return "", fmt.Errorf("QQ token 返回为空: %s", string(body))
	}
	return token, nil
}

func qqGetOpenID(token string) (string, error) {
	urlStr := fmt.Sprintf("https://graph.qq.com/oauth2.0/me?access_token=%s", token)
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	// QQ 返回格式: callback( {"client_id":"xxx","openid":"xxx"} )
	s := string(body)
	// 提取 JSON 部分
	var data struct {
		OpenID string `json:"openid"`
		Error  int    `json:"error"`
	}
	if err := json.Unmarshal([]byte(s), &data); err != nil {
		// 尝试提取 callback 中的 JSON
		if len(s) > 10 {
			start := 0
			for i, c := range s {
				if c == '{' {
					start = i
					break
				}
			}
			if end := len(s) - 1; end > start {
				json.Unmarshal([]byte(s[start:end]), &data)
			}
		}
	}

	if data.OpenID == "" {
		return "", fmt.Errorf("QQ openid 获取失败: %s", s)
	}
	return data.OpenID, nil
}

func qqGetUserInfo(token, appID, openID string) (*model.SSOProfile, error) {
	urlStr := fmt.Sprintf(
		"https://graph.qq.com/user/get_user_info?access_token=%s&oauth_consumer_key=%s&openid=%s&format=json",
		token, appID, openID,
	)
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Nickname  string `json:"nickname"`
		Gender    string `json:"gender"`
		Figureurl string `json:"figureurl_qq_2"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	return &model.SSOProfile{
		OpenID:    openID,
		Nickname:  data.Nickname,
		AvatarURL: data.Figureurl,
		Gender:    data.Gender,
	}, nil
}

// ====== 微信 API 调用 ======

func wechatGetToken(appID, secret, code string) (token, openID string, err error) {
	urlStr := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/oauth2/access_token?appid=%s&secret=%s&code=%s&grant_type=authorization_code",
		appID, secret, code,
	)
	resp, err := http.Get(urlStr)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	var data struct {
		AccessToken string `json:"access_token"`
		OpenID      string `json:"openid"`
		ErrCode     int    `json:"errcode"`
		ErrMsg      string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", err
	}

	if data.ErrCode != 0 {
		return "", "", fmt.Errorf("微信 API 错误: %d %s", data.ErrCode, data.ErrMsg)
	}
	return data.AccessToken, data.OpenID, nil
}

func wechatGetUserInfo(token, openID string) (*model.SSOProfile, error) {
	urlStr := fmt.Sprintf(
		"https://api.weixin.qq.com/sns/userinfo?access_token=%s&openid=%s&lang=zh_CN",
		token, openID,
	)
	resp, err := http.Get(urlStr)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var data struct {
		Nickname   string `json:"nickname"`
		Sex        int    `json:"sex"`
		Headimgurl string `json:"headimgurl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}

	gender := "未知"
	switch data.Sex {
	case 1:
		gender = "男"
	case 2:
		gender = "女"
	}

	return &model.SSOProfile{
		OpenID:    openID,
		Nickname:  data.Nickname,
		AvatarURL: data.Headimgurl,
		Gender:    gender,
	}, nil
}
