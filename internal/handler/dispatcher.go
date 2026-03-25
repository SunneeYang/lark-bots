package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
	"github.com/SunneeYang/lark-bots/internal/handler/matcher"
)

// SemanticMatchConfig 语义匹配配置
type SemanticMatchConfig struct {
	Enabled   bool
	Threshold float64
	Provider  string
	Model     string
	APIKey    string
	BaseURL   string
}

// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
	*BaseHandler

	userWhiteList   map[string]bool
	taskWhiteList   map[string]bool       // 旧模式：任务名白名单
	layeredMatcher  *matcher.LayeredMatcher // 新模式：分层匹配器
}

// NewDispatcherHandler 创建分发机器人处理器
// semanticCfg 用于旧模式精确匹配 + 语义匹配
func NewDispatcherHandler(semanticCfg *SemanticMatchConfig) *DispatcherHandler {
	return &DispatcherHandler{
		BaseHandler:    NewBaseHandler(),
		userWhiteList:  make(map[string]bool),
		taskWhiteList: make(map[string]bool),
	}
}

// SetLayeredMatcher 设置分层匹配器（用于新配置格式）
// 调用此方法后，Dispatcher 进入分层匹配模式，不再使用 taskWhiteList
func (h *DispatcherHandler) SetLayeredMatcher(layeredMatcher *matcher.LayeredMatcher) {
	h.layeredMatcher = layeredMatcher
}

// SetAllowedUsers 设置允许的用户列表
func (h *DispatcherHandler) SetAllowedUsers(users []string) {
	h.userWhiteList = make(map[string]bool)
	for _, user := range users {
		h.userWhiteList[user] = true
	}
}

// SetAllowedTasks 设置允许的任务列表（仅旧模式使用）
func (h *DispatcherHandler) SetAllowedTasks(tasks []string) {
	h.taskWhiteList = make(map[string]bool)
	for _, task := range tasks {
		h.taskWhiteList[task] = true
	}
}

// Handle 处理消息（支持新旧两种配置格式）
func (h *DispatcherHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
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

	// 群聊时：剥离飞书的 @mention 标记
	message = h.stripAtMention(message)

	chatType, _ := common.ExtractChatType(event)

	// 根据是否配置了分层匹配器决定使用哪种模式
	if h.layeredMatcher != nil {
		return h.handleLayeredMode(ctx, event, message, chatType, botClient)
	}
	return h.handleLegacyMode(ctx, event, message, chatType, botClient)
}

// ===== 新分层匹配模式 =====

// handleLayeredMode 使用分层匹配器处理消息
func (h *DispatcherHandler) handleLayeredMode(ctx context.Context, event interface{}, message, chatType string, botClient *bot.BotClient) error {
	result := h.layeredMatcher.Match(ctx, matcher.LayeredMatchInput{UserInput: message})

	// 否定检测
	if result.HasNegation {
		replyMsg := "检测到否定意图（如「不要」「别」），请重新描述你要执行的操作"
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
		return fmt.Errorf("检测到否定意图")
	}

	// 多意图检测
	if len(result.Matches) > 1 {
		replyMsg := "检测到多个匹配的任务，请一次只说一个服务器和一个操作"
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
		return fmt.Errorf("多意图请求: %d 个匹配", len(result.Matches))
	}

	// 无匹配
	if len(result.Matches) == 0 {
		replyMsg := "未找到匹配的任务，请检查输入是否包含服务器名和操作（如「土豆开发服重启」）"
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
		return fmt.Errorf("未找到匹配任务")
	}

	// 单个匹配，执行
	match := result.Matches[0]
	cmd := BuildTaskCommand(match.TaskName)
	msgToSend := cmd.String()

	fmt.Printf("📤 [%s] 分发任务 (分层匹配): %s → %s\n", botClient.Name, message, msgToSend)

	if err := h.SendToGroup(msgToSend, botClient); err != nil {
		return fmt.Errorf("分发任务失败: %w", err)
	}

	// 私聊回复用户
	if chatType == "p2p" {
		replyMsg := fmt.Sprintf("任务 [%s %s] 已转发", match.ExecutorName, match.Operation)
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
	}

	return nil
}

// ===== 旧精确匹配模式 =====

// handleLegacyMode 使用旧级联匹配器处理消息
func (h *DispatcherHandler) handleLegacyMode(ctx context.Context, event interface{}, message, chatType string, botClient *bot.BotClient) error {
	candidates := h.getTaskCandidates()
	if len(candidates) == 0 {
		return fmt.Errorf("无可用任务")
	}

	// 初始化旧匹配器（精确匹配）
	matchers := []matcher.TaskMatcher{matcher.NewExactMatcher()}
	cascade := matcher.NewCascadeMatcher(matchers...)

	result, err := cascade.Match(ctx, message, candidates)
	if err != nil {
		return fmt.Errorf("任务匹配失败: %w", err)
	}

	if result.Confidence == 0 {
		return fmt.Errorf("未找到匹配任务: %s", message)
	}

	matchedTask := result.Matched

	fmt.Printf("📤 [%s] 分发任务 (精确匹配): %s\n", botClient.Name, matchedTask)

	if err := h.SendToGroup(matchedTask, botClient); err != nil {
		return fmt.Errorf("分发任务失败: %w", err)
	}

	if chatType == "p2p" {
		replyMsg := fmt.Sprintf("任务 [%s] 已转发", matchedTask)
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
	}

	return nil
}

// ===== 辅助方法 =====

func (h *DispatcherHandler) getTaskCandidates() []string {
	candidates := make([]string, 0, len(h.taskWhiteList))
	for task := range h.taskWhiteList {
		candidates = append(candidates, task)
	}
	return candidates
}

// stripAtMention 剥离飞书的 @mention 标记
func (h *DispatcherHandler) stripAtMention(message string) string {
	message = strings.TrimSpace(message)
	if strings.HasPrefix(message, "@") {
		rest := strings.TrimPrefix(message, "@")
		parts := strings.Fields(rest)
		if len(parts) > 0 {
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
