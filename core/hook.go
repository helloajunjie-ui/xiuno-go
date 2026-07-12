// xiuno-go v2.1.0-beta 尼克修改版
package core

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
)

// Plugin 插件必须实现的接口
type Plugin interface {
	Name() string
	Title() string
	Version() string
	Desc() string
	Init(app *AppCtx) // 注册自己需要的 Hook
}

// HookFunc 钩子回调函数的签名
// data 为传入的数据，返回值中 data 可能被 Filter 修改
type HookFunc func(ctx context.Context, data interface{}) (interface{}, error)

// hookEntry 将 HookFunc 与所属插件绑定
type hookEntry struct {
	pluginName string
	fn         HookFunc
}

// HookManager 核心 Hook 引擎
// Filter: 同步执行，可修改传入数据（如敏感词过滤）
// Action: 旁路执行，不修改原数据（如发帖后送积分）
type HookManager struct {
	mu            sync.RWMutex
	plugins       map[string]Plugin
	activePlugins map[string]bool
	filters       map[string][]hookEntry
	actions       map[string][]hookEntry
}

// NewHookManager 创建 Hook 引擎
func NewHookManager() *HookManager {
	return &HookManager{
		plugins:       make(map[string]Plugin),
		activePlugins: make(map[string]bool),
		filters:       make(map[string][]hookEntry),
		actions:       make(map[string][]hookEntry),
	}
}

// Register 注册插件（在 main.go 中调用）
func (hm *HookManager) Register(app *AppCtx, p Plugin) {
	hm.mu.Lock()
	name := p.Name()
	hm.plugins[name] = p
	// 默认启用
	hm.activePlugins[name] = true
	hm.mu.Unlock()

	// Init 内部会调用 AddFilter/AddAction，它们也需要获取 hm.mu
	// 必须在释放锁之后调用，否则 Go 的 sync.Mutex 不可重入会导致死锁
	p.Init(app)
	log.Printf("[Plugin] Loaded: %s v%s — %s", name, p.Version(), p.Desc())
}

// ReloadActivePlugins 从 bbs_kv 加载 active_plugins 配置，热切换插件启用状态
// kv 为 LoadAllKV 返回的 map，key "active_plugins" 的 value 为 JSON 数组
// 例如: ["SpamBlocker"]
func (hm *HookManager) ReloadActivePlugins(kv map[string]string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	raw, ok := kv["active_plugins"]
	if !ok || raw == "" {
		// 没有配置则全部启用
		for name := range hm.plugins {
			hm.activePlugins[name] = true
		}
		return
	}

	var enabled []string
	if err := json.Unmarshal([]byte(raw), &enabled); err != nil {
		log.Printf("[Hook] 解析 active_plugins 失败: %v，全部启用", err)
		for name := range hm.plugins {
			hm.activePlugins[name] = true
		}
		return
	}

	// 构建启用集合
	enabledSet := make(map[string]bool, len(enabled))
	for _, name := range enabled {
		enabledSet[name] = true
	}

	// 更新每个插件的状态
	for name := range hm.plugins {
		hm.activePlugins[name] = enabledSet[name]
	}

	log.Printf("[Hook] 插件状态已刷新: %d 已注册, %d 已启用", len(hm.plugins), len(enabled))
}

// AddFilter 插件挂载过滤器
// pluginName 为插件名，hookName 为锚点名（如 "thread_create_before"）
func (hm *HookManager) AddFilter(pluginName, hookName string, fn HookFunc) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.filters[hookName] = append(hm.filters[hookName], hookEntry{
		pluginName: pluginName,
		fn:         fn,
	})
}

// AddAction 插件挂载动作
// pluginName 为插件名，hookName 为锚点名（如 "thread_create_after"）
func (hm *HookManager) AddAction(pluginName, hookName string, fn HookFunc) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.actions[hookName] = append(hm.actions[hookName], hookEntry{
		pluginName: pluginName,
		fn:         fn,
	})
}

// ApplyFilters 依次执行所有已启用插件的 Filter
// 如果任一插件返回 error，立刻中断并返回该 error
// 返回最终处理后的 data 和可能的错误
func (hm *HookManager) ApplyFilters(ctx context.Context, hookName string, data interface{}) (interface{}, error) {
	hm.mu.RLock()
	entries := hm.filters[hookName]
	hm.mu.RUnlock()

	var err error
	currentData := data
	for _, entry := range entries {
		hm.mu.RLock()
		isActive := hm.activePlugins[entry.pluginName]
		hm.mu.RUnlock()

		if !isActive {
			continue
		}

		currentData, err = entry.fn(ctx, currentData)
		if err != nil {
			log.Printf("[Hook] Filter '%s' aborted by plugin '%s': %v", hookName, entry.pluginName, err)
			return currentData, err // 核心：抛出 error，中断执行链
		}
	}
	return currentData, nil
}

// DoAction 触发动作
// 按注册顺序依次执行所有 Action，仅执行已启用插件的 Action
// Action 的 error 仅记录不中断
func (hm *HookManager) DoAction(ctx context.Context, hookName string, data interface{}) {
	hm.mu.RLock()
	entries := hm.actions[hookName]
	hm.mu.RUnlock()

	for _, entry := range entries {
		hm.mu.RLock()
		isActive := hm.activePlugins[entry.pluginName]
		hm.mu.RUnlock()

		if !isActive {
			continue
		}

		if _, err := entry.fn(ctx, data); err != nil {
			log.Printf("[Hook] Action %s error (plugin %s): %v", hookName, entry.pluginName, err)
		}
	}
}

// GetPlugins 返回所有已注册插件的副本（用于 Admin API）
func (hm *HookManager) GetPlugins() map[string]Plugin {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	cp := make(map[string]Plugin, len(hm.plugins))
	for k, v := range hm.plugins {
		cp[k] = v
	}
	return cp
}

// IsPluginActive 查询插件是否启用
func (hm *HookManager) IsPluginActive(name string) bool {
	hm.mu.RLock()
	defer hm.mu.RUnlock()
	return hm.activePlugins[name]
}

// LogPlugin 插件日志辅助函数
func LogPlugin(name string, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	log.Printf("[Plugin:%s] %s", name, msg)
}
