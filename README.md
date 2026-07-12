# Xiuno Go

> 基于 [Xiuno BBS 4.0.4](https://github.com/xiuno/bbs) 的 Go 语言完整重构版

将 Xiuno BBS 从 PHP 栈（PHP + MySQL + 服务端渲染）迁移到 Go + Vue 3 SPA 技术栈，保持原版全部功能的同时，实现单文件二进制部署。

## 商业模式：Open-Core + Headless 双轨制

Xiuno Go 采用业界经典的 **「开源引流 + 商业变现」** 策略，将产品划分为两个层级：

| 维度 | 免费/社区版 (Open-Core) | 商业/企业版 (Enterprise) |
|------|------------------------|-------------------------|
| **产品形态** | 10MB 单文件二进制，Vue SPA go:embed | Headless API Server，独立部署前端 |
| **定制能力** | 外观实验室（CSS 变量编辑 + 预设主题包） | 深度定制 UI + 多端覆盖 |
| **交付物** | 一个 exe 文件跑一切 | API Server + React/Next.js 前端 + 小程序/App |
| **目标用户** | 草根站长、极客、个人开发者 | 品牌方、大型社区、B 端客户 |
| **商业目的** | 生态繁荣，口碑占领市场 | 卖多端覆盖能力和 API 技术支持 |

> 详见 [`ARCHITECTURE.md`](ARCHITECTURE.md#一商业模式总览open-core--headless-双轨制)。

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
xiuno/
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
├── handler/            # 路由处理器层 (63 端点)
│   ├── user.go         # 用户注册/登录/资料
│   ├── thread.go       # 主题 CRUD
│   ├── post.go         # 回复 CRUD
│   ├── forum.go        # 版块列表
│   ├── tag.go          # 标签列表/详情/标签下帖子
│   ├── attach.go       # 文件上传
│   ├── sso.go          # QQ/微信 OAuth2 登录
│   ├── admin_*.go      # 后台管理（版块/标签/主题/用户/用户组/插件/日志）
│   └── ...             # 其他处理器
├── model/              # 数据模型层
│   ├── user.go         # 用户模型
│   ├── thread.go       # 主题模型
│   ├── post.go         # 回复模型
│   ├── forum.go        # 版块模型
│   ├── tag.go          # 标签模型 + 帖子-标签关联
│   ├── attach.go       # 附件模型
│   ├── cascade.go      # 级联删除（含标签关联清理）
│   ├── kv.go           # KV 配置存储
│   └── ...
├── plugin/             # 插件 (Compile-in)
│   └── spam_blocker/   # 垃圾内容拦截
├── ui/                 # 前端产物 + embed 入口
│   ├── embed.go        # go:embed 嵌入 SPA
│   └── dist/           # Vue SPA 构建产物
├── frontend/           # Vue 3 SPA 源码
│   └── src/
│       ├── views/      # 24 个页面组件
│       ├── stores/     # Pinia 状态
│       ├── router/     # 路由配置
│       └── components/ # 公共组件
├── schema.sql          # 建表 DDL
├── docker-compose.yml  # Docker 编排
├── Dockerfile          # 多阶段构建
└── xiuno.json          # 应用配置
```

## 快速开始

### 环境要求

- Go >= 1.21
- Node.js >= 18
- MySQL >= 5.7

### 开发环境

```bash
# 1. 导入数据库
mysql -u root -p < schema.sql

# 2. 启动前端 dev server (端口 5173)
cd frontend
npm install
npm run dev

# 3. 启动后端 (另一个终端，端口 8080)
go run ./cmd/xiuno/
```

前端 dev server 通过 Vite proxy 将 `/api/` 请求转发到后端 8080 端口。

### 生产构建

```bash
# 构建单文件二进制
cd frontend && npm run build && cd ..
go build -o xiuno.exe -ldflags="-s -w" ./cmd/xiuno/

# 部署：将 xiuno.exe + xiuno.json + schema.sql 复制到服务器
```

### Docker 部署

```bash
# 构建并启动
docker compose build --no-cache app
docker compose up -d

# 停止
docker compose down

# 完全重建（删除所有数据）
docker compose down -v
docker compose up -d
```

## 功能覆盖

基于 Xiuno BBS 4.0.4 原版，完整复刻全部功能：

- **用户系统**：注册、登录、资料编辑、密码修改、头像上传
- **版块系统**：多级版块、权限控制、版主管理
- **标签系统**：发帖添加标签（回车输入）、标签云聚合、标签筛选帖子
- **主题系统**：发帖、编辑、删除、置顶、精华、移动
- **回复系统**：回复、引用、编辑、删除
- **附件系统**：上传、图片/文件管理
- **后台管理**：控制台概览、全局配置、版块管理、标签管理、主题管理、用户组管理、用户管控、插件中枢、版务日志
- **第三方登录**：QQ 登录、微信登录
- **安全机制**：JWT 认证、权限策略、限流、内容过滤

## 架构文档

详见 [`ARCHITECTURE.md`](ARCHITECTURE.md)。

## 技术债与妥协

- `globalApp` 全局单例（Hook 系统限制）
- XiunoMD5 密码兼容层（老用户迁移）
- MySQL 会话存储（无 Redis）
- 仅本地文件系统存储
- 异步计数器容器重启丢失（版块计数已修复，用户/帖子计数仍依赖 AsyncCounter）
- 无正式测试用例

详见 [`ARCHITECTURE.md`](ARCHITECTURE.md#8-技术债与妥协)。

## License

MIT
