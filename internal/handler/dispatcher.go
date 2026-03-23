package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
)

// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
	*BaseHandler

	userWhiteList   map[string]bool
	taskWhiteList   map[string]bool
	taskScripts     map[string]string // 任务名 -> 脚本路径
	executorBots   map[string]string
	executorOpenID string // 执行机器人的 open_id（用于 @ 提及）
}

// NewDispatcherHandler 创建分发机器人处理器
func NewDispatcherHandler() *DispatcherHandler {
	return &DispatcherHandler{
		BaseHandler:   NewBaseHandler(),
		userWhiteList: make(map[string]bool),
		taskWhiteList: make(map[string]bool),
		taskScripts:   make(map[string]string),
	}
}

// SetAllowedUsers 设置允许的用户列表
func (h *DispatcherHandler) SetAllowedUsers(users []string) {
	h.userWhiteList = make(map[string]bool)
	for _, user := range users {
		h.userWhiteList[user] = true
	}
}

// SetAllowedTasks 设置允许的任务列表
func (h *DispatcherHandler) SetAllowedTasks(tasks []string) {
	h.taskWhiteList = make(map[string]bool)
	for _, task := range tasks {
		h.taskWhiteList[task] = true
	}
}

// SetTaskScripts 设置任务脚本映射 (任务名 -> 脚本路径)
func (h *DispatcherHandler) SetTaskScripts(scripts map[string]string) {
	h.taskScripts = scripts
}

// SetExecutorBots 设置执行机器人 (name -> name)
func (h *DispatcherHandler) SetExecutorBots(bots map[string]string) {
	h.executorBots = bots
}

// SetExecutorOpenID 设置执行机器人的 open_id（用于 @ 提及）
func (h *DispatcherHandler) SetExecutorOpenID(openID string) {
	h.executorOpenID = openID
}

// Handle 处理消息
func (h *DispatcherHandler) Handle(_ context.Context, event interface{}, botClient *bot.BotClient) error {
	// 解析事件
	senderID, err := common.ExtractSenderID(event)
	if err != nil {
		return fmt.Errorf("解析发送者失败: %w", err)
	}

	// 校验用户白名单
	if !h.userWhiteList[senderID] {
		return fmt.Errorf("用户不在白名单中: %s", senderID)
	}

	// 解析消息内容
	rawContent, err := common.ExtractMessageContent(event)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	message, err := common.ParseMessageContent(rawContent)
	if err != nil {
		return fmt.Errorf("解析消息内容失败: %w", err)
	}

	// 群聊时：剥离飞书的 @mention 标记（格式: "@_user_xxx task_name"）
	message = h.stripAtMention(message)

	// 解析任务名
	taskName, err := h.parseTaskName(message)
	if err != nil {
		return fmt.Errorf("解析任务名失败: %w", err)
	}

	// 校验任务白名单
	if !h.taskWhiteList[taskName] {
		return fmt.Errorf("任务不在白名单中: %s", taskName)
	}

	// 发送到机器人群（纯文本，executor 监听群里所有消息）
	chatType, _ := common.ExtractChatType(event)
	fmt.Printf("📤 [%s] 分发任务到机器人群: %s\n", botClient.Name, taskName)
	if err := h.SendToGroup(taskName, botClient); err != nil {
		return fmt.Errorf("分发任务失败: %w", err)
	}

	// 私聊时额外回复用户告知已转发
	if chatType == "p2p" {
		if err := h.replyToUser(event, "任务 ["+taskName+"] 已转发给执行机器人，请等待结果", botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
	}

	return nil
}

// stripAtMention 剥离飞书的 @mention 标记
// 群聊中用户 @ 机器人时，消息内容会包含 "@_user_xxx task_name" 前缀
func (h *DispatcherHandler) stripAtMention(message string) string {
	message = strings.TrimSpace(message)
	// 匹配 @_user_xxx 前缀（飞书使用此格式编码 @mention）
	if strings.HasPrefix(message, "@") {
		rest := strings.TrimPrefix(message, "@")
		// 跳过 user ID 部分，到达实际命令
		parts := strings.Fields(rest)
		if len(parts) > 0 {
			// 如果第一部分是 user ID，则取后续部分
			if strings.HasPrefix(parts[0], "_user_") || strings.HasPrefix(parts[0], "_bot_") {
				return strings.TrimSpace(strings.Join(parts[1:], " "))
			}
		}
	}
	return message
}

// replyToUser 私聊回复用户
func (h *DispatcherHandler) replyToUser(event interface{}, text string, botClient *bot.BotClient) error {
	chatID, err := common.ExtractSenderChatID(event)
	if err != nil {
		return fmt.Errorf("获取会话 ID 失败: %w", err)
	}

	sender := common.NewSender(botClient.LarkClient)
	if err := sender.SendToChatID(chatID, "text", text); err != nil {
		return fmt.Errorf("发送回复失败: %w", err)
	}
	fmt.Printf("📤 [%s] 已回复用户: %s\n", botClient.Name, text)
	return nil
}

// parseTaskName 从消息中解析任务名
func (h *DispatcherHandler) parseTaskName(message string) (string, error) {
	message = strings.TrimSpace(message)

	// 移除 "执行" 前缀（如果有）
	if strings.HasPrefix(message, "执行") {
		message = strings.TrimSpace(message[6:])
	}

	parts := strings.Fields(message)
	if len(parts) == 0 {
		return "", fmt.Errorf("消息格式错误，无法解析任务名")
	}

	taskName := parts[0]
	if taskName == "" {
		return "", fmt.Errorf("任务名为空")
	}

	return taskName, nil
}
