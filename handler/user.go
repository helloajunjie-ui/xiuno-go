// xiuno-go v2.1.0-beta 尼克修改版
package handler

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"xiuno/core"
	"xiuno/model"

	"github.com/jmoiron/sqlx"
)

// LoginReq 登录请求体
type LoginReq struct {
	Account  string `json:"account"` // 用户名或邮箱，前端不区分
	Password string `json:"password"`
}

// RegisterReq 注册请求体
type RegisterReq struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserLoginHandler 登录端点
func UserLoginHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req LoginReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "请求格式错误")
			return
		}

		req.Account = strings.TrimSpace(req.Account)
		if req.Account == "" || req.Password == "" {
			core.JSONError(w, 400, "账号和密码不能为空")
			return
		}

		// 1. 查询用户（支持用户名或邮箱）
		user, err := model.GetUserByAccount(r.Context(), app.DB, req.Account)
		if err != nil {
			if err == sql.ErrNoRows {
				core.JSONError(w, 401, "账号或密码错误")
				return
			}
			core.JSONErrorLog(w, 500, "服务器内部错误", err)
			return
		}

		// 2. 校验密码
		ok, needUpgrade := user.VerifyPassword(r.Context(), app.DB, req.Password)
		if !ok {
			core.JSONError(w, 401, "账号或密码错误")
			return
		}

		// 3. 静默升级密码（旧 MD5 → bcrypt）
		if needUpgrade {
			_ = user.UpgradePassword(r.Context(), app.DB, req.Password)
		}

		// 4. 更新登录信息
		_, _ = app.DB.ExecContext(r.Context(),
			`UPDATE bbs_user SET logins = logins + 1, login_date = UNIX_TIMESTAMP() WHERE uid = ?`,
			user.UID)

		// 5. 签发 JWT
		token, err := core.SignJWT(user.UID, user.GID, app.Conf.JWT.Secret, app.Conf.JWT.ExpireHour)
		if err != nil {
			core.JSONError(w, 500, "身份签发失败")
			return
		}

		// 6. 设置 HttpOnly Cookie（防御 XSS）+ SameSite=Lax（防御 CSRF）
		http.SetCookie(w, &http.Cookie{
			Name:     "xn_jwt",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 7,
		})

		// 7. 返回用户信息（密码等敏感字段已被 json:"-" 屏蔽）
		core.JSONSuccess(w, user)
	}
}

// UserRegisterHandler 注册端点
func UserRegisterHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "请求格式错误")
			return
		}

		req.Username = strings.TrimSpace(req.Username)
		req.Email = strings.TrimSpace(req.Email)

		// 服务端输入校验（对应 PHP check.func.php）
		if !model.IsUsername(req.Username) {
			core.JSONError(w, 400, "用户名需为 2-15 个字符，仅支持字母、数字、中文、下划线、横线")
			return
		}
		if !model.IsPassword(req.Password) {
			core.JSONError(w, 400, "密码长度需在 6-256 个字符之间")
			return
		}
		if !model.IsEmail(req.Email) {
			core.JSONError(w, 400, "邮箱格式不正确")
			return
		}

		// 检查冲突
		msg, err := model.CheckUserExists(r.Context(), app.DB, req.Username, req.Email)
		if err != nil {
			core.JSONError(w, 500, "服务器内部错误")
			return
		}
		if msg != "" {
			core.JSONError(w, 400, msg)
			return
		}

		// 事务创建用户
		var user *model.User
		err = app.Tx(func(tx *sqlx.Tx) error {
			var txErr error
			user, txErr = model.CreateUser(r.Context(), app.DB, tx, req.Username, req.Email, req.Password)
			return txErr
		})
		if err != nil {
			core.JSONError(w, 400, "注册失败，该邮箱或用户名可能已被注册")
			return
		}

		// 注册成功后直接签发 Token（无需重新登录）
		token, err := core.SignJWT(user.UID, user.GID, app.Conf.JWT.Secret, app.Conf.JWT.ExpireHour)
		if err != nil {
			core.JSONError(w, 500, "身份签发失败")
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     "xn_jwt",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 7,
		})

		core.JSONSuccess(w, user)
	}
}

// UserLogoutHandler GET /api/v1/user/logout
// 退出登录，后端签发失效 Cookie（HttpOnly Cookie 前端无法删除）
func UserLogoutHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:     "xn_jwt",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1, // -1 代表立刻销毁
		})
		core.JSONSuccess(w, nil)
	}
}

// PasswordUpdateReq 修改密码请求体
type PasswordUpdateReq struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

// UserPasswordUpdateHandler PUT /api/v1/user/password
// 修改密码，成功后强制下线（签发失效 Cookie）
func UserPasswordUpdateHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req PasswordUpdateReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "参数格式错误")
			return
		}

		if len(req.NewPassword) < 6 {
			core.JSONError(w, 400, "新密码不能少于6位")
			return
		}

		claims := core.GetClaims(r.Context())

		// 1. 查出当前用户
		user, err := model.GetUserByID(r.Context(), app.DB, claims.UID)
		if err != nil || user == nil {
			core.JSONError(w, 404, "用户不存在")
			return
		}

		// 2. 验证旧密码（复用双兼容校验逻辑）
		passed, _ := user.VerifyPassword(r.Context(), app.DB, req.OldPassword)
		if !passed {
			core.JSONError(w, 400, "旧密码不正确")
			return
		}

		// 3. 写入新密码（抹除 salt，全面拥抱 bcrypt）
		if err := model.UpdatePassword(r.Context(), app.DB, claims.UID, req.NewPassword); err != nil {
			core.JSONError(w, 500, "密码修改失败")
			return
		}

		// 失效用户缓存
		model.InvalidateUserCache(r.Context(), app.Cache, claims.UID)

		// 4. [安全高标] 密码修改成功，主动吊销当前登录状态
		http.SetCookie(w, &http.Cookie{
			Name:     "xn_jwt",
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		core.JSONSuccess(w, "密码修改成功，请重新登录")
	}
}

// --- 密码重置（P0 #3） ---

// SendCodeReq 发送验证码请求体
type SendCodeReq struct {
	Email string `json:"email"`
	Scene string `json:"scene"` // "register" 或 "resetpw"
}

// SendCodeResp 发送验证码响应
type SendCodeResp struct {
	ExpireSec int `json:"expire_sec"` // 验证码有效期（秒）
}

// UserSendCodeHandler POST /api/v1/user/send-code
// 发送邮件验证码，验证码存入缓存，有效期 10 分钟
func UserSendCodeHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SendCodeReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "参数格式错误")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		if !model.IsEmail(req.Email) {
			core.JSONError(w, 400, "邮箱格式不正确")
			return
		}

		// 检查 SMTP 是否配置
		if len(app.Conf.SMTP) == 0 {
			core.JSONError(w, 500, "SMTP 未配置，无法发送邮件")
			return
		}

		// 场景校验
		switch req.Scene {
		case "register":
			// 注册场景：邮箱不能已存在
			existing, err := model.GetUserByEmail(r.Context(), app.DB, req.Email)
			if err == nil && existing != nil {
				core.JSONError(w, 400, "该邮箱已被注册")
				return
			}
		case "resetpw":
			// 重置密码场景：邮箱必须存在
			existing, err := model.GetUserByEmail(r.Context(), app.DB, req.Email)
			if err != nil || existing == nil {
				core.JSONError(w, 400, "该邮箱未注册")
				return
			}
		default:
			core.JSONError(w, 400, "无效的场景参数")
			return
		}

		// 生成 6 位随机验证码
		code, err := generateCode()
		if err != nil {
			core.JSONError(w, 500, "验证码生成失败")
			return
		}

		// 存入缓存，key = "verify_code:{scene}:{email}"，有效期 10 分钟
		cacheKey := fmt.Sprintf("verify_code:%s:%s", req.Scene, req.Email)
		ctx := r.Context()
		app.Cache.Set(ctx, cacheKey, []byte(code), 10*time.Minute)

		// 发送邮件
		subject := fmt.Sprintf("【%s】验证码：%s", app.Conf.Site.Name, code)
		body := fmt.Sprintf("您的验证码是：%s\n\n有效期 10 分钟，请勿泄露给他人。\n\n—— %s", code, app.Conf.Site.Name)

		mailer := model.NewMailer(app.Conf.SMTP)
		if err := mailer.Send(req.Email, subject, body); err != nil {
			core.JSONError(w, 500, fmt.Sprintf("邮件发送失败: %v", err))
			return
		}

		core.JSONSuccess(w, SendCodeResp{ExpireSec: 600})
	}
}

// ResetPasswordReq 重置密码请求体
type ResetPasswordReq struct {
	Email    string `json:"email"`
	Code     string `json:"code"`
	Password string `json:"password"`
}

// UserResetPasswordHandler POST /api/v1/user/reset-password
// 验证码校验 + 设置新密码（合并 PHP resetpw POST + resetpw_complete POST）
func UserResetPasswordHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req ResetPasswordReq
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			core.JSONError(w, 400, "参数格式错误")
			return
		}

		req.Email = strings.TrimSpace(req.Email)
		req.Code = strings.TrimSpace(req.Code)

		if !model.IsEmail(req.Email) {
			core.JSONError(w, 400, "邮箱格式不正确")
			return
		}
		if req.Code == "" {
			core.JSONError(w, 400, "验证码不能为空")
			return
		}
		if !model.IsPassword(req.Password) {
			core.JSONError(w, 400, "密码长度需在 6-256 个字符之间")
			return
		}

		// 从缓存读取验证码
		cacheKey := fmt.Sprintf("verify_code:resetpw:%s", req.Email)
		ctx := r.Context()
		stored, ok := app.Cache.Get(ctx, cacheKey)
		if !ok {
			core.JSONError(w, 400, "验证码已过期或未发送，请重新获取")
			return
		}

		if string(stored) != req.Code {
			core.JSONError(w, 400, "验证码不正确")
			return
		}

		// 验证码正确，删除缓存（一次性使用）
		app.Cache.Del(ctx, cacheKey)

		// 查找用户
		user, err := model.GetUserByEmail(ctx, app.DB, req.Email)
		if err != nil {
			core.JSONError(w, 400, "该邮箱未注册")
			return
		}

		// 更新密码（bcrypt，抹除 salt）
		if err := model.UpdatePassword(ctx, app.DB, user.UID, req.Password); err != nil {
			core.JSONError(w, 500, "密码重置失败")
			return
		}

		// 失效用户缓存
		model.InvalidateUserCache(ctx, app.Cache, user.UID)

		core.JSONSuccess(w, "密码重置成功，请使用新密码登录")
	}
}

// UserSynloginHandler SSO 同步登录
// GET /api/v1/user/synlogin?token=xxx
// 对应 PHP: user.php?action=synlogin
// 用于跨站 SSO 登录状态同步：用已有 token 换取本站 JWT
func UserSynloginHandler(app *core.AppCtx) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			core.JSONError(w, http.StatusBadRequest, "缺少 token 参数")
			return
		}

		// 解析 token
		claims, err := core.ParseJWT(token, app.Conf.JWT.Secret)
		if err != nil {
			core.JSONError(w, http.StatusUnauthorized, "token 无效或已过期")
			return
		}

		// 验证用户是否存在
		user, err := model.GetUserByUID(r.Context(), app.DB, claims.UID)
		if err != nil || user == nil {
			core.JSONError(w, http.StatusUnauthorized, "用户不存在")
			return
		}

		// 签发新 JWT
		newToken, err := core.SignJWT(user.UID, user.GID, app.Conf.JWT.Secret, app.Conf.JWT.ExpireHour)
		if err != nil {
			core.JSONError(w, http.StatusInternalServerError, "身份签发失败")
			return
		}

		// 设置 Cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "xn_jwt",
			Value:    newToken,
			Path:     "/",
			HttpOnly: true,
			Secure:   r.TLS != nil,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   86400 * 7,
		})

		core.JSONSuccess(w, map[string]interface{}{
			"token": newToken,
			"user":  user,
		})
	}
}

// generateCode 生成 6 位随机数字验证码
func generateCode() (string, error) {
	max := big.NewInt(1000000)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
