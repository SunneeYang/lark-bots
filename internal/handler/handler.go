package handler

import (
	"context"
	"fmt"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
)

// MessageHandler 消息处理接口
type MessageHandler interface {
	Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error
}

// BaseHandler 基础处理器，提供通用功能
type BaseHandler struct {
	RobotGroupID string
}

// NewBaseHandler 创建基础处理器
func NewBaseHandler() *BaseHandler {
	return &BaseHandler{}
}

// SetRobotGroupID 设置机器人群 ID
func (h *BaseHandler) SetRobotGroupID(groupID string) {
	h.RobotGroupID = groupID
}

// SendToGroup 发送消息到机器人群
func (h *BaseHandler) SendToGroup(message string, botClient *bot.BotClient) error {
	sender := common.NewSender(botClient.LarkClient)
	sender.SetRobotGroupID(h.RobotGroupID)
	if err := sender.SendToGroup(message); err != nil {
		return fmt.Errorf("发送消息到机器人群失败: %w", err)
	}
	fmt.Printf("📤 [%s] 消息已发送到机器人群: %s\n", botClient.Name, message)
	return nil
}
