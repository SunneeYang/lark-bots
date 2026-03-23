package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/bot"
)

// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
	*BaseHandler

	// 白名单
	userWhiteList map[string]bool
	taskWhiteList map[string]bool

	// 机器人群 ID
	robotGroupID string
}

// NewDispatcherHandler 创建分发机器人处理器
func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		BaseHandler:   NewBaseHandler(),
		userWhiteList: make(map[string]bool),
		taskWhiteList: make(map[string]bool),
	}
}

// SetWhiteLists 设置白名单
func (h *DispatcherHandler) SetWhiteLists(users, tasks []string) {
	h.userWhiteList = make(map[string]bool)
	for _, user := range users {
		h.userWhiteList[user] = true
	}

	h.taskWhiteList = make(map[string]bool)
	for _, task := range tasks {
		h.taskWhiteList[task] = true
	}
}

// Handle 处理消息
func (h *DispatcherHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	// 解析事件
	senderID, err := extractSenderID(event)
	if err != nil {
		return fmt.Errorf("解析发送者失败: %w", err)
	}

	// 校验用户白名单
	if !h.userWhiteList[senderID] {
		return fmt.Errorf("用户不在白名单中: %s", senderID)
	}

	// 解析消息内容
	message, err := extractMessageContent(event)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 解析任务名
	taskName, err := h.parseTaskName(message)
	if err != nil {
		return fmt.Errorf("解析任务名失败: %w", err)
	}

	// 校验任务白名单
	if !h.taskWhiteList[taskName] {
		return fmt.Errorf("任务不在白名单中: %s", taskName)
	}

	// TODO: 在机器人群 @ 执行机器人
	// TODO: 创建任务记录

	return nil
}

// parseTaskName 从消息中解析任务名
func (h *DispatcherHandler) parseTaskName(message string) (string, error) {
	// 支持格式：
	// - "执行 deploy.sh"
	// - "deploy.sh"
	// - "deploy.sh --env=prod"

	message = strings.TrimSpace(message)

	// 移除 "执行" 前缀（如果有）
	if strings.HasPrefix(message, "执行") {
		message = strings.TrimSpace(message[6:]) // "执行" 是 3 个中文，6 个字节
	}

	// 提取任务名（第一个词）
	parts := strings.Fields(message)
	if len(parts) == 0 {
		return "", fmt.Errorf("消息格式错误，无法解析任务名")
	}

	taskName := parts[0]

	// 验证任务名不为空
	if taskName == "" {
		return "", fmt.Errorf("任务名为空")
	}

	return taskName, nil
}

// extractSenderID 从事件中提取发送者 user_id
func extractSenderID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if userID, ok := sender["user_id"].(string); ok {
				return userID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 sender.user_id")
}
