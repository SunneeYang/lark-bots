package router

import (
	"context"
	"fmt"

	"github.com/SunneeYang/lark-bots/internal/bot"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error
}

// MessageRouter 消息路由器
type MessageRouter struct {
	registry *bot.BotRegistry
	handlers map[string]MessageHandler // key: role or bot_name
}

// NewMessageRouter 创建消息路由器
func NewMessageRouter(registry *bot.BotRegistry) *MessageRouter {
	return &MessageRouter{
		registry: registry,
		handlers: make(map[string]MessageHandler),
	}
}

// RegisterHandler 注册处理器
func (r *MessageRouter) RegisterHandler(role string, handler MessageHandler) {
	r.handlers[role] = handler
}

// Route 路由事件到对应的处理器
func (r *MessageRouter) Route(ctx context.Context, event interface{}, receiverBot *bot.BotClient) error {
	if receiverBot == nil {
		return fmt.Errorf("机器人不能为空")
	}

	// 打印路由日志
	fmt.Printf("   🔀 路由到: %s (%s)\n", receiverBot.Name, receiverBot.Role)

	// 首先尝试按机器人名称查找 handler（用于支持每个 executor 独立的 handler）
	handler, exists := r.handlers[receiverBot.Name]
	if !exists {
		// 如果没有找到，则按角色查找 handler
		handler, exists = r.handlers[receiverBot.Role]
		if !exists {
			return fmt.Errorf("未找到机器人 '%s' (角色: %s) 的处理器", receiverBot.Name, receiverBot.Role)
		}
	}

	// 调用 handler
	return handler.Handle(ctx, event, receiverBot)
}
