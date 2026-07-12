# v2.1.0-beta Release Notes

> 发布日期: 2026-07-12
> 黄金链路稳定版 — 聚焦核心流程，社区驱动边缘场景

---

## 黄金链路（已验证通过）

| 链路 | 状态 |
|------|------|
| 身份流: 注册 → 登录 → JWT 签发 | ✅ |
| 内容流: 创建版块 → 发帖(带标签) → 渲染 → 回复 → 编辑 | ✅ |
| 管理流: 主题切换(颜色/布局) → 前台实时生效 → 软删除 | ✅ |

## 本次变更

### 缓存优化
- [`core/cache.go`](core/cache.go) — `Cache` 接口新增 `DelPrefix(ctx, prefix)` 方法
- [`model/cache_helper.go`](model/cache_helper.go) — `InvalidateAccessCacheByFID` 从 256 次 `Del()` 简化为 1 次 `DelPrefix()`
- 写锁保护 + 前缀匹配遍历，零外部依赖

### Bug 修复
- [`handler/attach.go`](handler/attach.go) — 附件下载 404 修复：通过 `time.Unix(att.CreateDate, 0).Format("200601/02")` 重建日期目录前缀
- [`frontend/src/components/layout/WaterfallList.vue`](frontend/src/components/layout/WaterfallList.vue) — 移除未使用的 `randomHeight` 函数和 `index` 变量
- [`frontend/src/views/ThreadList.vue`](frontend/src/views/ThreadList.vue) — 移除未使用的 `goDetail` 和 `timeAgo` 函数

### 性能
- [`cmd/xiuno/main.go`](cmd/xiuno/main.go) — 静态资源 Cache-Control 策略：
  - `/assets/*` → `public, max-age=31536000, immutable`
  - `index.html` → `no-cache`

### 文档
- [`ARCHITECTURE.md`](ARCHITECTURE.md) — 规范化：移除虚构设计、修正文件计数(64 .go / 24 .vue)、去重、更新版本号
- [`TESTING.md`](TESTING.md) — 规范化：统一状态标记、精简验证步骤
- [`README.md`](README.md) — 修正页面计数

## 技术债（已知，已记录）

| 项目 | 说明 |
|------|------|
| `AsyncCounter` | 内存批处理计数器，2s flush 间隔。用户发帖/回帖计数、帖子浏览/回复计数走异步；版块帖子数已改为事务内直接 DB 操作 |
| `globalApp` | 无全局单例，`AppCtx` 通过依赖注入传递 |
| 测试覆盖 | 不追求 100% 覆盖。边缘场景转为 GitHub `help wanted` issues |

## 构建

```bash
# Docker（推荐）
docker compose up -d

# 原生
go build -ldflags="-s -w" -o xiuno-server ./cmd/xiuno/
cd frontend && npm run build
```

## 下一步

- 社区反馈驱动的边缘场景修复
- 插件系统完善
- SSO 多平台适配
