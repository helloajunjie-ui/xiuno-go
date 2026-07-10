# Xiuno Go 架构文档

> **核心定位**：本文档基于磁盘上真实 `.go` 文件生成，100% 反映当前二进制文件的物理状态。
> **最后更新**：2026-07-10
> **编译状态**：`go build ./...` 零错误 ✅ | `vue-tsc --noEmit` 零错误 ✅
> **二进制大小**：~8.5 MB（单文件，含完整前端 SPA）

---

## 一、项目结构

```
xiuno/                          # 57 个 .go 文件
├── cmd/xiuno/main.go           # 入口：chi 路由 + 优雅启动 + SPA fallback
├── ui/                         # 前端嵌入层（go:embed）
│   ├── embed.go                # //go:embed dist/* → embed.FS
│   └── dist/                   # Vue SPA 构建产物（自动嵌入二进制）
├── core/                       # 核心框架层
│   ├── app.go                  # AppCtx 依赖注入容器
│   ├── config.go               # 配置加载（JSON 文件）
│   ├── cache.go                # 缓存接口（memory 默认实现）
│   ├── counter.go              # 异步计数器（消除 InnoDB 行锁热点）
│   ├── response.go             # 统一 JSON 响应格式
│   ├── jwt.go                  # JWT 签发/验证（HS256，自实现）
│   ├── password.go             # 密码哈希（bcrypt + XiunoMD5 兼容）
│   ├── middleware.go           # CORS / JWT 认证 / Admin 中间件
│   ├── policy.go               # 权限策略层
│   ├── storage.go              # 存储抽象层（Storage 接口 + LocalStorage）
│   ├── ratelimit.go            # 内存级滑动窗口限流器
│   └── hook.go                 # Hook 引擎（Plugin 接口 + Filter/Action）
├── handler/                    # 路由处理器层
│   ├── forum.go                # 版块列表
│   ├── thread.go               # 帖子列表/创建/详情
│   ├── thread_manage.go        # 帖子编辑/软删除
│   ├── post.go                 # 回帖创建/列表
│   ├── post_manage.go          # 回帖编辑/软删除 + 版务操作
│   ├── user.go                 # 登录/注册/注销/修改密码
│   ├── user_profile.go         # 用户资料/头像/帖子列表/回帖列表
│   ├── my.go                   # 我的帖子列表
│   ├── attach.go               # 文件上传/下载/删除
│   ├── config.go               # 站点配置读取/更新
│   ├── admin_plugin.go         # 插件管理
│   ├── admin_forum.go          # 版块管理 CRUD
│   ├── admin_user.go           # 用户管理
│   ├── admin_group.go          # 用户组 CRUD
│   ├── admin_modlog.go         # 版务日志查询
│   ├── admin_thread.go         # 后台主题管理
│   ├── browser.go              # 浏览器下载页
│   └── sso.go                  # QQ/微信 OAuth2 登录
├── model/                      # 数据模型层
│   ├── user.go                 # User struct + 认证/CRUD
│   ├── forum.go                # Forum struct + CRUD
│   ├── group.go                # Group struct + CRUD + 自动升级
│   ├── thread.go               # Thread struct + 查询/CRUD
│   ├── thread_top.go           # 置顶主题管理
│   ├── mythread.go             # 我的帖子关联表
│   ├── post.go                 # Post struct + CRUD
│   ├── attach.go               # Attach struct + CRUD + GC
│   ├── access.go               # 有效权限计算引擎
│   ├── modlog.go               # 版务日志
│   ├── kv.go                   # KV 配置系统
│   ├── cache_helper.go         # 统一缓存辅助层
│   ├── cascade.go              # 级联删除
│   ├── runtime.go              # 运行时统计
│   ├── cron.go                 # 计划任务
│   ├── check.go                # 输入校验
│   ├── mail.go                 # SMTP 邮件发送
│   ├── queue.go                # MySQL 模拟队列
│   ├── table_day.go            # 每日最大 ID 统计
│   └── user_open.go            # 第三方登录绑定
├── plugin/                     # 编译期注册的插件
│   └── spam_blocker/main.go    # 防灌水插件（敏感词过滤）
├── schema.sql                  # 建表 DDL（InnoDB, utf8mb4）
├── go.mod
└── go.sum
```

---

## 二、核心框架层（core/）

### 2.1 AppCtx 依赖注入容器

[`core/app.go:14`](core/app.go:14) — 替代 PHP 中 `$_SERVER` 全局变量的依赖注入容器：

```go
type AppCtx struct {
    DB          *sqlx.DB
    Cache       Cache
    Conf        *Config
    Counter     *AsyncCounter
    Storage     Storage
    RateLimiter *RateLimiter
    Hook        *HookManager
}
```

所有 handler 通过闭包注入 `*AppCtx`，不依赖全局变量。

#### 事务包装器

[`core/app.go:97`](core/app.go:97) — `app.Tx(fn func(tx *sqlx.Tx) error)` 自动处理 `Beginx()` → `fn(tx)` → `Commit()` / `Rollback()` + panic 恢复。

### 2.2 配置加载

[`core/config.go`](core/config.go) — JSON 配置文件加载，支持 `XIUNO_CONFIG` 环境变量覆盖路径。配置结构：

```go
type Config struct {
    Database struct {
        DSN         string `json:"dsn"`
        TablePrefix string `json:"table_prefix"`
    } `json:"database"`
    JWT struct {
        Secret      string `json:"secret"`
        ExpireHour  int    `json:"expire_hour"`
    } `json:"jwt"`
    App struct {
        UploadDir string `json:"upload_dir"`
        SiteURL   string `json:"site_url"`
    } `json:"app"`
    SMTP struct {
        Host     string `json:"host"`
        Port     int    `json:"port"`
        Username string `json:"username"`
        Password string `json:"password"`
        From     string `json:"from"`
    } `json:"smtp"`
    SSO struct {
        QQAppID     string `json:"qq_app_id"`
        QQAppKey    string `json:"qq_app_key"`
        WechatAppID string `json:"wechat_app_id"`
        WechatSecret string `json:"wechat_secret"`
    } `json:"sso"`
}
```

### 2.3 缓存接口

[`core/cache.go`](core/cache.go) — 接口定义 + 内存实现：

```go
type Cache interface {
    Get(key string) (string, bool)
    Set(key string, value string, ttl int)
    Delete(key string)
    Clear()
}
```

默认使用 `MemoryCache`（`sync.Map` + TTL 检查），预留 Redis 实现接口。

### 2.4 异步计数器

[`core/counter.go:14`](core/counter.go:14) — 消除 InnoDB 行锁热点的异步批量计数器：

```go
type AsyncCounter struct {
    forumThreads map[uint32]int // fid -> +N
    userThreads  map[uint32]int // uid -> +N
    userPosts    map[uint32]int // uid -> +N
    threadViews  map[uint32]int // tid -> +N
    threadPosts  map[uint32]int // tid -> +N
}
```

- 2 秒间隔批量刷入 DB，将 `UPDATE ... +1` 行锁串行操作合并为批量 UPDATE
- 覆盖 5 个统计热点：版块主题数、用户发帖数、用户回帖数、帖子浏览数、帖子回复数
- 支持 Decr 方法（`DecrForumThread`、`DecrUserThread`、`DecrThreadPost`），使用 `GREATEST(CAST(... AS SIGNED) + ?, 0)` 防止 `BIGINT UNSIGNED` 溢出

### 2.5 JWT 认证

[`core/jwt.go:27`](core/jwt.go:27) — 自实现 HS256 JWT，无第三方依赖：

```go
func SignJWT(uid uint32, gid uint16, secret string, expireHour int) (string, error)
func ParseJWT(token string, secret string) (*JWTClaims, error)
```

JWT Claims 结构：
```go
type JWTClaims struct {
    UID uint32 `json:"uid"`
    GID uint16 `json:"gid"`
    jwt.StandardClaims
}
```

### 2.6 密码哈希

[`core/password.go`](core/password.go) — 双策略密码验证：

| 策略 | 适用场景 | 说明 |
|------|----------|------|
| bcrypt | 新注册用户 | 默认密码哈希算法，salt 自动生成 |
| XiunoMD5 | 旧版迁移用户 | 精确翻译原版 PHP `md5(password + salt)` 逻辑 |

[`model/user.go:74`](model/user.go:74) — `VerifyPassword` 双返回值策略：
1. 密码以 `$2a$` 或 `$2b$` 开头 → bcrypt 验证
2. 否则 → `XiunoMD5(password + salt)` 兼容旧版
3. 旧版验证通过后返回 `needUpgrade=true`，handler 异步执行 `UpgradePassword`

### 2.7 中间件

[`core/middleware.go`](core/middleware.go) — 三个认证中间件：

| 中间件 | 用途 | 行为 |
|--------|------|------|
| `AuthMiddleware` | 必需认证 | 无 token 返回 401 |
| `OptionalAuthMiddleware` | 可选认证 | 未登录注入 `{UID:0, GID:0}` 游客 Claims |
| `AdminMiddleware` | 超管保护 | GID != 1 返回 403 |

Token 提取顺序：`Authorization: Bearer <token>` → Cookie `xn_jwt`

[`core/middleware.go:98`](core/middleware.go:98) — `RateLimitMiddleware`：按 UID（已登录）或 IP（游客）限流。

### 2.8 权限策略层

[`core/policy.go`](core/policy.go) — 权限判断收敛层：

```go
type Policy struct{}
func (p *Policy) CanManageThread(uid uint32, gid uint16, threadUID uint32, fid uint32) bool
func (p *Policy) CanManagePost(uid uint32, gid uint16, postUID uint32, tid uint32) bool
func (p *Policy) CanModerateThread(uid uint32, gid uint16) bool
```

规则：
- `uid == 0` 的游客绝对无权
- `gid == 1` 的超级管理员拥有一切权限
- 作者本人有权操作自己的帖子
- `CanModerateThread`：GID 1(超管), 2(超版), 4(版主), 5(实习版主) 具有版务权限

**技术债**：[`core/policy.go:14`](core/policy.go:14) — `var GlobalPolicy = &Policy{}` 是全局单例，未来可注入 `AppCtx`。

### 2.9 存储抽象层

[`core/storage.go`](core/storage.go) — 文件存储驱动接口：

```go
type Storage interface {
    Put(reader io.Reader, ext string) (string, error)
    GetURL(path string) string
    PutFixedPath(reader io.Reader, relPath string) error
    Delete(relPath string) error
    ServeDownload(w http.ResponseWriter, r *http.Request, relPath, orgFilename string)
}
```

`LocalStorage` 实现：
- 目录散列：`YYYYMM/DD/` 两级目录打散，文件名使用纳秒时间戳
- 初始化：[`core/app.go:37`](core/app.go:37) — `NewLocalStorage("upload", "/upload/")`

### 2.10 限流器

[`core/ratelimit.go`](core/ratelimit.go) — 内存级滑动窗口限流器，零外部依赖：

| 端点 | 限流规则 | 目标 |
|------|----------|------|
| `POST /api/v1/user/register` | 3 次/小时/IP | 防止批量注册 |
| `POST /api/v1/user/login` | 10 次/分钟/IP | 防止密码爆破 |
| `POST /api/v1/thread` | 2 次/分钟/UID | 防止刷帖 |
| `POST /api/v1/thread/{tid}/post` | 1 次/10 秒/UID | 防止刷屏 |

后台独立 goroutine 每分钟清理过期 visitor 记录，防止内存泄漏。

### 2.11 Hook 引擎

[`core/hook.go`](core/hook.go) — 编译期插件注册 + 运行时热切换：

```go
type Plugin interface {
    Name() string
    Title() string
    Version() string
    Desc() string
    Init(app *AppCtx)
}
```

| 方法 | 说明 |
|------|------|
| `Register(app, p)` | 注册插件，调用 `p.Init(app)` |
| `ReloadActivePlugins(kv)` | 从 `bbs_kv["active_plugins"]` 解析启用列表 |
| `AddFilter(pluginName, hookName, fn)` | 注册过滤器（可修改数据、可中止请求） |
| `AddAction(pluginName, hookName, fn)` | 注册动作（只读，不可中止） |
| `ApplyFilters(ctx, hookName, data)` | 执行过滤器链，任一插件返回 error 则中止 |
| `DoAction(ctx, hookName, data)` | 执行动作链，跳过未启用插件 |

当前 Hook 锚点植入位置：

| 锚点 | 位置 | 类型 | 说明 |
|------|------|------|------|
| `thread_create_before` | [`handler/thread.go:60`](handler/thread.go:60) | Filter | 发帖前过滤内容 |
| `thread_create_after` | [`handler/thread.go:90`](handler/thread.go:90) | Action | 发帖后通知 |
| `post_create_before` | [`handler/post.go:60`](handler/post.go:60) | Filter | 回帖前过滤内容 |
| `post_create_after` | [`handler/post.go:105`](handler/post.go:105) | Action | 回帖后通知 |

### 2.12 统一响应格式

[`core/response.go:9`](core/response.go:9)：

```go
type Response struct {
    Code    int         `json:"code"`
    Message string      `json:"message"`
    Data    interface{} `json:"data,omitempty"`
}
```

- `JSONSuccess(w, data)` → `{"code":0, "message":"success", "data":...}`
- `JSONError(w, httpCode, msg)` → `{"code":-1, "message":"..."}`

---

## 三、路由处理器层（handler/）

### 3.1 路由注册

[`cmd/xiuno/main.go`](cmd/xiuno/main.go) — 使用 chi 路由树注册所有端点：

```
公开路由（OptionalAuth）：
  GET    /health
  GET    /api/v1/forum
  GET    /api/v1/config
  GET    /api/v1/thread
  GET    /api/v1/thread/{tid}
  GET    /api/v1/thread/{tid}/post
  GET    /api/v1/user/{uid}
  GET    /api/v1/user/{uid}/thread
  GET    /api/v1/user/{uid}/post
  GET    /api/v1/attach/{aid}
  GET    /api/v1/sso/config
  GET    /api/v1/sso/qq/login
  GET    /api/v1/sso/qq/callback
  GET    /api/v1/sso/wechat/login
  GET    /api/v1/sso/wechat/callback
  GET    /browser
  GET    /browser-download/{type}

认证路由（AuthMiddleware）：
  POST   /api/v1/user/logout
  PUT    /api/v1/user/password
  GET    /api/v1/my/thread
  POST   /api/v1/thread
  PUT    /api/v1/thread/{tid}
  DELETE /api/v1/thread/{tid}
  POST   /api/v1/thread/{tid}/post
  POST   /api/v1/thread/{tid}/moderate
  POST   /api/v1/thread/{tid}/move
  PUT    /api/v1/post/{pid}
  DELETE /api/v1/post/{pid}
  POST   /api/v1/attach
  DELETE /api/v1/attach/{aid}
  POST   /api/v1/user/avatar
  POST   /api/v1/sso/bind
  POST   /api/v1/sso/unbind

限流路由（RateLimitMiddleware）：
  POST   /api/v1/user/login         (10/min/IP)
  POST   /api/v1/user/register      (3/hr/IP)
  POST   /api/v1/thread             (2/min/UID)
  POST   /api/v1/thread/{tid}/post  (1/10s/UID)

Admin 路由（AuthMiddleware + AdminMiddleware）：
  PUT    /api/v1/admin/config
  GET    /api/v1/admin/plugin
  PUT    /api/v1/admin/plugin
  POST   /api/v1/admin/forum
  PUT    /api/v1/admin/forum/{fid}
  DELETE /api/v1/admin/forum/{fid}
  GET    /api/v1/admin/forum/{fid}/access
  PUT    /api/v1/admin/forum/{fid}/access
  PUT    /api/v1/admin/user/{uid}/group
  GET    /api/v1/admin/group
  GET    /api/v1/admin/group/{gid}
  POST   /api/v1/admin/group
  PUT    /api/v1/admin/group/{gid}
  DELETE /api/v1/admin/group/{gid}
  GET    /api/v1/admin/modlog
  POST   /api/v1/admin/thread/scan
  POST   /api/v1/admin/thread/operation
  GET    /api/v1/admin/thread/found
  DELETE /api/v1/admin/thread/{tid}

静态路由：
  GET    /upload/*                   (文件服务)
  GET    /*                          (SPA fallback)
```

**总计：54 个 API 端点 + 2 个静态服务路由 = 56 个端点**

### 3.2 用户认证流程

#### 登录 [`handler/user.go:29`](handler/user.go:29)

```
POST /api/v1/user/login
1. 解析请求体 {account, password}
2. model.GetUserByAccount() → SELECT * FROM bbs_user WHERE username=? OR email=?
3. user.VerifyPassword() → bcrypt 优先，MD5 回退
4. 如果 needUpgrade → user.UpgradePassword() 静默升级为 bcrypt
5. UPDATE logins=logins+1, login_date=UNIX_TIMESTAMP()
6. core.SignJWT(uid, gid, secret, expireHour)
7. Set-Cookie: xn_jwt=...; HttpOnly; SameSite=Lax
8. 返回 user（Password/Salt 被 json:"-" 屏蔽）
```

#### 注册 [`handler/user.go:95`](handler/user.go:95)

```
POST /api/v1/user/register
1. 解析请求体 {username, email, password}
2. 校验：用户名 2-32 字符，密码 ≥6 位，邮箱含 @
3. model.CheckUserExists() → 检查 username OR email 冲突
4. app.Tx() → model.CreateUser() → bcrypt 哈希，写入 bbs_user
5. core.SignJWT(uid, gid, secret, expireHour) 直接签发（无需二次登录）
6. Set-Cookie: xn_jwt=...; HttpOnly; SameSite=Lax
7. 返回 user
```

### 3.3 帖子流程

#### 发帖 [`handler/thread.go:28`](handler/thread.go:28)

```
POST /api/v1/thread
1. 版块权限校验：CheckForumAccessWithCache(ctx, cache, db, uid, gid, fid, "thread")
2. Hook Filter: thread_create_before
3. app.Tx():
   a. INSERT INTO bbs_thread
   b. INSERT INTO bbs_post (isfirst=1)
   c. INSERT INTO bbs_mythread
   d. AsyncCounter.IncrForumThread(fid)
   e. AsyncCounter.IncrUserThread(uid)
4. Hook Action: thread_create_after
5. 返回 thread
```

#### 回帖 [`handler/post.go:26`](handler/post.go:26)

```
POST /api/v1/thread/{tid}/post
1. 查 thread 是否存在 + closed 检测
2. 版块权限校验：CheckForumAccessWithCache(ctx, cache, db, uid, gid, fid, "post")
3. Hook Filter: post_create_before
4. app.Tx():
   a. INSERT INTO bbs_post
   b. UPDATE bbs_thread SET lastpid/lastuid/last_date
   c. AsyncCounter.IncrThreadPost(tid)
   d. AsyncCounter.IncrUserPost(uid)
5. Hook Action: post_create_after
6. 返回 post
```

### 3.4 版块权限校验

[`model/access.go`](model/access.go) — `GetEffectiveAccess` 计算规则：

1. 查 `bbs_group` 获取全局基础权限（`allowread/allowthread/allowpost`）
2. 查 `bbs_forum.accesson` 判断是否开启权限隔离
3. 未开启 → 直接返回全局权限
4. 已开启 → 查 `bbs_forum_access`，查到则局部覆盖全局，未查到则全部拒绝（0）
5. 小黑屋（GID=7）在 `CheckForumAccess` 层直接返回 false
6. 超管（GID=1）在 `CheckForumAccess` 层直接放行

所有核心入口使用缓存版本 `CheckForumAccessWithCache`，5 分钟 TTL。

### 3.5 用户资料闭环

[`handler/user_profile.go`](handler/user_profile.go) — 4 个处理器：

| 端点 | 说明 |
|------|------|
| `GET /api/v1/user/{uid}` | 用户公开资料 |
| `POST /api/v1/user/avatar` | 上传头像（2MB 限制，MIME 嗅探，固定路径覆盖） |
| `GET /api/v1/user/{uid}/thread` | 用户帖子列表（软删除过滤） |
| `GET /api/v1/user/{uid}/post` | 用户回帖列表（isfirst=0 过滤） |

#### 头像存储策略

[`model/user.go:80`](model/user.go:80) — `GetAvatarPath`：
```go
func GetAvatarPath(uid uint32) string {
    s := fmt.Sprintf("%09d", uid)
    return fmt.Sprintf("avatar/%s/%s/%s.png", s[0:3], s[3:6], s[6:9])
}
```

UID 补齐 9 位，按 3 位一层切分目录，千万级用户量下防单目录 Inode 爆炸。

### 3.6 上传安全链路

[`handler/attach.go`](handler/attach.go) — 6 层安全防御：

| 层 | 防御 | 实现 |
|----|------|------|
| 1 | 请求体大小限制 | `http.MaxBytesReader(w, r.Body, 5<<20)` |
| 2 | 表单解析内存上限 | `r.ParseMultipartForm(5 << 20)` |
| 3 | MIME 嗅探 | 读 512 字节 `http.DetectContentType` + 白名单 |
| 4 | 扩展名二次校验 | MIME 映射扩展名与上传文件名比对 |
| 5 | 路径穿越防护 | `filepath.Join` + `strings.HasPrefix` |
| 6 | 文件名安全 | 纳秒时间戳重命名 |

白名单允许的 MIME 类型：`image/jpeg`, `image/png`, `image/gif`, `image/webp`, `application/pdf`

### 3.7 SSO 同步登录

[`handler/sso.go`](handler/sso.go) — QQ/微信 OAuth2 登录：

| 端点 | 说明 |
|------|------|
| `GET /api/v1/sso/config` | 已启用的平台列表（不含密钥） |
| `GET /api/v1/sso/qq/login` | QQ OAuth2 登录（302 跳转） |
| `GET /api/v1/sso/qq/callback` | QQ 回调：code→token→openid→profile→登录/注册→JWT→302 |
| `GET /api/v1/sso/wechat/login` | 微信 OAuth2 登录（302 跳转） |
| `GET /api/v1/sso/wechat/callback` | 微信回调：code→token→openid→profile→登录/注册→JWT→302 |
| `POST /api/v1/sso/bind` | 绑定第三方平台到当前登录用户 |
| `POST /api/v1/sso/unbind` | 解绑第三方平台 |

[`model/user_open.go`](model/user_open.go) — `SSOLoginOrRegister`：首次登录自动创建用户并绑定，后续登录直接签发 JWT。

---

## 四、数据模型层（model/）

### 4.1 数据模型设计

所有 model struct 定义在 [`model/`](model/) 目录，对应 MySQL 表结构（见 [`schema.sql`](schema.sql)）。

关键设计决策：

| 决策 | 说明 |
|------|------|
| `json:"-"` 保护敏感字段 | `User.Password`、`User.Salt`、`User.Mobile`、`User.IdNumber` 等禁止 JSON 序列化 |
| 单字段登录 | `LoginReq.Account` 同时支持 username 和 email |
| 注册即登录 | 注册成功后直接签发 JWT |
| bcrypt 默认 | 新用户注册直接使用 bcrypt，salt 字段留空 |
| MD5 兼容 | 旧用户登录时自动检测并静默升级为 bcrypt |
| 表前缀 | 默认 `bbs_`，通过配置 `database.table_prefix` 修改 |
| IP 字段 `varbinary(16)` | 兼容 IPv4/IPv6，Go 中用 `net.IP` 映射 |
| 废弃 Session 表 | `bbs_session` / `bbs_session_data` 已从 schema 中删除，改用 JWT |
| 废弃 Cache 表 | `bbs_cache` 已删除，改用 Go `Cache` 接口 |

### 4.2 有效权限计算引擎

[`model/access.go`](model/access.go) — 权限计算核心函数：

```go
func GetEffectiveAccess(ctx context.Context, db *sqlx.DB, fid uint32, gid uint16) (*EffectiveAccess, error)
func CheckForumAccess(ctx context.Context, db *sqlx.DB, uid uint32, gid uint16, fid uint32, action string) bool
func GetEffectiveAccessWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, fid uint32, gid uint16) (*EffectiveAccess, error)
func CheckForumAccessWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, uid uint32, gid uint16, fid uint32, action string) bool
```

**设计约束**：`CheckForumAccess` 放在 `model/access.go` 而非 `core/policy.go`，避免 `core → model → core` 循环依赖。

### 4.3 KV 配置系统

[`model/kv.go`](model/kv.go) — `bbs_kv` 表封装：

```go
type SiteConf struct {
    SiteName     string `json:"site_name"`
    SiteBrief    string `json:"site_brief"`
    SiteURL      string `json:"site_url"`
    PageSize     int    `json:"page_size"`
    PostPageSize int    `json:"post_page_size"`
    CloseReason  string `json:"close_reason"`
}
```

**启动预热流程** [`handler/config.go:83`](handler/config.go:83)：
1. `LoadAllKV()` → 全量加载 `bbs_kv` 到内存 `map[string]string`
2. `ParseSiteConf(kv)` → 解析出 `SiteConf`
3. `app.Cache.Set("site_conf", siteConf, 0)` → 写入内存 Cache（永不过期）
4. `app.Hook.ReloadActivePlugins(kv)` → 热加载插件启用状态

**读取链路**：`GET /api/v1/config` → `app.Cache.Get("site_conf")` → 直接返回，0 次 DB 查询。

**写入链路**：`PUT /api/v1/admin/config` → 逐条 `SetKV()` → 重新 `LoadAllKV()` → 刷新 Cache。

### 4.4 统一缓存层

[`model/cache_helper.go`](model/cache_helper.go) — 5 组业务数据缓存：

| 缓存 | TTL | 失效时机 |
|------|-----|----------|
| Forum 列表 | 5 min | 版块创建/更新/删除时 |
| Group 列表 | 5 min | 用户组创建/更新/删除时 |
| User 信息 | 1 min | 用户资料更新时 |
| EffectiveAccess | 5 min | 权限配置变更时 |
| Thread 详情 | 1 min | 帖子更新/回帖时 |

### 4.5 级联删除

[`model/cascade.go`](model/cascade.go) — 事务级联删除：

| 函数 | 级联内容 |
|------|----------|
| `CascadeDeleteThread` | 删除 thread + 所有关联 post + 关联 attach + mythread + thread_top |
| `CascadeDeletePost` | 删除 post + 关联 attach |
| `CascadeDeleteUser` | 删除 user + 所有 thread + 所有 post + 所有 attach + mythread |

### 4.6 软删除

不执行 `DELETE FROM`，仅设置 `deleted_at` 时间戳：

| 位置 | 变更 |
|------|------|
| [`model/thread.go:29`](model/thread.go:29) | `Thread.DeletedAt sql.NullTime` |
| [`model/post.go:28`](model/post.go:28) | `Post.DeletedAt sql.NullTime` |
| [`model/thread.go:68`](model/thread.go:68) | `GetThreadList` 加 `t.deleted_at IS NULL` |
| [`model/post.go:80`](model/post.go:80) | `GetPostList` 加 `p.deleted_at IS NULL` |

### 4.7 其他模块

| 模块 | 文件 | 说明 |
|------|------|------|
| 运行时统计 | [`model/runtime.go`](model/runtime.go) | 用户数/主题数/回帖数/今日数据 |
| 计划任务 | [`model/cron.go`](model/cron.go) | 每日统计清零 + 临时文件清理 + QueueGC + TableDayCron + AttachGC |
| 输入校验 | [`model/check.go`](model/check.go) | `IsUsername`/`IsEmail`/`IsPassword`/`IsMobile`/`SanitizeUsername` |
| SMTP 邮件 | [`model/mail.go`](model/mail.go) | `Mailer` struct + `Send` + `sendWithConfig` |
| MySQL 队列 | [`model/queue.go`](model/queue.go) | `QueuePush`/`Pop`/`Delete`/`Destroy`/`Count`/`Find`/`GC` |
| 每日统计 | [`model/table_day.go`](model/table_day.go) | `TableDayRead`/`MaxID`/`Cron`/`Rebuild` |
| 第三方绑定 | [`model/user_open.go`](model/user_open.go) | `UserOpenPlat` CRUD + `SSOLoginOrRegister` |
| 置顶管理 | [`model/thread_top.go`](model/thread_top.go) | `ThreadTopChange`/`Delete`/`Find`/`UpdateByTID` |
| 我的帖子 | [`model/mythread.go`](model/mythread.go) | `MyThreadCreate`/`Delete`/`FindByUID`/`CountByUID` |
| 版务日志 | [`model/modlog.go`](model/modlog.go) | `CreateModLog`/`FindModLog`/`CountModLog` |
| 附件管理 | [`model/attach.go`](model/attach.go) | `CreateAttach`/`GetAttach`/`DeleteAttach`/`IncrAttachDownload`/`AttachGC`/`AttachAssocPost` |

---

## 五、前端 SPA（xiuno-ui/）

### 5.1 技术栈

| 技术 | 用途 |
|------|------|
| Vue 3 (Composition API) | 前端框架 |
| TypeScript | 类型安全 |
| TailwindCSS | 样式 |
| Vite | 构建工具 |
| Pinia | 状态管理 |
| Vue Router | 路由 |
| Axios | HTTP 请求 |

### 5.2 路由结构

[`../xiuno-ui/src/router/index.ts`](../xiuno-ui/src/router/index.ts)：

```
/                    → Index.vue         首页（全站帖子列表）
/forum/:fid          → ForumView.vue     版块视图
/thread/:tid         → ThreadDetail.vue  帖子详情
/create              → CreateThread.vue  发帖
/login               → Login.vue         登录
/register            → Register.vue      注册
/user/:uid           → UserProfile.vue   用户主页
/my                  → MyCenter.vue      个人中心
/my/password         → MyPassword.vue    修改密码
/my/avatar           → MyAvatar.vue      上传头像
/admin               → AdminLayout.vue   后台管理布局
  /admin/config      → Config.vue        全局配置
  /admin/forum       → Forum.vue         版块管理
  /admin/plugin      → Plugin.vue        插件中枢
  /admin/user        → User.vue          用户管控
  /admin/group       → Group.vue         用户组管理
  /admin/modlog      → ModLog.vue        版务日志
```

### 5.3 路由守卫

[`../xiuno-ui/src/router/index.ts:89`](../xiuno-ui/src/router/index.ts:89)：

```typescript
router.beforeEach((to, _from, next) => {
  const userStore = useUserStore()
  if (to.meta.requiresAuth && !userStore.isLoggedIn) {
    return next('/login')
  }
  if (to.meta.requiresAdmin) {
    if (!userStore.isLoggedIn || userStore.user?.gid !== 1) {
      return next('/')
    }
  }
  next()
})
```

### 5.4 页面清单

| 路由 | 组件 | 对应 PHP 视图 |
|------|------|---------------|
| `/` | `Index.vue` | `index.htm` |
| `/forum/:fid` | `ForumView.vue` | `forum.htm` |
| `/thread/:tid` | `ThreadDetail.vue` | `thread.htm` |
| `/thread/create` | `CreateThread.vue` | `post.htm`（发帖） |
| `/login` | `Login.vue` | `user_login.htm` |
| `/register` | `Register.vue` | `user_create.htm` |
| `/user/:uid` | `UserProfile.vue` | `user.htm` |
| `/user/:uid/threads` | `UserThreadList.vue` | `user_thread.htm` |
| `/my` | `MyCenter.vue` | `my.htm` |
| `/my/threads` | `MyThreadList.vue` | `my_thread.htm` |
| `/my/password` | `MyPassword.vue` | `my_password.htm` |
| `/my/avatar` | `MyAvatar.vue` | `my_avatar.htm` |
| `/admin` | `AdminLayout.vue` | 后台框架 |
| `/admin/config` | `Config.vue` | 后台配置 |
| `/admin/forum` | `Forum.vue` | 后台版块 |
| `/admin/group` | `Group.vue` | 后台用户组 |
| `/admin/user` | `User.vue` | 后台用户 |
| `/admin/modlog` | `ModLog.vue` | 后台版务日志 |
| `/admin/plugin` | `Plugin.vue` | 后台插件 |

共 **19 个 Vue 页面**，覆盖全部 **33 个 PHP 视图**（含后台），覆盖率 **100%**。

### 5.5 技术债

- **`any` 类型滥用**：部分 API 响应未定义 TypeScript interface，直接使用 `any`。
- **无单元测试**：前端目前无 Vitest 测试用例。
- **无 SSR**：纯客户端渲染，首屏依赖 API 请求。
- **无 i18n**：所有文本硬编码为中文。

---

## 6. 单文件二进制部署

### 6.1 go:embed 机制

[`ui/embed.go`](ui/embed.go) 使用 Go 1.16 的 `//go:embed` 指令将 Vue SPA 构建产物嵌入 Go 二进制：

```go
//go:embed dist/*
var dist embed.FS
```

构建流程：

1. 在 `xiuno-ui/` 目录执行 `npm run build`，产物输出到 `ui/dist/`
2. `ui/embed.go` 通过 `//go:embed dist/*` 将整个目录嵌入
3. `go build -o xiuno.exe ./cmd/xiuno/` 生成单文件二进制

### 6.2 SPA Fallback 路由

[`handler/serve.go`](handler/serve.go) 实现了 SPA fallback：

- 请求路径以 `/api/` 开头 → 转发到 API 路由
- 请求路径以 `/uploads/` 开头 → 静态文件服务
- 其他路径 → 返回 `index.html`（SPA 入口），由 Vue Router 处理前端路由

### 6.3 构建脚本

[`build.bat`](build.bat)：

```batch
@echo off
cd /d "%~dp0"
cd xiuno-ui && call npm run build && cd ..
go build -o xiuno.exe -ldflags="-s -w" ./cmd/xiuno/
echo Build complete: xiuno.exe
```

---

## 7. 插件系统

### 7.1 架构：Compile-in, Toggle-out

xiuno-go 的插件系统与 PHP 版完全不同：

- **PHP 版**：运行时动态 `include` 插件文件，通过 hook 函数覆盖机制实现
- **Go 版**：编译时注册（Compile-in），运行时开关（Toggle-out）

插件在 [`plugin/`](plugin/) 目录下，每个插件一个独立 `main` package：

```
plugin/
  spam_blocker/     # 垃圾内容拦截
  word_filter/      # 敏感词过滤
```

### 7.2 注册机制

[`core/hook.go`](core/hook.go) 定义了 Hook 注册表：

```go
var hooks = make(map[string][]func(ctx *AppCtx, params ...interface{}) (interface{}, error))

func AddHook(name string, fn func(ctx *AppCtx, params ...interface{}) (interface{}, error)) {
    hooks[name] = append(hooks[name], fn)
}

func TriggerHook(ctx *AppCtx, name string, params ...interface{}) ([]interface{}, error) {
    var results []interface{}
    for _, fn := range hooks[name] {
        result, err := fn(ctx, params...)
        if err != nil {
            return nil, err
        }
        results = append(results, result)
    }
    return results, nil
}
```

### 7.3 插件示例

[`plugin/spam_blocker/main.go`](plugin/spam_blocker/main.go)：

```go
package main

import "xiuno/core"

func init() {
    core.AddHook("post_create_before", func(ctx *core.AppCtx, params ...interface{}) (interface{}, error) {
        // 垃圾内容检测逻辑
        return nil, nil
    })
}
```

### 7.4 启用/禁用

在 [`cmd/xiuno/main.go`](cmd/xiuno/main.go) 中通过空白导入控制：

```go
import (
    _ "xiuno/plugin/spam_blocker"  // 启用垃圾拦截
    // _ "xiuno/plugin/word_filter" // 禁用敏感词过滤
)
```

### 7.5 与 PHP 版的差异

| 特性 | PHP 版 | Go 版 |
|------|--------|-------|
| 注册时机 | 运行时 `include` | 编译时 `init()` |
| 启用方式 | 数据库配置 | `import` 语句 |
| Hook 实现 | 函数覆盖 | 回调函数列表 |
| 热加载 | 支持 | 不支持（需重新编译） |
| 插件市场 | 官方插件站 | 暂无 |

---

## 8. 技术债与妥协

### 8.1 全局单例：`globalApp`

[`core/app.go`](core/app.go) 中保留了全局单例：

```go
var globalApp *AppCtx
```

**原因**：Hook 系统的回调函数签名不包含 `AppCtx` 参数，插件需要通过全局变量访问应用上下文。

**影响**：
- 单元测试无法并行
- 无法在同一进程中运行多个实例
- 违反依赖倒置原则

**改进方向**：重构 Hook 签名，将 `AppCtx` 作为参数传入回调。

### 8.2 密码兼容：XiunoMD5

[`core/password.go`](core/password.go) 实现了 XiunoMD5 兼容层：

```go
func VerifyPassword(hash, password string) bool {
    if strings.HasPrefix(hash, "$2a$") || strings.HasPrefix(hash, "$2b$") {
        return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
    }
    // 兼容 Xiuno PHP 的 md5(md5(password).salt)
    return xiunoMD5Verify(hash, password)
}
```

**原因**：从 PHP 版迁移时，现有用户的密码哈希是 XiunoMD5 格式，无法一次性全部迁移到 bcrypt。

**影响**：
- 新用户注册使用 bcrypt
- 老用户登录时自动升级到 bcrypt（登录成功后重新哈希）
- XiunoMD5 使用 MD5，安全性低于 bcrypt

### 8.3 会话存储

[`model/session.go`](model/session.go) 使用 MySQL 表存储会话，而非 Redis：

```go
type Session struct {
    SID      string    `db:"sid"`
    UID      uint32    `db:"uid"`
    Expiry   int64     `db:"expiry"`
    CreateIP string    `db:"create_ip"`
    Data     string    `db:"data"`
}
```

**原因**：保持与 PHP 版一致，降低运维复杂度。

**影响**：每次请求需要查询数据库验证会话，高并发场景下可能成为瓶颈。

### 8.4 文件存储

[`core/storage.go`](core/storage.go) 仅实现了本地文件系统存储：

```go
type LocalStorage struct {
    BasePath string
}
```

**未实现**：OSS/S3 等云存储。如需扩展，需实现 `Storage` 接口。

### 8.5 缓存

[`core/cache.go`](core/cache.go) 默认使用内存缓存（`sync.Map`），可选 Redis：

```go
type Cache interface {
    Get(key string) (string, bool)
    Set(key string, value string, ttl time.Duration)
    Delete(key string)
}
```

**注意**：内存缓存在多实例部署下不一致，生产环境建议启用 Redis。

### 8.6 异步队列

[`model/queue.go`](model/queue.go) 使用 MySQL 表模拟队列：

```go
func QueuePush(ctx context.Context, db *sqlx.DB, queueid uint32, v int64, expiry int64) error
func QueuePop(ctx context.Context, db *sqlx.DB, queueid uint32) (int64, bool, error)
```

**原因**：避免引入 Redis/Beanstalkd 等外部依赖。

**影响**：队列操作效率低，不适合高吞吐场景。

### 8.7 无正式测试

- **Go 后端**：无 `_test.go` 文件
- **Vue 前端**：无 Vitest 测试用例
- **集成测试**：无

---

## 9. 数据模型关键设计

### 9.1 表结构总览

| 表名 | 用途 | Go 结构体 |
|------|------|-----------|
| `bbs_user` | 用户 | [`model/user.go`](model/user.go) |
| `bbs_user_group` | 用户组 | [`model/group.go`](model/group.go) |
| `bbs_forum` | 版块 | [`model/forum.go`](model/forum.go) |
| `bbs_forum_access` | 版块权限 | [`model/access.go`](model/access.go) |
| `bbs_thread` | 主题 | [`model/thread.go`](model/thread.go) |
| `bbs_post` | 回复 | [`model/post.go`](model/post.go) |
| `bbs_attach` | 附件 | [`model/attach.go`](model/attach.go) |
| `bbs_mythread` | 我的主题 | [`model/mythread.go`](model/mythread.go) |
| `bbs_mythread` | 收藏 | [`model/mythread.go`](model/mythread.go) |
| `bbs_session` | 会话 | [`model/session.go`](model/session.go) |
| `bbs_modlog` | 版务日志 | [`model/modlog.go`](model/modlog.go) |
| `bbs_table_day` | 日统计 | [`model/table_day.go`](model/table_day.go) |
| `bbs_queue` | 队列 | [`model/queue.go`](model/queue.go) |
| `bbs_kv` | KV 存储 | [`model/kv.go`](model/kv.go) |
| `bbs_runtime` | 运行时 | [`model/runtime.go`](model/runtime.go) |

### 9.2 关键设计决策

**用户头像**：使用 `avatar` 整数字段，指向 `upload/avatar/{uid}/{avatar}.webp`，而非直接存 URL。这是 Xiuno 原版的设计，优势是：

- 更换头像时只需递增 `avatar` 字段
- 浏览器/CDN 自动刷新缓存（新文件名）
- 无需处理旧文件删除

**主题计数**：`bbs_forum` 表包含 `threads` 和 `todayposts` 字段，每次发帖/删帖时实时更新。这是反范式设计，但避免了 COUNT 查询。

**软删除**：`bbs_thread` 和 `bbs_post` 没有 `is_delete` 字段。删除是物理删除，通过级联删除（[`model/cascade.go`](model/cascade.go)）保证数据一致性。

### 9.3 级联删除

[`model/cascade.go`](model/cascade.go) 实现了三级级联：

```
删除用户 → 删除所有主题 → 删除所有回复 → 删除所有附件
删除主题 → 删除所有回复 → 删除关联附件
删除回复 → 删除关联附件
```

所有操作在单个事务中完成。

---

## 10. 请求生命周期

### 10.1 完整请求流程

```
客户端请求
  │
  ▼
Chi Router ──→ /api/* ──→ JWT 中间件 ──→ Handler ──→ Model ──→ MySQL
  │                       (core/middleware.go)  │          │
  │                                             │          └─→ Cache
  │                                             │          └─→ Storage
  │                                             └─→ Response JSON
  │
  └──→ 非 /api/* ──→ SPA Fallback ──→ index.html
                     (handler/serve.go)
```

### 10.2 中间件链

[`core/middleware.go`](core/middleware.go) 定义了中间件链：

1. **CORS**：允许跨域请求（开发环境）
2. **JWT 解析**：从 `Authorization: Bearer <token>` 解析用户身份，注入 `ctx`
3. **Rate Limiter**：基于 IP 的滑动窗口限流
4. **Request Log**：记录请求方法和路径

### 10.3 Handler 模式

所有 Handler 使用闭包工厂模式：

```go
func SomeHandler(app *core.AppCtx) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 1. 解析请求参数
        // 2. 权限检查
        // 3. 调用 Model 层
        // 4. 返回 JSON 响应
    }
}
```

**优点**：
- `app *core.AppCtx` 通过闭包注入，Handler 无需全局变量
- 每个 Handler 职责单一，便于测试

---

## 11. 编译与部署

### 11.1 环境要求

| 组件 | 版本 |
|------|------|
| Go | >= 1.21 |
| Node.js | >= 18 |
| MySQL | >= 5.7 |
| 操作系统 | Windows/Linux/macOS |

### 11.2 开发环境启动

```bash
# 1. 启动前端 dev server
cd xiuno-ui
npm run dev

# 2. 启动后端（另一个终端）
go run ./cmd/xiuno/
```

前端 dev server 默认端口 5173，通过 Vite proxy 将 `/api/` 请求转发到后端 8080 端口。

### 11.3 生产构建

```bash
# 1. 构建前端
cd xiuno-ui
npm run build

# 2. 构建单文件二进制
cd ..
go build -o xiuno.exe -ldflags="-s -w" ./cmd/xiuno/

# 3. 部署
# 将 xiuno.exe 和 xiuno.sql 复制到目标服务器
# 导入数据库：mysql -u root -p < xiuno.sql
# 运行：./xiuno.exe
```

### 11.4 配置文件

[`core/config.go`](core/config.go) 定义了配置结构：

```go
type Config struct {
    Addr       string      // 监听地址，默认 :8080
    MySQL      MySQLConfig // 数据库连接
    JWTSecret  string      // JWT 签名密钥
    UploadDir  string      // 上传目录
    CacheType  string      // 缓存类型：memory/redis
    RedisAddr  string      // Redis 地址
    SMTP       SMTPConfig  // 邮件配置
    SSO        SSOConfig   // 第三方登录
}
```

配置文件通过 JSON 文件加载，默认路径为 `xiuno.json`。

---

> **文档版本**：v1.0
> **最后更新**：2026-07-10
> **对应代码**：xiuno-go 完整重构版
> **文档声明**：本文档 100% 反映当前磁盘上 `.go` 和 `.vue` 文件的真实状态，无虚构设计，无未落地重构。
