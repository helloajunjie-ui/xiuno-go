package core

import (
	"context"
	"net/http"
	"strings"
)

type contextKey string

const (
	ContextKeyClaims contextKey = "claims"
)

// CORSMiddleware 跨域中间件
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Credentials", "true")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthMiddleware JWT 认证中间件（未登录返回 401）
func AuthMiddleware(app *AppCtx) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				JSONError(w, http.StatusUnauthorized, "请先登录")
				return
			}

			claims, err := ParseJWT(token, app.Conf.JWT.Secret)
			if err != nil {
				JSONError(w, http.StatusUnauthorized, "登录已过期，请重新登录")
				return
			}

			ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OptionalAuthMiddleware 可选认证中间件（未登录也能访问）
// 无论是否携带 Token，都会往 Context 注入 Claims
// 未登录时注入 UID=0, GID=0 的默认 Claims，代表游客
// 这样 GetClaims 在公开路由上永远不会返回 nil
func OptionalAuthMiddleware(app *AppCtx) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token != "" {
				claims, err := ParseJWT(token, app.Conf.JWT.Secret)
				if err == nil {
					ctx := context.WithValue(r.Context(), ContextKeyClaims, claims)
					next.ServeHTTP(w, r.WithContext(ctx))
					return
				}
			}
			// 未登录：注入游客 Claims（UID=0, GID=0）
			guestClaims := &JWTClaims{UID: 0, GID: 0}
			ctx := context.WithValue(r.Context(), ContextKeyClaims, guestClaims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims 从 context 中获取 JWT claims
func GetClaims(ctx context.Context) *JWTClaims {
	claims, ok := ctx.Value(ContextKeyClaims).(*JWTClaims)
	if !ok {
		return nil
	}
	return claims
}

// AdminMiddleware 管理员权限中间件（GID=1 超管专享）
// 必须在 AuthMiddleware 之后使用，因为依赖 Context 中的 Claims
func AdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaims(r.Context())
			if claims == nil || claims.GID != 1 {
				JSONError(w, http.StatusForbidden, "皇家禁地，闲人止步")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// extractToken 从请求中提取 token
// 优先从 Authorization header 获取，回退到 Cookie
func extractToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	cookie, err := r.Cookie("xn_jwt")
	if err == nil {
		return cookie.Value
	}
	return ""
}
