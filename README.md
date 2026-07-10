# Xiuno Go

> 基于 [Xiuno BBS 4.0.4](https://github.com/xiuno/bbs) 的 Go 语言完整重构版

将 Xiuno BBS 从 PHP 栈（PHP + MySQL + 服务端渲染）迁移到 Go + Vue 3 SPA 技术栈，保持原版全部功能的同时，实现单文件二进制部署。

## 技术栈

| 层级 | 技术 |
|------|------|
| 后端语言 | Go 1.21+ |
| HTTP 框架 | [Chi Router](https://github.com/go-chi/chi) |
| 数据库 | MySQL 5.7+ / [sqlx](https://github.com/jmoiron/sqlx) |
| 前端框架 | Vue 3 + TypeScript |
| 状态管理 | Pinia |
| 路由 | Vue Router 4 |
| 构建工具 | Vite |
| 认证 | JWT (HS256) |
| 密码 | bcrypt + XiunoMD5 兼容 |
| 缓存 | 内存 / Redis |
| 存储 | 本地文件系统 |

## 项目结构

```
xiuno-go/
├── cmd/xiuno/          # 入口 main.go
├── core/               # 核心框架层
│   ├── app.go          # 应用上下文 (AppCtx)
│   ├── config.go       # 配置管理
│   ├── cache.go        # 缓存抽象
│   ├── jwt.go          # JWT 认证
│   ├── password.go     # 密码哈希 (bcrypt + XiunoMD5)
│   ├── middleware.go   # HTTP 中间件
│   ├── policy.go       # 权限策略
│   ├── storage.go      # 文件存储抽象
│   ├── ratelimit.go    # 限流器
│   ├── counter.go      # 异步计数器
│   ├── hook.go         # 插件 Hook 注册表
│   └── response.go     # JSON 响应工具
├── handler/            # 路由处理器层 (56 端点)
│   ├── user.go         # 用户注册/登录/资料
│   ├── thread.go       # 主题 CRUD
│   ├── post.go         # 回复 CRUD
│   ├── forum.go        # 版块列表
│   ├── attach.go       # 文件上传
│   ├── sso.go          # QQ/微信 OAuth2 登录
│   ├── admin_*.go      # 后台管理
│   └── serve.go        # SPA 静态文件服务
├── model/              # 数据模型层
│   ├── user.go         # 用户模型
│   ├── thread.go       # 主题模型
│   ├── post.go         # 回复模型
│   ├── forum.go        # 版块模型
│   ├── attach.go       # 附件模型
│   ├── cascade.go      # 级联删除
│   ├── session.go      # 会话管理
│   ├── kv.go           # KV 配置存储
│   └── ...
├── plugin/             # 插件 (Compile-in)
│   ├── spam_blocker/   # 垃圾内容拦截
│   └── word_filter/    # 敏感词过滤
├── ui/                 # 前端产物 + embed 入口
│   └── embed.go        # go:embed 嵌入 SPA
├── reference/          # Xiuno PHP 4.0.4 原版参考
└── xiuno-ui/           # Vue 3 SPA 源码
    └── src/
        ├── views/      # 19 个页面组件
        ├── stores/     # Pinia 状态
        ├── router/     # 路由配置
        └── components/ # 公共组件
```

## 快速开始

### 环境要求

- Go >= 1.21
- Node.js >= 18
- MySQL >= 5.7

### 开发环境

```bash
# 1. 导入数据库
mysql -u root -p < xiuno.sql

# 2. 启动前端 dev server (端口 5173)
cd xiuno-ui
npm install
npm run dev

# 3. 启动后端 (另一个终端，端口 8080)
go run ./cmd/xiuno/
```

前端 dev server 通过 Vite proxy 将 `/api/` 请求转发到后端 8080 端口。

### 生产构建

```bash
# 构建单文件二进制
cd xiuno-ui && npm run build && cd ..
go build -o xiuno.exe -ldflags="-s -w" ./cmd/xiuno/

# 部署：将 xiuno.exe + xiuno.json + xiuno.sql 复制到服务器
```

## 功能覆盖

基于 Xiuno BBS 4.0.4 原版，完整复刻全部功能：

- **用户系统**：注册、登录、资料编辑、密码修改、头像上传
- **版块系统**：多级版块、权限控制、版主管理
- **主题系统**：发帖、编辑、删除、置顶、精华、移动
- **回复系统**：回复、引用、编辑、删除
- **附件系统**：上传、图片/文件管理
- **后台管理**：配置、版块、用户组、用户、插件、版务日志
- **第三方登录**：QQ 登录、微信登录
- **安全机制**：JWT 认证、权限策略、限流、内容过滤

## 架构文档

详见 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

## 技术债与妥协

- `globalApp` 全局单例（Hook 系统限制）
- XiunoMD5 密码兼容层（老用户迁移）
- MySQL 会话存储（无 Redis）
- 仅本地文件系统存储
- 无正式测试用例

详见 [`ARCHITECTURE.md`](ARCHITECTURE.md#8-技术债与妥协)。

## License

MIT
