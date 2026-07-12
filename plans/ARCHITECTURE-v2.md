# Xiuno Go 架构文档 (v2.0 备份)

> **核心定位**：本文档基于磁盘上真实 `.go` 文件生成，100% 反映当前二进制文件的物理状态。
> **最后更新**：2026-07-12
> **编译状态**：`go build ./...` 零错误 ✅ | `npm run build` 零错误 ✅
> **二进制大小**：~8.5 MB（单文件，含完整前端 SPA）

---

## 一、商业模式总览：Open-Core + Headless 双轨制

Xiuno Go 采用业界经典的 **「开源引流 + 商业变现」** 策略，将产品划分为两个层级：

```
                    ┌─────────────────────────────────────┐
                    │         Xiuno Go 产品矩阵             │
                    ├──────────────────┬──────────────────┤
                    │   免费/社区版     │   商业/企业版      │
                    │   (Open-Core)    │   (Enterprise)    │
                    ├──────────────────┼──────────────────┤
                    │ 10MB 单文件二进制 │ Headless API Server│
                    │ Vue SPA go:embed │ 独立部署高级前端    │
                    │ 外观实验室(CSS)   │ 移动端矩阵         │
                    │ 预设主题包        │ API 技术支持       │
                    ├──────────────────┼──────────────────┤
                    │ 目标：生态繁荣    │ 目标：商业变现      │
                    │ 草根站长/极客     │ B端客户/品牌方     │
                    └──────────────────┴──────────────────┘
```

### 1.1 免费/社区版（初级+中级）：主打"生态繁荣"与"开箱即用"

| 维度 | 说明 |
|------|------|
| **产品形态** | 依然是那个完美的 `10MB 单文件二进制`。标准版 Vue 前端直接 `go:embed` 嵌在里面 |
| **定制能力** | 后台提供极简的「外观实验室」。暴露几十个 CSS 变量（主色、圆角、字体、背景图），允许玩家在后台填色号、传图片。甚至可以通过后台的下拉菜单，切换几套内置的排版布局（瀑布流版 / 传统列表版） |
| **商业目的** | 极大地满足草根站长和极客的 DIY 欲望，降低建站门槛，靠口碑占领市场，形成庞大的基础用户基数 |

### 1.2 商业/企业版（高级）：主打"全端覆盖"与"品牌深度定制"

| 维度 | 说明 |
|------|------|
| **产品形态** | **Headless BBS（无头社区架构）**。此时，`xiuno.exe` 退居幕后，变成一个纯粹的、高性能的 API Server |
| **商业交付物** | ① 独立部署的高级前端：基于 React/Next.js 为企业客户定制的、利于极端 SEO 的大型 Web 端；② 移动端矩阵：直接调用同一套 API 开发的微信小程序、抖音小程序、原生 App（Flutter / React Native） |
| **商业目的** | 向有预算的 B 端客户（品牌方、大型社区）卖"多端覆盖能力"、"深度定制化 UI"以及"API 级别的技术支持" |

### 1.3 架构安全底线

> ⚠️ **向后看（技术护城河）**：API 必须严格版本化（当前 `/api/v1/` 命名非常好，要坚持）。商业客户的前端是独立部署的，未来 Go 后端升级，绝不能导致旧版商业 App 的接口崩溃。保持 API 的向下兼容，是最大的商业信誉。

> ⚠️ **向前看（UX 与美学设计）**：在做免费版的「外观实验室」时，**千万不要给用户 100% 的自由**。不懂设计的站长很容易把配色调得极度难看，从而败坏了 Xiuno Go "现代、优雅"的品牌心智。必须做 **"受限的自由"**：提供精心调配的预设主题包（如极简白、赛博暗黑、莫兰迪色系），或者限制色盘范围，用技术手段守住产品的美学底线。

---

## 二、项目结构

```
xiuno/                          # 61 个 .go 文件
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
│   ├── tag.go                  # 标签列表/详情/标签下帖子
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
│   ├── admin_tag.go            # 后台标签管理 CRUD
│   ├── admin_thread.go         # 后台主题管理（扫描/批量操作/硬删除）
│   ├── browser.go              # 浏览器下载页
│   └── sso.go                  # QQ/微信 OAuth2 登录
├── model/                      # 数据模型层
│   ├── tag.go                  # Tag struct + CRUD + 标签-帖子关联
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
│   ├── cascade.go              # 级联删除（含标签关联清理）
│   ├── runtime.go              # 运行时统计
│   ├── cron.go                 # 计划任务
│   ├── check.go                # 输入校验
│   ├── mail.go                 # SMTP 邮件发送
│   ├── queue.go                # MySQL 模拟队列
│   ├── table_day.go            # 每日最大 ID 统计
│   ├── utils.go                # 工具函数（IP2Long, modelLong2IP 等）
│   └── user_open.go            # 第三方登录绑定
├── plugin/                     # 编译期注册的插件
│   └── spam_blocker/main.go    # 防灌水插件（敏感词过滤）
├── frontend/                   # Vue 3 SPA 源码
│   ├── src/
│   │   ├── views/              # 24 个页面组件
│   │   ├── stores/             # Pinia 状态
│   │   ├── router/             # 路由配置
│   │   └── components/         # 公共组件
│   ├── package.json
│   └── vite.config.ts
├── schema.sql                  # 建表 DDL（InnoDB, utf8mb4）
├── go.mod
└── go.sum
```

---

## 三、核心框架层（core/）

### 3.1 AppCtx 依赖注入容器

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

### 3.2 配置加载

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

### 3.3 缓存接口

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

### 3.4 异步计数器

[`core/counter.go:14`](core/counter.go:14) — 消除 InnoDB 行锁热点的异步批量计数器：

```go
type AsyncCounter struct {
    forumThreads map[uint32]int // fid -> +N（已废弃，版块计数改为事务内直接操作 DB）
    userThreads  map[uint32]int // uid -> +N
    userPosts    map[uint32]int // uid -> +N
    threadViews  map[uint32]int // tid -> +N
    threadPosts  map[uint32]int // tid -> +N
}
```

- 2 秒间隔批量刷入 DB，将 `UPDATE ... +1` 行锁串行操作合并为批量 UPDATE
- 覆盖 4 个统计热点：用户发帖数、用户回帖数、帖子浏览数、帖子回复数
- **版块帖子计数已从 AsyncCounter 移除**，改为在 [`model/thread.go:299`](model/thread.go:299) 的 `CreateThreadAndFirstPost` 和 [`model/thread.go:206`](model/thread.go:206) 的 `SoftDeleteThread` 事务内直接操作 DB，避免容器重启丢失计数
- 支持 Decr 方法（`DecrUserThread`、`DecrThreadPost`），使用 `GREATEST(CAST(... AS SIGNED) + ?, 0)` 防止 `BIGINT UNSIGNED` 溢出

### 3.5 JWT 认证

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

### 3.6 密码哈希

[`core/password.go`](core/password.go) — 双策略密码验证：

| 策略 | 适用场景 | 说明 |
|------|----------|------|
| bcrypt | 新注册用户 | 默认密码哈希算法，salt 自动生成 |
| XiunoMD5 | 旧版迁移用户 | 精确翻译原版 PHP `md5(password + salt)` 逻辑 |

[`model/user.go:74`](model/user.go:74) — `VerifyPassword` 双返回值策略：
1. 密码以 `$2a$` 或 `$2b$` 开头 → bcrypt 验证
2. 否则 → `XiunoMD5(password + salt)` 兼容旧版
3. 旧版验证通过后返回 `needUpgrade=true`，handler 异步执行 `UpgradePassword`

### 3.7 中间件

[`core/middleware.go`](core/middleware.go) — 三个认证中间件：

| 中间件 | 用途 | 行为 |
|--------|------|------|
| `AuthMiddleware` | 必需认证 | 无 token 返回 401 |
| `OptionalAuthMiddleware` | 可选认证 | 未登录注入 `{UID:0, GID:0}` 游客 Claims |
| `AdminMiddleware` | 超管保护 | GID != 1 返回 403 |

Token 提取顺序：`Authorization: Bearer <token>` → Cookie `xn_jwt`

[`core/middleware.go:98`](core/middleware.go:98) — `RateLimitMiddleware`：按 UID（已登录）或 IP（游客）限流。

### 3.8 权限策略层

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

### 3.9 存储抽象层

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

### 3.10 限流器

[`core/ratelimit.go`](core/ratelimit.go) — 内存级滑动窗口限流器，零外部依赖：

| 端点 | 限流规则 | 目标 |
|------|----------|------|
| `POST /api/v1/user/register` | 3 次/小时/IP | 防止批量注册 |
| `POST /api/v1/user/login` | 10 次/分钟/IP | 防止密码爆破 |
| `POST /api/v1/thread` | 2 次/分钟/UID | 防止刷帖 |
| `POST /api/v1/thread/{tid}/post` | 1 次/10 秒/UID | 防止刷屏 |

后台独立 goroutine 每分钟清理过期 visitor 记录，防止内存泄漏。

### 3.11 Hook 引擎

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
| `thread_create_before` | [`handler/thread.go:87`](handler/thread.go:87) | Filter | 发帖前过滤内容 |
| `thread_create_after` | [`handler/thread.go:131`](handler/thread.go:131) | Action | 发帖后通知 |
| `post_create_before` | [`handler/post.go:60`](handler/post.go:60) | Filter | 回帖前过滤内容 |
| `post_create_after` | [`handler/post.go:105`](handler/post.go:105) | Action | 回帖后通知 |

### 3.12 统一响应格式

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

## 四、路由处理器层（handler/）

### 4.1 路由注册

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
  GET    /api/v1/tag
  GET    /api/v1/tag/{tagid}
  GET    /api/v1/tag/{tagid}/thread
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
  GET    /api/v1/admin/tag
  POST   /api/v1/admin/tag
  PUT    /api/v1/admin/tag/{tagid}
  DELETE /api/v1/admin/tag/{tagid}

静态路由：
  GET    /upload/*                   (文件服务)
  GET    /*                          (SPA fallback)
```

**总计：61 个 API 端点 + 2 个静态服务路由 = 63 个端点**

### 4.2 用户认证流程

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

### 4.3 帖子流程

#### 发帖 [`handler/thread.go:30`](handler/thread.go:30)

```
POST /api/v1/thread
1. 版块权限校验：CheckForumAccessWithCache(ctx, cache, db, uid, gid, fid, "thread")
2. Hook Filter: thread_create_before
3. app.Tx():
   a. INSERT INTO bbs_thread
   b. INSERT INTO bbs_post (isfirst=1)
   c. INSERT INTO bbs_mythread
   d. UPDATE bbs_forum SET threads = threads + 1（事务内直接操作 DB）
4. 失效版块缓存（InvalidateForumListCache + InvalidateForumCache）
5. 异步计数器：IncrUserThread(uid)（仅用户发帖数，版块计数已由事务处理）
6. 标签处理（事务外）：如果请求包含 `tags` 字段（逗号分隔的标签名），调用 `model.TagSetThreadTags()` → DELETE 旧关联 + INSERT IGNORE 新关联 + UPDATE 标签计数
7. Hook Action: thread_create_after
8. 返回 thread
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

### 4.4 版块权限校验

[`model/access.go`](model/access.go) — `GetEffectiveAccess` 计算规则：

1. 查 `bbs_group` 获取全局基础权限（`allowread/allowthread/allowpost`）
2. 查 `bbs_forum.accesson` 判断是否开启权限隔离
3. 未开启 → 直接返回全局权限
4. 已开启 → 查 `bbs_forum_access`，查到则局部覆盖全局，未查到则全部拒绝（0）
5. 小黑屋（GID=7）在 `CheckForumAccess` 层直接返回 false
6. 超管（GID=1）在 `CheckForumAccess` 层直接放行

所有核心入口使用缓存版本 `CheckForumAccessWithCache`，5 分钟 TTL。

### 4.5 用户资料闭环

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

#### 默认头像

[`model/user.go:120`](model/user.go:120) — `EnsureDefaultAvatar` 在服务器启动时生成 128×128 灰色占位 PNG，写入 `upload/avatar/0.png`。前端 `<img>` 的 `@error` 回退链指向该文件，防止用户未上传头像时显示破碎图片图标。

### 4.6 上传安全链路

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

### 4.7 标签系统

[`handler/tag.go`](handler/tag.go) — 3 个 API 端点：

| 端点 | 说明 |
|------|------|
| `GET /api/v1/tag` | 标签列表（按关联帖子数降序，分页） |
| `GET /api/v1/tag/{tagid}` | 单个标签详情 |
| `GET /api/v1/tag/{tagid}/thread` | 标签下的帖子列表 |

[`model/tag.go`](model/tag.go) — 标签 CRUD + 标签-帖子关联：

```go
type Tag struct {
    TagID      uint32 `db:"tagid" json:"tagid"`
    Name       string `db:"name" json:"name"`
    Threads    uint32 `db:"threads" json:"threads"`
    CreateDate uint32 `db:"create_date" json:"create_date"`
}
```

核心函数：
- `TagSetThreadTags(ctx, db, tid, tags)` — 设置帖子标签：DELETE 旧关联 → INSERT IGNORE 新关联 → UPDATE 标签计数
- `TagFindByTID(ctx, db, tid)` — 获取帖子关联的所有标签
- `TagCreateOrGet(ctx, db, name)` — 创建标签（已存在则返回现有 tagid，幂等）

**发帖标签流程**：[`handler/thread.go:124`](handler/thread.go:124) — `ThreadCreateHandler` 在事务提交后，解析 `req.Tags`（逗号分隔的标签名），调用 `TagSetThreadTags` 建立关联。

**帖子详情标签加载**：[`model/thread.go:155`](model/thread.go:155) — `GetThreadDetail` 在查询帖子后，调用 `TagFindByTID` 填充 `detail.Tags` 字段。

**数据库表**：
- `bbs_tag` — 标签主表（tagid, name UNIQUE, threads, create_date）
- `bbs_thread_tag` — 帖子-标签关联表（tid, tagid 复合主键）

### 4.8 SSO 同步登录

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

## 五、数据模型层（model/）

### 5.1 数据模型设计

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
| IP 字段 `int(11) unsigned` | 存储 IPv4 数值（`uint32`），Go 中用 `uint32` 映射，通过 `IP2Long()`/`modelLong2IP()` 转换 |
| 废弃 Session 表 | `bbs_session` / `bbs_session_data` 已从 schema 中删除，改用 JWT |
| 废弃 Cache 表 | `bbs_cache` 已删除，改用 Go `Cache` 接口 |

### 5.2 有效权限计算引擎

[`model/access.go`](model/access.go) — 权限计算核心函数：

```go
func GetEffectiveAccess