# Xiuno Go 测试文档

> **核心定位**：本文档记录所有测试环境配置、数据库连接方式、测试账号和 API 测试过程。
> **最后更新**：2026-07-12
> **编译状态**：`go build ./...` 零错误 ✅
> **API 测试状态**：全部通过 ✅
> **前端构建**：`npm run build` 零错误 ✅
> **Docker 构建**：`docker compose build` 零错误 ✅
> **依赖版本**：Go 1.26.4 / MySQL 8.0 / chi v5.3.1 / sqlx v1.4.0 / go-sql-driver v1.10.0 / crypto v0.54.0
> **扩展测试覆盖**：C1-C23（头像/插件/用户组/运行时/SSO/速率限制/SpamBlocker/后台管理/TAG/标签云/回帖编辑器/图片渲染/版块计数）— 全部通过 ✅
> **已修复 Bug**：#13 附件下载404 / #19 CascadeDeleteThread标签关联 / #20 版块计数 / #21 回帖编辑器 / #22 图片渲染 / #23 itoa清理 / #24 DelPrefix / #25 Cache-Control — 全部 ✅
> **未测试 API 端点**：17 个（D1-D17，见第十章）
> **未测试前端页面**：9 个（E1-E9，见第十章）
> **边界情况待测**：9 个（F1-F9，见第十章）

---

## 一、测试环境概览

### 1.1 环境拓扑

```
┌──────────────┐     :3307       ┌──────────────┐
│   curl/CLI   │ ───────────────→│  xiuno-app   │
│   (宿主机)    │    HTTP :8080   │  (容器内)     │
└──────────────┘                 └──────┬───────┘
                                         │
                                    DSN  │ xiuno:xiuno123@tcp(mysql:3306)/xiuno
                                         │
                                         ▼
                                   ┌──────────────┐
                                   │  xiuno-mysql  │
                                   │  (容器内)      │
                                   │  MySQL 8.0    │
                                   └──────────────┘
```

### 1.2 宿主机信息

| 项目 | 值 |
|------|-----|
| 操作系统 | Windows 10 |
| 项目路径 | `f:\新建文件夹\Xiuno` |
| Docker 版本 | Docker Compose V2 |
| 默认 Shell | `cmd.exe` |

---

## 二、Docker 环境

### 2.1 容器信息

| 容器名 | 镜像 | 端口映射 | 网络 |
|--------|------|----------|------|
| `xiuno-app` | `xiuno-app:latest`（本地构建） | `0.0.0.0:8080→8080/tcp` | `xiuno_xiuno-net` |
| `xiuno-mysql` | `mysql:8.0` | `0.0.0.0:3307→3306/tcp` | `xiuno_xiuno-net` |

### 2.2 数据库连接

#### 从宿主机直连 MySQL 容器

```bash
# 方式 1：通过 docker exec
docker exec -it xiuno-mysql mysql -uroot -proot123 xiuno

# 方式 2：通过映射端口（宿主机 3307 → 容器 3306）
mysql -h 127.0.0.1 -P 3307 -u root -proot123 xiuno
mysql -h 127.0.0.1 -P 3307 -u xiuno -pxiuno123 xiuno
```

#### 数据库凭据

| 角色 | 用户名 | 密码 | 数据库 |
|------|--------|------|--------|
| root | `root` | `root123` | `xiuno` |
| 应用用户 | `xiuno` | `xiuno123` | `xiuno` |

#### 应用 DSN（容器内）

```
xiuno:xiuno123@tcp(mysql:3306)/xiuno?charset=utf8mb4&parseTime=True
```

### 2.3 Docker 常用命令

```bash
# 构建并启动
cd f:\新建文件夹\Xiuno
docker compose build --no-cache app   # 强制重新构建 app 镜像
docker compose up -d                   # 启动所有服务
docker compose down                    # 停止并移除容器（保留 volume）

# 完全重建（删除所有数据）
docker compose down -v                 # 停止并删除 volume
docker compose up -d                   # 重新创建 volume 并启动

# 查看日志
docker logs xiuno-app --tail 50        # 查看最近 50 行日志
docker logs xiuno-app -f               # 实时跟踪日志

# 数据库重置（清空所有数据）
docker exec -i xiuno-mysql mysql -uroot -proot123 -e "DROP DATABASE IF EXISTS xiuno; CREATE DATABASE xiuno CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;"
docker restart xiuno-app               # 重启应用触发自动建表

# 健康检查
curl http://localhost:8080/health
```

### 2.4 镜像管理

```bash
# 查看所有镜像
docker images --format "table {{.Repository}}:{{.Tag}}\t{{.CreatedAt}}"

# 删除旧镜像
docker rmi xiuno-app:latest
```

---

## 三、测试账号

### 3.1 管理员账号

| 字段 | 值 |
|------|-----|
| 用户名 | `admin` |
| 邮箱 | `admin@xiuno.com` |
| 密码 | `admin123` |
| UID | `1` |
| GID | `1`（超级管理员） |

> **注意**：第一个注册的用户自动获得 GID=1（管理员）。如果数据库已有用户，新注册用户获得 GID=101（普通用户）。

### 3.2 Cookie 文件

| 文件 | 用途 |
|------|------|
| [`cookies_admin.txt`](cookies_admin.txt) | 管理员 JWT Cookie（uid=1, gid=1） |
| [`cookies.txt`](cookies.txt) | 普通用户 JWT Cookie |

Cookie 文件格式为 Netscape HTTP Cookie File，由 curl 自动管理。

---

## 四、API 测试过程

### 4.1 测试前置条件

每次完整测试前，确保：

1. Docker 容器运行中
2. 数据库已初始化（自动建表）
3. 管理员账号已注册（uid=1, gid=1）
4. 至少有一个版块（fid >= 1）

### 4.2 完整测试流程

#### 步骤 1：注册管理员

```bash
curl -s -c cookies_admin.txt -X POST http://localhost:8080/api/v1/user/register ^
  -H "Content-Type: application/json" ^
  -d "{\"username\":\"admin\",\"email\":\"admin@xiuno.com\",\"password\":\"admin123\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success","data":{"uid":1,"gid":1,...}}`

#### 步骤 2：登录（获取新 JWT）

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/user/login ^
  -H "Content-Type: application/json" ^
  -d "{\"account\":\"admin\",\"password\":\"admin123\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success","data":{"uid":1,"gid":1,...}}`

#### 步骤 3：创建版块

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/admin/forum ^
  -H "Content-Type: application/json" ^
  -d "{\"name\":\"测试版块\",\"brief\":\"这是一个测试版块\",\"rank\":1}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success","data":{"fid":2}}`

#### 步骤 4：查看版块列表

```bash
curl -s http://localhost:8080/api/v1/forum
```

**预期**：返回版块列表，包含 fid=1（默认版块）和 fid=2（测试版块），`threads` 计数正确

#### 步骤 5：发帖

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/thread ^
  -H "Content-Type: application/json" ^
  -d "{\"fid\":2,\"subject\":\"测试帖子标题\",\"message\":\"这是测试帖子的内容，Markdown 格式\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success","data":{"tid":1}}`

#### 步骤 6：查看帖子列表

```bash
curl -s "http://localhost:8080/api/v1/thread?fid=2"
```

**预期**：返回帖子列表，包含刚创建的帖子

#### 步骤 7：查看帖子详情

```bash
curl -s http://localhost:8080/api/v1/thread/1
```

**预期**：返回帖子详情，包含标题、内容、作者信息

#### 步骤 8：回复帖子

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/thread/1/post ^
  -H "Content-Type: application/json" ^
  -d "{\"message\":\"这是回复内容\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success","data":{"pid":2}}`

#### 步骤 9：查看回复列表

```bash
curl -s http://localhost:8080/api/v1/thread/1/post
```

**预期**：返回回复列表，包含刚创建的回复

#### 步骤 10：编辑帖子

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X PUT http://localhost:8080/api/v1/thread/1 ^
  -H "Content-Type: application/json" ^
  -d "{\"subject\":\"修改后的标题\",\"message\":\"修改后的内容\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success"}`

#### 步骤 11：编辑回复

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X PUT http://localhost:8080/api/v1/post/2 ^
  -H "Content-Type: application/json" ^
  -d "{\"message\":\"修改后的回复内容\"}" ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success"}`

#### 步骤 12：删除回复

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X DELETE http://localhost:8080/api/v1/post/2 ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success"}`

#### 步骤 13：删除帖子

```bash
curl -s -c cookies_admin.txt -b cookies_admin.txt -X DELETE http://localhost:8080/api/v1/thread/1 ^
  -H "X-Real-IP: 192.168.1.1"
```

**预期**：`{"code":0,"message":"success"}`

---

## 五、已发现的问题与修复记录

### 5.1 问题 #1：登录 500 — `User` 结构体缺少字段（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | `POST /api/v1/user/login` 返回 500 |
| **根因** | `model/user.go` 的 `User` 结构体缺少 `QQ`, `Golds`, `Rmbs` 字段，`sqlx` 的 `SELECT *` 扫描 `bbs_user` 表时无法映射这些列 |
| **修复** | 在 `User` 结构体添加缺失字段 |
| **涉及文件** | [`model/user.go`](model/user.go) |

### 5.2 问题 #2：版块列表 500 — `rank` 是 MySQL 保留关键字（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | `GET /api/v1/forum` 返回 500 |
| **根因** | `bbs_forum` 表的 `rank` 字段是 MySQL 保留关键字，`ORDER BY rank` 和 `INSERT ... (rank)` 未加反引号导致 SQL 语法错误 |
| **修复** | 所有 `rank` 引用加反引号：`` `rank` `` |
| **涉及文件** | [`model/forum.go`](model/forum.go), [`model/access.go`](model/access.go) |

### 5.3 问题 #3：发帖 500 — `net.IP` 无法存入 `int(11) unsigned`（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | `POST /api/v1/thread` 返回 500 "发帖失败" |
| **根因** | `CreateThreadAndFirstPost` 和 `CreateReply` 的 `userIP` 参数类型为 `net.IP`，调用 `.To16()` 传入 `[]byte`（16 字节）到 `bbs_thread.userip int(11) unsigned` 列，MySQL 无法转换 |
| **修复** | 1. 新增 `IP2Long()` 函数将 `net.IP` 转为 `uint32`<br>2. `CreateThreadAndFirstPost` 参数 `userIP net.IP` → `uint32`<br>3. `CreateReply` 参数 `userIP net.IP` → `uint32`<br>4. `Thread.UserIP` 和 `Post.UserIP` 字段 `net.IP` → `uint32`<br>5. 调用处添加 `model.IP2Long(userIP)` 转换 |
| **涉及文件** | [`model/utils.go`](model/utils.go), [`model/thread.go`](model/thread.go), [`model/post.go`](model/post.go), [`handler/thread.go`](handler/thread.go), [`handler/post.go`](handler/post.go) |

### 5.4 操作注意事项：Docker 镜像未更新导致旧代码运行

`docker compose up -d` 不会自动重新构建镜像。修改代码后需先执行 `docker compose build --no-cache app` 再 `docker compose up -d`。

### 5.5 问题 #5：SSO OAuth2 回调路由路径重复（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | SSO OAuth2 回调路由（QQ/微信登录）无法访问 |
| **根因** | [`cmd/xiuno/main.go`](cmd/xiuno/main.go):154-157 的 SSO 路由注册在 `r.Route("/api/v1", ...)` 块内部，但路径以 `/api/v1/sso/...` 开头，导致实际路径变为 `/api/v1/api/v1/sso/...` |
| **修复** | 将 SSO 路由移出 `r.Route("/api/v1", ...)` 块，放到根路由级别，路径保持 `/api/v1/sso/...` 不变 |
| **涉及文件** | [`cmd/xiuno/main.go`](cmd/xiuno/main.go) |
| **状态** | ✅ **已修复** |

### 5.6 问题 #6：管理日志查询 500 — SQL 拼接错误（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | `GET /api/v1/admin/modlog` 返回 500 "查询版务日志失败" |
| **根因** | [`model/modlog.go`](model/modlog.go):41-69 的 `FindModLog` 函数中，当 `action` 为空时 `baseWhere = "FROM bbs_modlog"`，拼接后的 SQL 变为 `SELECT m.*, u.username FROM bbs_modlog m LEFT JOIN bbs_user u ON m.uid = u.uid FROM bbs_modlog`，重复 `FROM` 导致语法错误 |
| **修复** | 将 `baseWhere` 改为 `WHERE` 子句片段（`WHERE m.action = ?` 或空字符串），COUNT 查询也使用 `FROM bbs_modlog m` 别名形式保持一致 |
| **涉及文件** | [`model/modlog.go`](model/modlog.go) |
| **状态** | ✅ **已修复** |

### 5.7 问题 #7：创建用户组 500 — `bbs_group.gid` 非自增主键（已修复）

| 项目 | 内容 |
|------|------|
| **症状** | `POST /api/v1/admin/group` 返回 500 "创建失败" |
| **根因** | `bbs_group` 表的 `gid` 字段是 `smallint unsigned` 且**没有 `auto_increment`**，`CreateGroup` 的 INSERT 语句未指定 `gid`，MySQL 尝试插入 `gid=0` 与游客组（gid=0）主键冲突 |
| **修复** | `CreateGroup` 先调用 `GroupMaxID()` 获取当前最大 gid，`nextGID = maxID + 1`，INSERT 时显式指定 `gid` 值 |
| **涉及文件** | [`model/group.go`](model/group.go) |
| **状态** | ✅ **已修复** |

---

## 六、扩展测试流程

### 6.1 管理员操作流程

| 编号 | 操作 | 端点 | 结果 |
|------|------|------|------|
| A1 | 注册管理员 | `POST /api/v1/user/register` | ✅ uid=1, gid=1 |
| A2 | 登录管理员 | `POST /api/v1/user/login` | ✅ |
| A3 | 创建版块 | `POST /api/v1/admin/forum` | ✅ fid=2 |
| A4 | 版块列表 | `GET /api/v1/admin/forum` | ✅ 返回 2 个版块 |
| A5 | 更新版块 | `PUT /api/v1/admin/forum/2` | ✅ |
| A6 | 用户组列表 | `GET /api/v1/admin/group` | ✅ 返回 12 个用户组 |
| A7 | 创建用户组 | `POST /api/v1/admin/group` | ✅（修复问题 #7 后） |
| A8 | 用户列表 | `GET /api/v1/admin/user` | ✅ |
| A9 | 获取站点配置 | `GET /api/v1/config` | ✅ |
| A10 | 更新站点配置 | `PUT /api/v1/admin/config` | ✅ |
| A11 | 插件列表 | `GET /api/v1/admin/plugin` | ✅ SpamBlocker active |
| A12 | 管理日志 | `GET /api/v1/admin/modlog` | ✅（修复问题 #6 后） |
| A13 | 版块权限列表 | `GET /api/v1/admin/forum/2/access` | ✅ |
| A14 | 更新版块权限 | `PUT /api/v1/admin/forum/2/access` | ✅ |
| A15 | 后台主题扫描 | `POST /api/v1/admin/thread/scan` | ✅ |
| A16 | 获取主题配置 | `GET /api/v1/theme` | ✅ |
| A17 | 更新主题配置 | `PUT /api/v1/admin/theme` | ✅ |

### 6.2 普通用户操作流程

| 编号 | 操作 | 端点 | 结果 |
|------|------|------|------|
| B1 | 注册普通用户 | `POST /api/v1/user/register` | ✅ uid=2, gid=101 |
| B2 | 普通用户登录 | `POST /api/v1/user/login` | ✅ |
| B3 | 发帖 | `POST /api/v1/thread` | ✅ tid=1 |
| B4 | 查看帖子列表 | `GET /api/v1/thread?fid=2` | ✅ |
| B5 | 帖子详情 | `GET /api/v1/thread/1` | ✅ |
| B6 | 回复帖子 | `POST /api/v1/thread/1/post` | ✅ pid=2 |
| B7 | 回复列表 | `GET /api/v1/thread/1/post` | ✅ |
| B8 | 编辑帖子 | `PUT /api/v1/thread/1` | ✅ |
| B9 | 编辑回复 | `PUT /api/v1/post/2` | ✅ |
| B10 | 个人中心 | `GET /api/v1/my/profile` | ✅ |
| B11 | 我的帖子列表 | `GET /api/v1/my/thread` | ✅ |
| B12 | 修改密码 | `PUT /api/v1/user/password` | ✅ |
| B13 | 新密码重新登录 | `POST /api/v1/user/login` | ✅ |
| B14 | 用户资料 | `GET /api/v1/user/2` | ✅ |
| B15 | 用户帖子列表 | `GET /api/v1/user/2/thread` | ✅ |
| B16 | 用户回复列表 | `GET /api/v1/user/2/post` | ✅ |
| B17 | 退出登录 | `GET /api/v1/user/logout` | ✅ |
| B18 | 未登录发帖（权限验证） | `POST /api/v1/thread` | ✅ 返回"请先登录" |
| B19 | 普通用户访问管理接口（权限验证） | `GET /api/v1/admin/forum` | ✅ 返回"皇家禁地，闲人止步" |

### 6.3 高级功能测试（C 系列）

| 编号 | 操作 | 端点 | 结果 | 备注 |
|------|------|------|------|------|
| C1 | 置顶帖子 | `POST /api/v1/thread/{tid}/moderate` | ✅ | `{"top":1}` 置顶，`{"top":0}` 取消 |
| C2 | 关闭帖子 | `POST /api/v1/thread/{tid}/moderate` | ✅ | `{"closed":1}` 关闭，`{"closed":0}` 重新打开 |
| C3 | 引用回复 | `POST /api/v1/thread/{tid}/post` | ✅ | 传 `quotepid` 参数，返回引用原文 |
| C4 | 附件上传 | `POST /api/v1/attach` | ✅ | multipart/form-data，10MB 限制，MIME 嗅探 |
| C5 | 头像上传 | `POST /api/v1/user/avatar` | ✅ | multipart/form-data，返回 avatar 编号 |
| C6 | 插件开关 | `PUT /api/v1/admin/plugin` | ✅ | SpamBlocker `"active":true/false` |
| C7 | 修改用户组 | `PUT /api/v1/admin/user/{uid}/group` | ✅ | 管理员可修改任意用户组 |
| C8 | 运行时信息 | `GET /api/v1/runtime` | ✅ | 返回用户/主题/回帖/附件统计 |
| C9 | SSO 配置 | `GET /api/v1/sso/config` | ✅ | 返回已启用的第三方登录平台列表 |
| C10 | 速率限制 | `POST /api/v1/thread/{tid}/post` | ✅ | 发帖 1/10s 限制，超限返回"操作过于频繁" |
| C11 | SpamBlocker 敏感词过滤 | `POST /api/v1/thread/{tid}/post` | ✅ | 含"色情"内容被拦截，返回"内容包含违禁词，已拦截" |
| C12 | 附件删除 | `DELETE /api/v1/attach/{aid}` | ✅ | 删除附件记录 |
| C13 | 后台主题扫描 | `POST /api/v1/admin/thread/scan` | ✅ | 返回 queueid 和匹配的 tid 列表 |
| C14 | 后台主题批量操作 | `POST /api/v1/admin/thread/operation` | ✅ | 批量删除 `{"count":1,"tids":[7]}` |
| C15 | 密码重置（send-code） | `POST /api/v1/user/send-code` | ✅ | 速率限制 1/60s 生效；SMTP 未配置时返回 500 |
| C16 | 密码重置（reset-password） | `POST /api/v1/user/reset-password` | ✅ | 无验证码时返回"验证码已过期或未发送" |
| C17 | 用户删除（mod） | `DELETE /api/v1/user/{uid}/delete` | ✅ | 级联删除用户及其所有关联数据 |
| C18 | 外链图片显示（doctype=0 旧帖子） | 前端渲染 | ✅ | `marked.parse` 渲染 doctype=0 内容，图片正常显示 |
| C19 | 上传图片显示 | 前端渲染 | ✅ | 上传后通过 `/upload/*` 静态服务访问，图片正常显示 |
| C20 | TAG 标签系统 | `GET /api/v1/tag` + `GET /api/v1/tag/{tagid}` + `GET /api/v1/tag/{tagid}/thread` | ✅ | 标签列表/详情/标签下帖子列表，全部返回正确数据 |
| C21 | 回帖编辑器初始化 | `POST /api/v1/thread/{tid}/post` | ✅ | `watch` 使用 `{ flush: 'post' }` 确保 DOM 渲染后初始化 Toast UI Editor |
| C22 | Markdown 图片渲染 | 前端渲染 | ✅ | `marked.parse` 使用 `{ async: true }` 模式，`renderedContent` 预渲染后展示 |
| C23 | 版块帖子计数一致性 | `GET /api/v1/forum` + 发帖/删帖 | ✅ | 发帖时 `CreateThreadAndFirstPost` 事务内 `UPDATE bbs_forum SET threads = threads + 1`；软删除时 `SoftDeleteThread` 事务内 `GREATEST(CAST(threads AS SIGNED) - 1, 0)`；启动时 `AutoMigrate` 自动修正 |

---

## 七、配置参考

### 7.1 应用配置 [`xiuno.json`](xiuno.json)

```json
{
  "server": { "addr": ":8080" },
  "database": {
    "dsn": "xiuno:xiuno123@tcp(mysql:3306)/xiuno?charset=utf8mb4&parseTime=True",
    "table_prefix": "bbs_"
  },
  "jwt": { "secret": "xiuno-go-secret-change-me", "expire_hour": 72 }
}
```

### 7.2 环境变量

| 变量 | 用途 | 默认值 |
|------|------|--------|
| `XIUNO_CONFIG` | 配置文件路径 | `xiuno.json` |

---

## 八、快速参考卡片

### 一键重置并测试

```bash
# 1. 重建镜像
cd f:\新建文件夹\Xiuno && docker compose build --no-cache app

# 2. 重启
docker compose down && docker compose up -d

# 3. 等 10 秒后注册管理员
curl -s -c cookies_admin.txt -X POST http://localhost:8080/api/v1/user/register -H "Content-Type: application/json" -d "{\"username\":\"admin\",\"email\":\"admin@xiuno.com\",\"password\":\"admin123\"}" -H "X-Real-IP: 192.168.1.1"

# 4. 创建版块
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/admin/forum -H "Content-Type: application/json" -d "{\"name\":\"测试版块\",\"brief\":\"测试\",\"rank\":1}" -H "X-Real-IP: 192.168.1.1"

# 5. 发帖
curl -s -c cookies_admin.txt -b cookies_admin.txt -X POST http://localhost:8080/api/v1/thread -H "Content-Type: application/json" -d "{\"fid\":2,\"subject\":\"测试\",\"message\":\"测试内容\"}" -H "X-Real-IP: 192.168.1.1"
```

### 查看日志中的错误

```bash
docker logs xiuno-app --tail 20 | findstr "ERROR PANIC"
```

---

## 九、修复记录

### 2026-07-12 — Issue #11: 前端未传 doctype 导致图片不显示

**根因**：[`frontend/src/views/CreateThread.vue`](frontend/src/views/CreateThread.vue) 的 `handleSubmit()` 和 [`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue) 的 `handleReply()` 在 POST 请求中未传递 `doctype` 字段。Go 中 `DocType` 零值为 0（HTML 模式），admin（GID=1）的 doctype=0 不会被后端降级为 Markdown。导致 `message_fmt` 存的是 Markdown 原文（htmlEscape 转义后），但前端 `renderPostContent()` 对 doctype=0 直接输出 `message_fmt` 而不经过 `marked.parse()` 渲染，最终用户看到的是 Markdown 源码文本而非渲染后的 HTML。

**修复**：
- [`frontend/src/views/CreateThread.vue`](frontend/src/views/CreateThread.vue): `handleSubmit()` 添加 `doctype: 2`
- [`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue): `handleReply()` 添加 `doctype: 2`

**验证**：`go build ./...` ✅ / `npm run build` ✅ / POST 发帖/回帖带 `doctype:2` 返回正确 ✅

### 2026-07-12 — Issue #12: 用户级联删除 `[]uint32` IN 查询参数错误（已修复）

**根因**：[`model/cascade.go`](model/cascade.go):172 中 `DELETE FROM bbs_post WHERE pid IN (?)` 直接传入 `[]uint32` 切片作为参数。sqlx 的 `ExecContext` 不支持将切片直接展开为 `IN` 参数，导致 `sql: converting argument $1 type: unsupported type []uint32` 错误。

**修复**：使用 `sqlx.In()` 展开切片参数，再用 `tx.Rebind()` 重写占位符。

**涉及文件**：[`model/cascade.go`](model/cascade.go)

**验证**：
- `DELETE /api/v1/user/3/delete` → `{"code":0,"message":"success"}` ✅
- `GET /api/v1/user/3` → `{"code":-1,"message":"该用户已遁入虚空"}` ✅

### 2026-07-12 — Issue #13: 附件下载 404 — 文件名 vs 路径不匹配（已修复）

**根因**：[`handler/attach.go`](handler/attach.go) 的 `AttachDownloadHandler` 从数据库获取 `att.Filename`（仅文件名，如 `104240.672417202.png`），直接传给 [`core/storage.go`](core/storage.go) 的 `ServeDownload`。但 `ServeDownload` 需要完整的相对路径（如 `202607/12/104240.672417202.png`）。数据库 `bbs_attach.filename` 字段只存储了文件名，未存储目录路径。

**影响**：`GET /api/v1/attach/{aid}` 始终返回 404。

**修复**：采用方案 A（最小侵入），利用 `att.CreateDate`（Unix 时间戳）重建日期目录前缀：

```go
dateDir := time.Unix(att.CreateDate, 0).Format("200601/02")
fullRelPath := dateDir + "/" + att.Filename
app.Storage.ServeDownload(w, r, fullRelPath, att.OrgFilename)
```

**涉及文件**：[`handler/attach.go`](handler/attach.go) — 添加 `"time"` import，3 行代码修复

**技术债**：依赖 `att.CreateDate` 字段准确性；未处理 `Storage.Put` 自定义目录前缀场景

**验证**：`go build ./...` ✅ / `GET /api/v1/attach/{aid}` 返回 HTTP 200 + Content-Disposition ✅

**状态**：✅ **已修复**

### 2026-07-12 — Issue #14: doctype=0 旧帖子图片不显示（已修复）

**根因**：旧帖子（Issue #11 修复前发布）的 `doctype=0`（HTML 模式），但内容实际是 Markdown 格式（Toast UI Editor 输出）。后端 [`model/post.go`](model/post.go):170 对 doctype=0 的 `messageFmt = msg`（原始消息直接存储），前端 [`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue) 的 `renderPostContent` 对 doctype=0 直接输出 `message_fmt`（Markdown 源码文本），不经过 `marked.parse` 渲染。用户看到的是 Markdown 源码（如 `![外链测试](http://...)`）而非渲染后的图片。

**修复**：修改 `renderPostContent`，对 `doctype=0` 也使用 `marked.parse(post.message)` 渲染，兼容旧帖子数据。

**涉及文件**：[`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue)

**验证**：`npm run build` ✅ / 旧帖子 doctype=0 Markdown 正确渲染 ✅ / 外链图片和上传图片均正常显示 ✅

### 2026-07-12 — Issue #15: 数据库中文乱码 — MySQL latin1 连接导致 double-encoded UTF-8（已修复）

**根因**：旧数据库中的数据是通过 MySQL 默认 `latin1` 字符集连接写入的，导致 UTF-8 字节被 double-encoded。例如 `"这是"` 的 UTF-8 字节 `E8 BF 99 E6 98 AF` 被存储为 `C3 A8 C2 BF C2 99 C3 A6 C2 98 C2 AF`。虽然应用 DSN 指定了 `charset=utf8mb4`，但 MySQL 服务端的 `character_set_client` 默认为 `latin1`，导致 `go-sql-driver` 发送的 `SET NAMES utf8mb4` 被 MySQL 以 latin1 解释，数据被双重编码。

**修复**：`docker compose down -v` 删除 volume，`docker compose up -d` 重建全新数据库。新数据库通过 DSN `charset=utf8mb4` 正常存储 UTF-8。

**验证**：
- 新数据库 HEX 检查：`E8 BF 99 E6 98 AF` = 正确 UTF-8 ✅
- API 返回中文正常：`"subject":"外链图片测试 - doctype=0"` ✅
- 前端页面中文显示正常 ✅

**教训**：数据库首次初始化必须确保 `character_set_client` 为 `utf8mb4`；`docker compose down -v` 是彻底解决方式；旧数据需 `mysqldump --default-character-set=utf8mb4` 导出再导入

### 2026-07-12 — Issue #16: 默认头像缺失导致 `<img>` 404 闪烁（已修复）

**根因**：用户未上传头像时，前端 `<img>` 的 `src` 指向 `/upload/avatar/000/000/001.png`（后端 `GetAvatarPath` 返回的三级路径），该文件不存在返回 404。`@error` 回退链指向 `/upload/avatar/0.png`，该文件也不存在，导致最终显示破碎图片图标。同时，`v-if="user"` 在旧构建中未正确编译到 JS，导致 `fetchProfile()` 完成前 `<img>` 已渲染，引发闪烁。

**修复**：
1. [`model/user.go`](model/user.go):120-178 — 新增 `EnsureDefaultAvatar(uploadDir string) error` 函数，在服务器启动时生成 128×128 灰色占位 PNG（浅灰背景 + 深灰人形轮廓），写入 `upload/avatar/0.png`
2. [`cmd/xiuno/main.go`](cmd/xiuno/main.go):35-38 — 在 `AutoMigrate` 后、`InitSiteConf` 前调用 `model.EnsureDefaultAvatar("upload")`，失败仅打 WARN 日志（非致命）
3. 前端 `v-if="user"` 已在 Issue #14 修复中重新构建，现在正确编译为条件渲染

**技术债**：`EnsureDefaultAvatar` 使用 Go 标准库逐像素绘制，非矢量；路径硬编码为 `upload/avatar/0.png`，仅作 `@error` 回退目标

**验证**：`GET /upload/avatar/0.png` → HTTP 200 ✅ / 容器正常启动 ✅ / 登录和 profile 接口正常 ✅

### 2026-07-12 — Issue #17: TAG 标签系统实现（已修复）

**根因**：原版 Xiuno PHP 包含标签（TAG）功能，但 Go 重构版未实现。缺少 `bbs_tag` 和 `bbs_thread_tag` 数据库表、Go 模型层、API 端点、前端页面。

**修复**：完整实现标签系统，覆盖后端到前端全链路。

**涉及文件**：
- [`model/tag.go`](model/tag.go) — 新增：`Tag` 结构体 + `TagList`/`TagRead`/`TagCreate`/`TagCreateOrGet`/`TagFindByTID`/`TagFindThreads`/`TagThreadCount`/`TagSetThreadTags` + `parseTagNames`
- [`model/migration.go`](model/migration.go) — 新增：`bbs_tag` 和 `bbs_thread_tag` 建表 DDL
- [`model/thread.go`](model/thread.go) — 修改：`ThreadDetail` 新增 `Tags []Tag` 字段（`db:"-"`），`GetThreadDetail` 加载标签
- [`handler/tag.go`](handler/tag.go) — 新增：`TagListHandler`/`TagReadHandler`/`TagThreadListHandler`
- [`handler/thread.go`](handler/thread.go) — 修改：`ThreadCreateReq` 新增 `Tags string`，发帖后调用 `TagSetThreadTags`
- [`cmd/xiuno/main.go`](cmd/xiuno/main.go) — 修改：注册 3 个 tag API 路由
- [`frontend/src/views/TagCloud.vue`](frontend/src/views/TagCloud.vue) — 新增：标签云聚合页
- [`frontend/src/views/TagThreadList.vue`](frontend/src/views/TagThreadList.vue) — 新增：标签下帖子列表页
- [`frontend/src/views/ThreadList.vue`](frontend/src/views/ThreadList.vue) — 修改：三栏布局，右侧标签云
- [`frontend/src/views/CreateThread.vue`](frontend/src/views/CreateThread.vue) — 修改：发帖表单添加标签输入（chips 模式，Enter 添加，× 删除，最多 10 个）
- [`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue) — 修改：帖子详情显示标签
- [`frontend/src/router/index.ts`](frontend/src/router/index.ts) — 修改：添加 `/tags` 和 `/tag/:tagid` 路由
- [`frontend/src/components/layout/NavBar.vue`](frontend/src/components/layout/NavBar.vue) — 修改：导航栏添加标签链接

**数据库变更**：
```sql
CREATE TABLE IF NOT EXISTS bbs_tag (
  tagid int(11) unsigned NOT NULL auto_increment,
  name char(32) NOT NULL default '' COMMENT '标签名称',
  threads int(11) unsigned NOT NULL default '0' COMMENT '关联主题数',
  create_date int(11) unsigned NOT NULL default '0' COMMENT '创建时间',
  PRIMARY KEY (tagid),
  UNIQUE KEY name (name)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS bbs_thread_tag (
  tid int(11) unsigned NOT NULL default '0',
  tagid int(11) unsigned NOT NULL default '0',
  PRIMARY KEY (tid, tagid),
  KEY tagid (tagid)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

**API 端点新增**（3 个）：
| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/tag` | GET | 标签列表（按关联帖子数降序，分页） |
| `/api/v1/tag/{tagid}` | GET | 单个标签详情 |
| `/api/v1/tag/{tagid}/thread` | GET | 标签下的帖子列表 |

**技术债**：标签名纯文本输入无自动补全；标签计数实时更新，高并发下可能有短暂不一致

**验证**：`go build ./...` ✅ / `npm run build` ✅ / 标签列表/发帖带标签/帖子详情/标签云/标签筛选 全部通过 ✅

---

### 2026-07-12 — Issue #18: 后台管理功能补全（已修复）

**根因**：后台管理界面缺少控制台概览（Dashboard）、标签管理（Tag）、主题管理（Thread）三个页面，侧边栏导航和路由配置也未包含这些功能。

**修复**：完整实现三个后台管理页面，覆盖后端 API 到前端 UI 全链路。

**涉及文件**：
- [`handler/admin_tag.go`](handler/admin_tag.go) — 新增：标签管理 CRUD API（List/Create/Update/Delete）
- [`frontend/src/views/admin/Dashboard.vue`](frontend/src/views/admin/Dashboard.vue) — 新增：控制台概览页（站点统计卡片 + 快捷入口）
- [`frontend/src/views/admin/Tag.vue`](frontend/src/views/admin/Tag.vue) — 新增：标签管理页（表格 + 内联编辑 + 创建弹窗 + 删除确认）
- [`frontend/src/views/admin/Thread.vue`](frontend/src/views/admin/Thread.vue) — 新增：主题管理页（搜索/扫描/批量操作/硬删除）
- [`frontend/src/views/admin/AdminLayout.vue`](frontend/src/views/admin/AdminLayout.vue) — 修改：侧边栏添加 控制台概览/标签管理/主题管理 链接
- [`frontend/src/router/index.ts`](frontend/src/router/index.ts) — 修改：添加 Dashboard/Tag/Thread 三个 admin 子路由，重定向改为 `/admin`
- [`cmd/xiuno/main.go`](cmd/xiuno/main.go) — 修改：注册 4 个 admin tag API 路由

**API 端点新增**（4 个）：
| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/admin/tag` | GET | 标签列表（分页） |
| `/api/v1/admin/tag` | POST | 创建标签 |
| `/api/v1/admin/tag/{tagid}` | PUT | 更新标签名称 |
| `/api/v1/admin/tag/{tagid}` | DELETE | 删除标签（级联删除关联） |

**技术债**：Dashboard 统计无聚合端点；批量操作依赖 MySQL 队列

**验证**：`go build ./...` ✅ / `npm run build` ✅ / 容器正常启动 ✅ / 标签 CRUD 全部通过 ✅

---

### 2026-07-12 — Issue #19: CascadeDeleteThread 标签关联未清理（已修复）

**根因**：[`model/cascade.go`](model/cascade.go) 的 `CascadeDeleteThread` 在硬删除帖子时，删除了 `bbs_thread`、`bbs_post`、`bbs_mythread`、`bbs_thread_top` 等关联数据，并更新了 `bbs_forum.threads` 和 `bbs_user.threads` 计数器，但**没有删除 `bbs_thread_tag` 关联记录**，也没有递减 `bbs_tag.threads` 计数器。

**症状**：`GET /api/v1/tag/{tagid}/thread` 返回 `"threads":[],"total":2` — `total` 来自 `bbs_thread_tag` 表计数（显示 2 条），但实际查询 JOIN `bbs_thread` 并过滤 `deleted_at IS NULL`，返回空结果。数据不一致。

**修复**：在 [`model/cascade.go`](model/cascade.go):67-76 的 `CascadeDeleteThread` 函数中新增步骤 5.1：
```go
// 5.1 删除标签关联并更新标签计数
var tagIDs []uint32
_ = tx.SelectContext(ctx, &tagIDs, `SELECT tagid FROM bbs_thread_tag WHERE tid = ?`, tid)
if len(tagIDs) > 0 {
    _, _ = tx.ExecContext(ctx, `DELETE FROM bbs_thread_tag WHERE tid = ?`, tid)
    for _, tagid := range tagIDs {
        _, _ = tx.ExecContext(ctx,
            `UPDATE bbs_tag SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE tagid = ?`, tagid)
    }
}
```

**验证**：创建带标签帖子 → 删除前计数一致 ✅ → 硬删除后标签计数归零 ✅ → 删除后标签帖子列表一致 ✅

---

### 2026-07-12 — Issue #20: 版块帖子计数错误（已修复）

**根因**：版块帖子计数（`bbs_forum.threads`）存在三个问题：

1. **`CreateThreadAndFirstPost` 未更新版块计数**：发帖时仅依赖 `AsyncCounter.IncrForumThread` 异步递增，但 `AsyncCounter` 是纯内存计数器，每 2 秒 flush 一次。容器重启会丢失所有未 flush 的计数，导致版块计数永久偏低。

2. **`SoftDeleteThread` 未更新版块计数**：软删除帖子时仅依赖 `AsyncCounter.DecrForumThread` 异步递减，同样面临容器重启丢失计数的问题。

3. **`AsyncCounter` 非持久化**：`core/counter.go` 的异步计数器是纯内存实现，容器重启后所有未 flush 的增量/减量全部丢失。

**修复**（涉及 7 处修改）：

1. [`model/thread.go`](model/thread.go):297-302 — `CreateThreadAndFirstPost` 在事务内新增 `UPDATE bbs_forum SET threads = threads + 1 WHERE fid = ?`，发帖时直接操作数据库，不依赖异步计数器。

2. [`model/thread.go`](model/thread.go):185-212 — `SoftDeleteThread` 返回类型从 `error` 改为 `(uint32, error)`（返回 `fid` 供调用方失效缓存）。函数内先查询帖子的 `fid`，软删除后执行 `UPDATE bbs_forum SET threads = GREATEST(CAST(threads AS SIGNED) - 1, 0) WHERE fid = ?`。

3. [`handler/thread.go`](handler/thread.go):108-110 — `ThreadCreateHandler` 移除 `app.Counter.IncrForumThread(req.Fid)` 调用（避免重复计数），改为 `InvalidateForumListCache` + `InvalidateForumCache` 使版块缓存立即失效。

4. [`handler/thread_manage.go`](handler/thread_manage.go):120-148 — `ThreadDeleteHandler` 移除 `app.Counter.DecrForumThread(uint32(thread.FID))` 调用（已在 `SoftDeleteThread` 事务中处理），使用 `SoftDeleteThread` 返回的 `deletedFID` 失效版块缓存。

5. [`handler/admin_thread.go`](handler/admin_thread.go):154-170 — `AdminThreadOperationHandler` 的 `delete` 分支使用 `SoftDeleteThread` 返回的 `deletedFID` 失效版块缓存。

6. [`handler/admin_thread.go`](handler/admin_thread.go):257 — `AdminThreadDeleteHandler`（硬删除）在 `CascadeDeleteThread` 后失效版块缓存。

7. [`model/migration.go`](model/migration.go):364-377 — `AutoMigrate` 新增启动时修正步骤：
```sql
UPDATE bbs_forum f
SET f.threads = (
    SELECT COUNT(*) FROM bbs_thread t
    WHERE t.fid = f.fid AND t.deleted_at IS NULL
)
```
每次应用启动时自动修正版块计数，确保即使 `AsyncCounter` 丢失计数也能恢复。

**技术债**：
- `bbs_forum.threads` 仍是 denormalized 计数器字段，未移除
- `AsyncCounter` 仍用于 `IncrUserThread`、`DecrUserThread`、`DecrThreadPost` 等非关键路径（允许短暂不一致）
- 启动时修正使用全表扫描子查询，版块数量少时无性能问题

**验证**：发帖/软删除/硬删除后 `threads` 即时 ±1 ✅ / 容器重启后 `AutoMigrate` 自动修正 ✅ / `go build ./...` ✅

---

### 2026-07-12 — Issue #21: 回帖输入框不显示（已修复）

**根因**：[`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue) 中，`initReplyEditor()` 初始化 Toast UI Editor 的逻辑使用 `watch(loading, callback)` 监听 `loading` 状态变化。当 `fetchDetail()` 异步加载完成后设置 `loading = false`，`watch` 回调触发，但此时 DOM 尚未更新（`v-if="!loading"` 的条件渲染还未完成），导致 `document.getElementById('reply-box')` 返回 `null`，编辑器初始化失败。

**修复**：将 `watch` 的第三个参数改为 `{ flush: 'post' }`，确保回调在 DOM 更新完成后执行：
```typescript
watch(loading, (newVal) => {
  if (!newVal) {
    initReplyEditor()
  }
}, { flush: 'post' })
```

**涉及文件**：[`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue)

**验证**：`npm run build` ✅ / 回复输入框正常显示 ✅ / Toast UI Editor 工具栏正常 ✅ / 提交回复成功 ✅

---

### 2026-07-12 — Issue #22: 图片渲染为链接文本（已修复）

**根因**：[`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue) 的 `renderPostContent()` 函数使用 `marked.parse()` 同步调用。但在浏览器 ESM 环境下，`marked` v18.0.5 的 `parse()` 返回 `Promise<string>` 而非 `string`。Vue 模板中 `v-html="renderPostContent(detail)"` 将 Promise 对象渲染为字符串 `[object Promise]`，导致所有 Markdown 内容（包括图片 `![name](url)`）显示为未渲染的源码文本。

**修复**（涉及 3 处修改）：

1. 将同步 `renderPostContent()` 替换为异步 `renderContent()`：
```typescript
async function renderContent(post: { doctype?: number; message_fmt?: string; message?: string }): Promise<string> {
  const doctype = post.doctype ?? 2
  const msg = (doctype === 0 || doctype === 2) ? post.message_fmt || '' : post.message || ''
  if (!msg) return ''
  if (doctype === 1) return `<pre>${msg}</pre>`
  return await marked.parse(msg, { async: true })
}
```

2. 新增 `renderedContent` 响应式引用存储预渲染 HTML：
```typescript
const renderedContent = ref<Record<number, string>>({})
```

3. 新增 `renderAllContent()` 在 `fetchDetail()` 数据加载完成后调用，遍历主帖和所有回复，异步渲染后存入 `renderedContent`：
```typescript
async function renderAllContent() {
  const map: Record<number, string> = {}
  if (detail.value) {
    map[-1] = await renderContent(detail.value)
  }
  if (replyList.value) {
    for (const reply of replyList.value) {
      map[reply.pid] = await renderContent(reply)
    }
  }
  renderedContent.value = map
}
```

4. 模板从 `v-html="renderPostContent(detail)"` 改为 `v-html="renderedContent[-1] || ''"`，从 `v-html="renderPostContent(reply)"` 改为 `v-html="renderedContent[reply.pid] || ''"`。

**涉及文件**：[`frontend/src/views/ThreadDetail.vue`](frontend/src/views/ThreadDetail.vue)

**验证**：`npm run build` ✅ / 图片 Markdown 正确渲染 ✅ / 外链/上传图片正常 ✅ / 刷新后正常 ✅

### 2026-07-12 — Issue #23: itoa 自定义函数清理（已修复）

**根因**：[`model/kv.go`](model/kv.go) 和 [`model/cache_helper.go`](model/cache_helper.go) 使用了自定义的 `itoa(int) string` 和 `parseInt(string) int` 辅助函数，而非 Go 标准库 `strconv.Itoa` / `strconv.Atoi`。这些自定义函数是 PHP 迁移遗留代码，增加了维护负担和潜在的边界情况处理不一致。

**修复**：
1. [`model/kv.go`](model/kv.go) — 添加 `"strconv"` import，替换 2 处 `parseInt(v)` → `strconv.Atoi(v)`，替换 2 处 `itoa(conf.PageSize)` / `itoa(conf.PostPageSize)` → `strconv.Itoa(...)`，删除自定义 `parseInt` 和 `itoa` 函数（原 lines 230-241）
2. [`model/cache_helper.go`](model/cache_helper.go) — 添加 `"strconv"` import，替换全部 12 处 `itoa(int(...))` → `strconv.Itoa(int(...))`

**影响**：零行为变更，纯代码清理。删除 ~12 行自定义函数代码。

**验证**：`go build ./...` ✅

**状态**：✅ **已修复**

---

### 2026-07-12 — Issue #24: Cache 接口 DelPrefix 优化（已修复）

**根因**：[`model/cache_helper.go`](model/cache_helper.go) 的 `InvalidateAccessCacheByFID` 使用 256 次 `Del()` 调用的暴力迭代方式失效权限缓存：

```go
// 优化前
for gid := uint32(0); gid <= 255; gid++ {
    cache.Del(ctx, cachePrefixAccess+strconv.Itoa(int(fid))+":"+strconv.Itoa(int(gid)))
}
```

**修复**：
1. [`core/cache.go`](core/cache.go) — `Cache` 接口新增 `DelPrefix(ctx context.Context, prefix string)` 方法，`memoryCache` 实现使用写锁保护的全量 map 遍历 + 字符串前缀匹配
2. [`model/cache_helper.go`](model/cache_helper.go) — `InvalidateAccessCacheByFID` 简化为单次 `DelPrefix` 调用：

```go
// 优化后
cache.DelPrefix(ctx, cachePrefixAccess+strconv.Itoa(int(fid))+":")
```

**影响**：
- 接口变更：`Cache` 接口新增方法，所有实现需适配
- 性能：256 次 map 查找 + 256 次 `Del` 调用 → 1 次 map 遍历（O(n)）
- 未来 Redis 迁移：`DelPrefix` 可直接映射为 `SCAN 0 MATCH prefix*` + `DEL key1 key2 ...`

**技术债**：`memoryCache.DelPrefix` 使用写锁保护的全量 map 遍历，百万级键时可能阻塞其他操作数十毫秒。

**验证**：`go build ./...` ✅

**状态**：✅ **已修复**

---

### 2026-07-12 — Issue #25: 静态资源 Cache-Control 优化（已修复）

**根因**：[`cmd/xiuno/main.go`](cmd/xiuno/main.go) 的 SPA 文件服务路由未设置任何缓存头，浏览器每次请求都重新下载所有静态资源，浪费带宽并影响页面加载速度。

**修复**：在 [`cmd/xiuno/main.go`](cmd/xiuno/main.go) 的 SPA 文件服务 handler 中添加 Cache-Control 策略：

| 路径模式 | Cache-Control | 理由 |
|----------|---------------|------|
| `/assets/*` | `public, max-age=31536000, immutable` | Vite 构建产物含内容哈希指纹，内容变更时 URL 自动变化 |
| `index.html` | `no-cache` | SPA 入口，需始终检查最新版本 |

**涉及文件**：[`cmd/xiuno/main.go`](cmd/xiuno/main.go)

**验证**：`go build ./...` ✅ / `/assets/*` 返回 `Cache-Control: public, max-age=31536000, immutable` ✅ / `index.html` 返回 `no-cache` ✅

**状态**：✅ **已修复**

---

## 十、未测试功能清单

以下 API 端点和前端功能尚未纳入自动化测试，按优先级排列：

### 10.1 未测试的 API 端点

| 编号 | 端点 | 方法 | 功能 | 优先级 | 备注 |
|------|------|------|------|--------|------|
| D1 | `/api/v1/thread/{tid}/move` | POST | 移动帖子到其他版块 | 中 | 需要两个版块 |
| D2 | `/api/v1/admin/thread/found` | GET | 查看已扫描的主题列表 | 低 | 依赖扫描队列 |
| D3 | `/api/v1/admin/thread/{tid}` | DELETE | 后台硬删除主题 | 低 | 级联删除 |
| D4 | `/api/v1/admin/group/{gid}` | GET | 读取单个用户组 | 低 | 读取单个用户组详情 |
| D5 | `/api/v1/admin/group/{gid}` | PUT | 更新用户组 | 低 | 修改用户组名称/权限 |
| D6 | `/api/v1/admin/group/{gid}` | DELETE | 删除用户组 | 低 | 删除自定义用户组 |
| D7 | `/api/v1/admin/forum/{fid}` | DELETE | 删除版块 | 低 | 删除版块及关联数据 |
| D8 | `/api/v1/forum/{fid}` | GET | 读取单个版块详情 | 低 | 公开接口 |
| D9 | `/api/v1/user/synlogin` | GET | SSO 同步登录 | 低 | 需要 SSO 服务端配置 |
| D10 | `/api/v1/sso/bind` | POST | SSO 绑定第三方账号 | 低 | 需要 SSO 服务端配置 |
| D11 | `/api/v1/sso/unbind` | POST | SSO 解绑第三方账号 | 低 | 需要 SSO 服务端配置 |
| D12 | `/api/v1/sso/qq/login` | GET | QQ 登录跳转 | 低 | 需要 QQ 互联 AppID |
| D13 | `/api/v1/sso/qq/callback` | GET | QQ 登录回调 | 低 | 需要 QQ 互联 AppID |
| D14 | `/api/v1/sso/wechat/login` | GET | 微信登录跳转 | 低 | 需要微信开放平台凭据 |
| D15 | `/api/v1/sso/wechat/callback` | GET | 微信登录回调 | 低 | 需要微信开放平台凭据 |
| D16 | `/browser` | GET | 浏览器兼容提示页 | 低 | 纯静态 HTML |
| D17 | `/browser-download/{type}` | GET | 浏览器下载重定向 | 低 | 302 跳转 |

### 10.2 未测试的前端页面功能

| 编号 | 页面路由 | 功能 | 优先级 | 备注 |
|------|----------|------|--------|------|
| E1 | `/my/avatar` | 用户头像上传页面 | 中 | 前端调用 `POST /api/v1/user/avatar` |
| E2 | `/my/password` | 修改密码页面 | 中 | 前端调用 `PUT /api/v1/user/password` |
| E3 | `/admin/config` | 后台全局配置页 | 中 | 站点名称、SEO 等配置 |
| E4 | `/admin/forum` | 后台版块管理页 | 中 | 创建/编辑/删除版块 |
| E5 | `/admin/plugin` | 后台插件中枢页 | 中 | 启用/禁用插件 |
| E6 | `/admin/user` | 后台用户管控页 | 中 | 用户列表、修改用户组 |
| E7 | `/admin/group` | 后台用户组管理页 | 中 | 创建/编辑/删除用户组 |
| E8 | `/admin/modlog` | 后台版务日志页 | 中 | 查看操作日志 |
| E9 | `/admin/theme` | 后台外观实验室 | 中 | 主题配置（CSS 变量 + 布局切换） |
| E10 | 搜索功能 | ThreadList.vue 搜索框 | 中 | 调用 `GET /api/v1/thread?keyword=...` |

### 10.3 边界情况 / 压力测试

| 编号 | 场景 | 优先级 | 备注 |
|------|------|--------|------|
| F1 | 超长帖子内容（>64KB） | 低 | 数据库 longtext 上限测试 |
| F2 | 特殊字符（XSS 注入、SQL 注入） | 中 | DOMPurify 和参数化查询验证 |
| F3 | 并发发帖（竞态条件） | 低 | 计数器原子性验证 |
| F4 | 超大文件上传（>10MB） | 中 | 上传限制验证 |
| F5 | 非法 MIME 类型上传 | 中 | MIME 嗅探绕过测试 |
| F6 | 无效 JWT token | 中 | 认证中间件容错 |
| F7 | 过期 JWT token | 中 | 认证中间件容错 |
| F8 | 数据库断连恢复 | 低 | 连接池重试机制 |
| F9 | 多语言内容（非中文 UTF-8） | 低 | 表情符号、日文、韩文 |
