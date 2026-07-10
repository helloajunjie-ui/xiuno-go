package spam_blocker

import (
	"context"
	"errors"
	"strings"

	"xiuno/core"
)

// SpamPlugin 防灌水插件示例
type SpamPlugin struct {
	badWords []string
}

// Name 返回插件标识名
func (p *SpamPlugin) Name() string {
	return "SpamBlocker"
}

// Title 返回插件显示名称
func (p *SpamPlugin) Title() string {
	return "防灌水系统"
}

// Version 返回插件版本号
func (p *SpamPlugin) Version() string {
	return "1.0.0"
}

// Desc 返回插件描述
func (p *SpamPlugin) Desc() string {
	return "敏感词过滤，拦截违规内容"
}

// Init 注册插件 Hook
func (p *SpamPlugin) Init(app *core.AppCtx) {
	p.badWords = []string{"澳门博彩", "兼职日结", "色情"}

	// 挂载发帖前内容过滤
	app.Hook.AddFilter(p.Name(), "thread_create_before", p.checkBadWords)
	// 挂载回帖前内容过滤
	app.Hook.AddFilter(p.Name(), "post_create_before", p.checkBadWords)
	// 挂载发帖后动作（仅记录日志示例）
	app.Hook.AddAction(p.Name(), "thread_create_after", p.onThreadCreated)

	core.LogPlugin(p.Name(), "已加载，敏感词列表: %v", p.badWords)
}

// checkBadWords 敏感词过滤器
func (p *SpamPlugin) checkBadWords(ctx context.Context, data interface{}) (interface{}, error) {
	message, ok := data.(string)
	if !ok {
		return data, nil
	}

	lowerMsg := strings.ToLower(message)
	for _, word := range p.badWords {
		if strings.Contains(lowerMsg, strings.ToLower(word)) {
			core.LogPlugin(p.Name(), "拦截包含违禁词的帖子: %s", word)
			return message, errors.New("内容包含违禁词，已拦截")
		}
	}
	return message, nil
}

// onThreadCreated 发帖后动作示例
func (p *SpamPlugin) onThreadCreated(ctx context.Context, data interface{}) (interface{}, error) {
	// 这里可以做积分赠送、通知等旁路操作
	// data 为 map[string]interface{}{"tid": ..., "fid": ..., "uid": ..., "subject": ...}
	core.LogPlugin(p.Name(), "新帖已创建: %v", data)
	return data, nil
}
