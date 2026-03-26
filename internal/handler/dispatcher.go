package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"

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
	groupProjectMap map[string]string      // 群组 ID 到项目名的映射（自动补充项目关键词）
	userInfoCache   map[string]string      // OpenID → 真实姓名缓存
	userInfoCacheMu sync.RWMutex           // 保护 userInfoCache 的读写锁
}

// NewDispatcherHandler 创建分发机器人处理器
// semanticCfg 用于旧模式精确匹配 + 语义匹配
func NewDispatcherHandler(semanticCfg *SemanticMatchConfig) *DispatcherHandler {
	return &DispatcherHandler{
		BaseHandler:    NewBaseHandler(),
		userWhiteList:  make(map[string]bool),
		taskWhiteList:  make(map[string]bool),
		userInfoCache:  make(map[string]string), // 初始化用户信息缓存
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

// SetGroupProjectMap 设置群组到项目名的映射（用于自动补充项目关键词）
func (h *DispatcherHandler) SetGroupProjectMap(groupProjectMap map[string]string) {
	h.groupProjectMap = groupProjectMap
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
func (h *DispatcherHandler) handleLayeredMode(ctx context.Context, event interface{}, message, _ string, botClient *bot.BotClient) error {
	// 自动补充群组的项目关键词
	enhancedMessage := h.enhanceMessageWithGroupProject(ctx, event, message)

	result := h.layeredMatcher.Match(ctx, matcher.LayeredMatchInput{UserInput: enhancedMessage})

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

	// 回复用户（支持群聊和私聊）
	taskDisplay := match.DisplayName
	if taskDisplay == "" {
		// 如果未配置 display_name，使用任务名
		taskDisplay = match.TaskName
	}
	replyMsg := fmt.Sprintf("✅ 任务 [%s] 开始执行", taskDisplay)
	if err := h.replyToUser(event, replyMsg, botClient); err != nil {
		return fmt.Errorf("回复用户失败: %w", err)
	}

	return nil
}

// ===== 旧精确匹配模式 =====

// handleLegacyMode 使用旧级联匹配器处理消息
func (h *DispatcherHandler) handleLegacyMode(ctx context.Context, event interface{}, message, _ string, botClient *bot.BotClient) error {
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

	// 回复用户（支持群聊和私聊）
	replyMsg := fmt.Sprintf("✅ 任务 [%s] 开始执行", matchedTask)
	if err := h.replyToUser(event, replyMsg, botClient); err != nil {
		return fmt.Errorf("回复用户失败: %w", err)
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

// replyToUser 回复用户消息（统一方案，支持群聊和私聊）
// 优先使用回复消息方式（ReplyToMessage），回退到发送消息到会话（SendToChatID）
func (h *DispatcherHandler) replyToUser(event interface{}, text string, botClient *bot.BotClient) error {
	// 尝试提取 message_id，用于回复消息
	messageID, err := common.ExtractMessageID(event)
	if err == nil && messageID != "" {
		// 使用回复消息的方式
		sender := common.NewSender(botClient.LarkClient)
		if err := sender.ReplyToMessage(messageID, "text", text); err != nil {
			return fmt.Errorf("回复消息失败: %w", err)
		}
		fmt.Printf("📤 [%s] 已回复消息: %s\n", botClient.Name, text)
		return nil
	}

	// 回退到私聊方式
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

// enhanceMessageWithGroupProject 根据群组配置自动补充项目关键词
// 如果用户输入中已经包含项目名，则不再补充
func (h *DispatcherHandler) enhanceMessageWithGroupProject(_ context.Context, event interface{}, message string) string {
	// 未配置群组项目映射，直接返回原始消息
	if len(h.groupProjectMap) == 0 {
		return message
	}

	// 提取 chat_id
	chatID, err := common.ExtractSenderChatID(event)
	if err != nil {
		// 无法提取 chat_id，返回原始消息
		return message
	}

	// 查找群组配置的项目名
	projectName, exists := h.groupProjectMap[chatID]
	if !exists || projectName == "" {
		// 群组未配置项目名，返回原始消息
		return message
	}

	// 检查用户输入是否已经包含项目关键词
	// 如果用户已经明确指定了项目（如 "土豆"、"迷雾"），则不自动补充
	commonProjectKeywords := []string{"土豆", "potato", "迷雾", "mist", "番茄", "tomato"}
	lowerMessage := strings.ToLower(message)
	for _, keyword := range commonProjectKeywords {
		if strings.Contains(lowerMessage, strings.ToLower(keyword)) {
			// 用户已明确指定项目，不自动补充
			fmt.Printf("🔍 [Dispatcher] 用户已指定项目，不自动补充: '%s'\n", message)
			return message
		}
	}

	// 自动补充群组的项目关键词
	enhanced := fmt.Sprintf("%s %s", projectName, message)
	fmt.Printf("🔍 [Dispatcher] 自动补充群组项目关键词: chat_id=%s, project='%s', 原始='%s', 增强='%s'\n",
		chatID, projectName, message, enhanced)

	return enhanced
}
