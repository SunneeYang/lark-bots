package handler

import (
	"context"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error
}

// BaseHandler 基础处理器，提供通用功能
type BaseHandler struct {
	// 通用字段
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}
