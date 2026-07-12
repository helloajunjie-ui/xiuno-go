// xiuno-go v2.1.0-beta 尼克修改版
package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"xiuno/core"
	"xiuno/handler"
	"xiuno/model"
	"xiuno/plugin/spam_blocker"
	"xiuno/ui"

	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
)

func main() {
	// 1. 加载配置
	cfg, err := core.LoadConfig("xiuno.json")
	if err != nil {
		log.Fatalf("[FATAL] 配置加载失败: %v", err)
	}

	// 2. 初始化应用上下文（DB、Cache）
	app := core.NewAppCtx(cfg)
	defer app.Close()

	// 2.3 自动建表 + 初始数据填充（幂等设计）
	// 放在 NewAppCtx 之后、使用 DB 之前，避免 core→model 循环依赖
	if err := model.AutoMigrate(app.DB); err != nil {
		log.Fatalf("[FATAL] 自动建表失败: %v", err)
	}

	// 2.4 确保默认头像文件存在（生成 128x128 灰色占位 PNG）
	if err := model.EnsureDefaultAvatar("upload"); err != nil {
		log.Printf("[WARN] 默认头像生成失败: %v", err)
	}

	// 2.5 预热站点配置缓存（从 bbs_kv 全量加载到内存）
	handler.InitSiteConf(app)

	// 2.6 启动计划任务（后台协程）
	app.Cron = model.NewCron(app.DB, app.Cache, "upload")

	// 2.7 注册插件（编译期注册，import 即生效）
	app.Hook.Register(app, &spam_blocker.SpamPlugin{})

	// 3. 路由注册
	r := chi.NewRouter()

	// 全局中间件
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(chimw.Logger)
	r.Use(chimw.Recoverer)
	r.Use(core.CORSMiddleware)

	// 健康检查
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		core.JSONSuccess(w, map[string]string{"status": "ok"})
	})

	// 浏览器兼容提示页（无需认证，纯静态 HTML）
	r.Get("/browser", handler.BrowserPageHandler())
	r.Get("/browser-download/{type}", handler.BrowserDownloadHandler())

	// API v1 路由组
	r.Route("/api/v1", func(r chi.Router) {
		// 公开接口（无需认证）
		r.Group(func(r chi.Router) {
			r.Use(core.OptionalAuthMiddleware(app))
			r.Get("/forum", handler.ForumListHandler(app))
			r.Get("/forum/{fid}", handler.ForumReadHandler(app))
			r.Get("/thread", handler.ThreadListHandler(app))
			r.Get("/thread/{tid}", handler.ThreadReadHandler(app))
			r.Get("/thread/{tid}/post", handler.ReplyListHandler(app))
			r.With(core.RateLimitMiddleware(app, "login", 10, 60)).
				Post("/user/login", handler.UserLoginHandler(app))
			r.With(core.RateLimitMiddleware(app, "register", 3, 3600)).
				Post("/user/register", handler.UserRegisterHandler(app))
			r.Get("/user/{uid}", handler.UserProfileHandler(app))
			r.Get("/user/{uid}/thread", handler.UserThreadListHandler(app))
			r.Get("/user/{uid}/post", handler.UserPostListHandler(app))
			r.With(core.RateLimitMiddleware(app, "send_code", 1, 60)).
				Post("/user/send-code", handler.UserSendCodeHandler(app))
			r.With(core.RateLimitMiddleware(app, "resetpw", 1, 60)).
				Post("/user/reset-password", handler.UserResetPasswordHandler(app))
			r.Get("/config", handler.ConfigHandler(app))
			r.Get("/theme", handler.ThemeHandler(app))
			r.Get("/runtime", handler.RuntimeHandler(app))
			r.Get("/attach/{aid}", handler.AttachDownloadHandler(app))
			r.Get("/tag", handler.TagListHandler(app))
			r.Get("/tag/{tagid}", handler.TagReadHandler(app))
			r.Get("/tag/{tagid}/thread", handler.TagThreadListHandler(app))
			// SSO 第三方登录配置（公开）
			r.Get("/sso/config", handler.SSOConfigHandler(app))
		})

		// 需要认证的接口
		r.Group(func(r chi.Router) {
			r.Use(core.AuthMiddleware(app))
			r.With(core.RateLimitMiddleware(app, "thread", 2, 60)).
				Post("/thread", handler.ThreadCreateHandler(app))
			r.Put("/thread/{tid}", handler.ThreadUpdateHandler(app))
			r.Delete("/thread/{tid}", handler.ThreadDeleteHandler(app))
			r.With(core.RateLimitMiddleware(app, "post", 1, 10)).
				Post("/thread/{tid}/post", handler.ReplyCreateHandler(app))
			r.Post("/thread/{tid}/moderate", handler.ThreadModerateHandler(app))
			r.Post("/thread/{tid}/move", handler.ThreadMoveHandler(app))
			r.Put("/post/{pid}", handler.PostUpdateHandler(app))
			r.Delete("/post/{pid}", handler.PostDeleteHandler(app))
			r.Post("/attach", handler.UploadHandler(app))
			r.Post("/user/avatar", handler.UserAvatarUploadHandler(app))
			r.Get("/user/logout", handler.UserLogoutHandler(app))
			r.Put("/user/password", handler.UserPasswordUpdateHandler(app))
			r.Delete("/attach/{aid}", handler.AttachDeleteHandler(app))
			r.Delete("/user/{uid}/delete", handler.UserDeleteByModHandler(app))
			r.Get("/my/profile", handler.MyProfileHandler(app))
			r.Get("/my/thread", handler.MyThreadListHandler(app))
			// SSO 同步登录（用已有 token 换取本站 JWT）
			r.Get("/user/synlogin", handler.UserSynloginHandler(app))
			// SSO 第三方登录绑定/解绑（需登录）
			r.Post("/sso/bind", handler.SSOBindHandler(app))
			r.Post("/sso/unbind", handler.SSOUnbindHandler(app))
		})

		// 管理员接口（Auth + Admin 双层防护）
		r.Route("/admin", func(r chi.Router) {
			r.Use(core.AuthMiddleware(app))
			r.Use(core.AdminMiddleware())
			r.Put("/config", handler.AdminConfigUpdateHandler(app))
			r.Put("/theme", handler.AdminThemeUpdateHandler(app))
			r.Get("/plugin", handler.AdminPluginListHandler(app))
			r.Put("/plugin", handler.AdminPluginToggleHandler(app))
			r.Get("/forum", handler.AdminForumListHandler(app))
			r.Post("/forum", handler.AdminForumCreateHandler(app))
			r.Put("/forum/{fid}", handler.AdminForumUpdateHandler(app))
			r.Delete("/forum/{fid}", handler.AdminForumDeleteHandler(app))
			r.Get("/forum/{fid}/access", handler.AdminForumAccessListHandler(app))
			r.Put("/forum/{fid}/access", handler.AdminForumAccessUpdateHandler(app))
			r.Get("/user", handler.AdminUserListHandler(app))
			r.Put("/user/{uid}/group", handler.AdminUserGroupHandler(app))
			r.Get("/group", handler.AdminGroupListHandler(app))
			r.Get("/group/{gid}", handler.AdminGroupReadHandler(app))
			r.Post("/group", handler.AdminGroupCreateHandler(app))
			r.Put("/group/{gid}", handler.AdminGroupUpdateHandler(app))
			r.Delete("/group/{gid}", handler.AdminGroupDeleteHandler(app))
			r.Get("/modlog", handler.AdminModLogListHandler(app))
			// 后台标签管理
			r.Get("/tag", handler.AdminTagListHandler(app))
			r.Post("/tag", handler.AdminTagCreateHandler(app))
			r.Put("/tag/{tagid}", handler.AdminTagUpdateHandler(app))
			r.Delete("/tag/{tagid}", handler.AdminTagDeleteHandler(app))
			// 后台主题管理（MySQL 模拟队列）
			r.Post("/thread/scan", handler.AdminThreadScanHandler(app))
			r.Post("/thread/operation", handler.AdminThreadOperationHandler(app))
			r.Get("/thread/found", handler.AdminThreadFoundHandler(app))
			r.Delete("/thread/{tid}", handler.AdminThreadDeleteHandler(app))
		})
	})

	// SSO OAuth2 回调路由（无需认证，外部 OAuth 回调）
	// 注意：必须在 /api/v1 路由组外部注册，避免路径前缀重复
	r.Get("/api/v1/sso/qq/login", handler.SSOQQLoginHandler(app))
	r.Get("/api/v1/sso/qq/callback", handler.SSOQQCallbackHandler(app))
	r.Get("/api/v1/sso/wechat/login", handler.SSOWechatLoginHandler(app))
	r.Get("/api/v1/sso/wechat/callback", handler.SSOWechatCallbackHandler(app))

	// 4. 上传文件静态服务（本地磁盘 upload/ 目录）
	//    必须在 SPA fallback 之前注册，否则 /* 会吞掉所有 /upload/* 请求
	r.Get("/upload/*", http.StripPrefix("/upload/",
		http.FileServer(http.Dir("upload"))).ServeHTTP)

	// 5. SPA 静态文件服务（go:embed 单文件部署）
	staticFS := ui.GetStaticFS()
	fileServer := http.FileServer(staticFS)

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// 只处理 GET 请求
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		// 尝试打开请求的文件
		f, err := staticFS.Open(path)
		if err == nil {
			f.Close()
			// 带 hash 指纹的静态资源（/assets/*）设置强缓存
			if strings.HasPrefix(path, "assets/") {
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			}
			fileServer.ServeHTTP(w, r)
			return
		}

		// SPA fallback: 未匹配到静态文件则返回 index.html
		indexFile, err := staticFS.Open("index.html")
		if err != nil {
			http.Error(w, "Not found", http.StatusNotFound)
			return
		}
		defer indexFile.Close()

		stat, err := indexFile.Stat()
		if err != nil {
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		// index.html 不缓存，确保 SPA 更新后浏览器获取最新版本
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeContent(w, r, "index.html", stat.ModTime(), indexFile.(io.ReadSeeker))
	})

	// 6. 启动 HTTP 服务
	srv := &http.Server{
		Addr:         cfg.Server.Addr,
		Handler:      r,
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
	}

	// 后台协程启动服务
	go func() {
		log.Printf("[INFO] Xiuno-Go 启动于 %s 🚀", cfg.Server.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[FATAL] 服务异常: %v", err)
		}
	}()

	// 7. 优雅停机 (Graceful Shutdown)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("[INFO] 捕获退出信号，准备优雅停机...")

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		time.Duration(cfg.Server.ShutdownTimeout)*time.Second,
	)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("[FATAL] 停机异常: %v", err)
	}

	log.Println("[INFO] Xiuno-Go 已安全退出。")
}
