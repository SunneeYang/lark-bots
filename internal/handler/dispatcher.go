package handler

import (
	"context"
	"fmt"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
	"github.com/SunneeYang/lark-bots/internal/embedding"
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

	userWhiteList map[string]bool
	taskWhiteList map[string]bool
	matcher       *matcher.CascadeMatcher
}

// NewDispatcherHandler 创建分发机器人处理器
func NewDispatcherHandler(semanticCfg *SemanticMatchConfig) *DispatcherHandler {
	h := &DispatcherHandler{
		BaseHandler:   NewBaseHandler(),
		userWhiteList: make(map[string]bool),
		taskWhiteList: make(map[string]bool),
	}

	// 初始化匹配器：精确匹配 + 语义匹配
	matchers := []matcher.TaskMatcher{matcher.NewExactMatcher()}

	if semanticCfg != nil && semanticCfg.Enabled {
		provider := embedding.NewGLMProvider(semanticCfg.APIKey, semanticCfg.Model)
		embeddingMatcher := matcher.NewEmbeddingMatcher(provider, semanticCfg.Threshold)
		matchers = append(matchers, embeddingMatcher)
	}

	h.matcher = matcher.NewCascadeMatcher(matchers...)

	return h
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

// Handle 处理消息
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

	// 群聊时：剥离飞书的 @mention 标记（格式: "@_user_xxx task_name"）
	message = h.stripAtMention(message)

	// 获取候选任务列表
	candidates := h.getTaskCandidates()
	if len(candidates) == 0 {
		return fmt.Errorf("无可用任务")
	}

	// 任务匹配（使用级联匹配器）
	result, err := h.matcher.Match(ctx, message, candidates)
	if err != nil {
		fmt.Printf("⚠️ [%s] 任务匹配失败: %v\n", botClient.Name, err)
		return fmt.Errorf("任务匹配失败: %w", err)
	}

	if result.Confidence == 0 {
		return fmt.Errorf("未找到匹配任务: %s", message)
	}

	matchedTask := result.Matched

	// 日志输出匹配信息
	if result.Method == "exact" {
		fmt.Printf("📤 [%s] 分发任务 (精确匹配): %s\n", botClient.Name, matchedTask)
	} else {
		fmt.Printf("📤 [%s] 分发任务 (语义匹配: %.0f%%): '%s' → %s\n",
			botClient.Name, result.Confidence*100, message, matchedTask)
	}

	// 发送到机器人群（纯文本，executor 监听群里所有消息）
	chatType, _ := common.ExtractChatType(event)
	if err := h.SendToGroup(matchedTask, botClient); err != nil {
		return fmt.Errorf("分发任务失败: %w", err)
	}

	// 私聊时额外回复用户告知已转发
	if chatType == "p2p" {
		confStr := fmt.Sprintf("%.0f%%", result.Confidence*100)
		if result.Method == "exact" {
			confStr = "100%"
		}
		replyMsg := fmt.Sprintf("任务 [%s] 已转发（匹配度: %s）", matchedTask, confStr)
		if err := h.replyToUser(event, replyMsg, botClient); err != nil {
			return fmt.Errorf("回复用户失败: %w", err)
		}
	}

	return nil
}

// getTaskCandidates 获取候选任务列表（从 taskWhiteList）
func (h *DispatcherHandler) getTaskCandidates() []string {
	candidates := make([]string, 0, len(h.taskWhiteList))
	for task := range h.taskWhiteList {
		candidates = append(candidates, task)
	}
	return candidates
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
