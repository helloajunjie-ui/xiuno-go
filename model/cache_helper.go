package model

import (
	"context"
	"encoding/json"
	"time"

	"xiuno/core"

	"github.com/jmoiron/sqlx"
)

// ==================== 缓存 Key 常量 ====================

const (
	cachePrefixForumList = "forum:list" // 版块全列表
	cachePrefixForum     = "forum:"     // 单个版块 forum:{fid}
	cachePrefixGroupList = "group:list" // 用户组全列表
	cachePrefixGroup     = "group:"     // 单个用户组 group:{gid}
	cachePrefixUser      = "user:"      // 用户数据 user:{uid}
	cachePrefixAccess    = "access:"    // 权限缓存 access:{fid}:{gid}
	cachePrefixThread    = "thread:"    // 帖子详情 thread:{tid}
)

// 缓存 TTL 常量
const (
	CacheTTLForum  = 5 * time.Minute  // 版块数据变化不频繁，5 分钟
	CacheTTLGroup  = 10 * time.Minute // 用户组极少变化，10 分钟
	CacheTTLUser   = 2 * time.Minute  // 用户数据可能变化，2 分钟
	CacheTTLAccess = 5 * time.Minute  // 权限配置变化不频繁，5 分钟
	CacheTTLThread = 1 * time.Minute  // 帖子详情（浏览数可能变化），1 分钟
	CacheTTLShort  = 30 * time.Second // 短缓存
	CacheTTLNever  = 0                // 0 = 永不过期，由写入/删除时主动失效
)

// ==================== 版块缓存 ====================

// GetForumListWithCache 获取版块全列表（带缓存）
// 缓存 key: forum:list
func GetForumListWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB) ([]Forum, error) {
	cacheKey := cachePrefixForumList

	// 1. 尝试从缓存读取
	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var list []Forum
		if err := json.Unmarshal(data, &list); err == nil {
			return list, nil
		}
	}

	// 2. 缓存未命中，查 DB
	list, err := GetAllForums(ctx, db)
	if err != nil {
		return nil, err
	}

	// 3. 回填缓存
	if encoded, err := json.Marshal(list); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLForum)
	}

	return list, nil
}

// InvalidateForumListCache 失效版块列表缓存（写入/删除版块后调用）
func InvalidateForumListCache(ctx context.Context, cache core.Cache) {
	cache.Del(ctx, cachePrefixForumList)
}

// GetForumWithCache 获取单个版块（带缓存）
// 缓存 key: forum:{fid}
func GetForumWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, fid uint32) (*Forum, error) {
	cacheKey := cachePrefixForum + itoa(int(fid))

	// 1. 尝试从缓存读取
	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var f Forum
		if err := json.Unmarshal(data, &f); err == nil {
			return &f, nil
		}
	}

	// 2. 缓存未命中，查 DB
	f, err := GetForum(ctx, db, fid)
	if err != nil {
		return nil, err
	}

	// 3. 回填缓存
	if encoded, err := json.Marshal(f); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLForum)
	}

	return f, nil
}

// InvalidateForumCache 失效单个版块缓存
func InvalidateForumCache(ctx context.Context, cache core.Cache, fid uint32) {
	cache.Del(ctx, cachePrefixForum+itoa(int(fid)))
}

// ==================== 用户组缓存 ====================

// GetGroupListWithCache 获取用户组全列表（带缓存）
// 缓存 key: group:list
func GetGroupListWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB) ([]Group, error) {
	cacheKey := cachePrefixGroupList

	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var list []Group
		if err := json.Unmarshal(data, &list); err == nil {
			return list, nil
		}
	}

	list, err := GetAllGroups(ctx, db)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(list); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLGroup)
	}

	return list, nil
}

// InvalidateGroupListCache 失效用户组列表缓存
func InvalidateGroupListCache(ctx context.Context, cache core.Cache) {
	cache.Del(ctx, cachePrefixGroupList)
}

// GetGroupWithCache 获取单个用户组（带缓存）
// 缓存 key: group:{gid}
func GetGroupWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, gid uint32) (*Group, error) {
	cacheKey := cachePrefixGroup + itoa(int(gid))

	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var g Group
		if err := json.Unmarshal(data, &g); err == nil {
			return &g, nil
		}
	}

	g, err := GetGroup(ctx, db, gid)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(g); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLGroup)
	}

	return g, nil
}

// InvalidateGroupCache 失效单个用户组缓存
func InvalidateGroupCache(ctx context.Context, cache core.Cache, gid uint32) {
	cache.Del(ctx, cachePrefixGroup+itoa(int(gid)))
}

// ==================== 用户缓存 ====================

// GetUserWithCache 获取用户数据（带缓存）
// 缓存 key: user:{uid}
// 注意：用户数据变更频繁（发帖数、积分等），TTL 较短
func GetUserWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, uid uint32) (*User, error) {
	// 游客不走缓存
	if uid == 0 {
		return nil, nil
	}

	cacheKey := cachePrefixUser + itoa(int(uid))

	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var u User
		if err := json.Unmarshal(data, &u); err == nil {
			return &u, nil
		}
	}

	u, err := GetUserByUID(ctx, db, uid)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(u); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLUser)
	}

	return u, nil
}

// InvalidateUserCache 失效用户缓存（用户信息更新后调用）
func InvalidateUserCache(ctx context.Context, cache core.Cache, uid uint32) {
	if uid == 0 {
		return
	}
	cache.Del(ctx, cachePrefixUser+itoa(int(uid)))
}

// ==================== 权限缓存 ====================

// GetEffectiveAccessWithCache 获取有效权限（带缓存）
// 缓存 key: access:{fid}:{gid}
// 权限配置变化不频繁，但每次请求都会调用，缓存收益极高
func GetEffectiveAccessWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, fid uint32, gid uint16) (*EffectiveAccess, error) {
	// 超管 (gid=1) 永远有权限，无需查库
	if gid == 1 {
		return &EffectiveAccess{AllowRead: 1, AllowThread: 1, AllowPost: 1}, nil
	}

	cacheKey := cachePrefixAccess + itoa(int(fid)) + ":" + itoa(int(gid))

	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var a EffectiveAccess
		if err := json.Unmarshal(data, &a); err == nil {
			return &a, nil
		}
	}

	a, err := GetEffectiveAccess(ctx, db, fid, gid)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(a); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLAccess)
	}

	return a, nil
}

// InvalidateAccessCache 失效权限缓存（权限配置更新后调用）
func InvalidateAccessCache(ctx context.Context, cache core.Cache, fid, gid uint32) {
	cache.Del(ctx, cachePrefixAccess+itoa(int(fid))+":"+itoa(int(gid)))
}

// InvalidateAccessCacheByFID 失效某个版块的所有权限缓存（删除版块时调用）
func InvalidateAccessCacheByFID(ctx context.Context, cache core.Cache, fid uint32) {
	// 遍历所有可能的 gid（1-255），批量失效
	// 更精确的做法是在写入时记录 key，但这里简单遍历
	for gid := uint32(0); gid <= 255; gid++ {
		cache.Del(ctx, cachePrefixAccess+itoa(int(fid))+":"+itoa(int(gid)))
	}
}

// ==================== 帖子缓存 ====================

// GetThreadDetailWithCache 获取帖子详情（带缓存）
// 缓存 key: thread:{tid}
// 帖子内容不变，但浏览数可能变化，TTL 较短
func GetThreadDetailWithCache(ctx context.Context, cache core.Cache, db *sqlx.DB, tid uint32) (*ThreadDetail, error) {
	cacheKey := cachePrefixThread + itoa(int(tid))

	data, ok := cache.Get(ctx, cacheKey)
	if ok && data != nil {
		var d ThreadDetail
		if err := json.Unmarshal(data, &d); err == nil {
			return &d, nil
		}
	}

	d, err := GetThreadDetail(ctx, db, tid)
	if err != nil {
		return nil, err
	}

	if encoded, err := json.Marshal(d); err == nil {
		cache.Set(ctx, cacheKey, encoded, CacheTTLThread)
	}

	return d, nil
}

// InvalidateThreadCache 失效帖子详情缓存（帖子更新/删除后调用）
func InvalidateThreadCache(ctx context.Context, cache core.Cache, tid uint32) {
	cache.Del(ctx, cachePrefixThread+itoa(int(tid)))
}

// ==================== 批量失效辅助 ====================

// InvalidateForumRelated 版块相关写入后统一失效
// 版块列表 + 单个版块 + 该版块所有权限缓存
func InvalidateForumRelated(ctx context.Context, cache core.Cache, fid uint32) {
	InvalidateForumListCache(ctx, cache)
	InvalidateForumCache(ctx, cache, fid)
	InvalidateAccessCacheByFID(ctx, cache, fid)
}

// InvalidateGroupRelated 用户组写入后统一失效
// 用户组列表 + 单个用户组
func InvalidateGroupRelated(ctx context.Context, cache core.Cache, gid uint32) {
	InvalidateGroupListCache(ctx, cache)
	InvalidateGroupCache(ctx, cache, gid)
}
