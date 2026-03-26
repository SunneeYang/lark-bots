package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/config"
	"github.com/SunneeYang/lark-bots/internal/embedding"
	"github.com/SunneeYang/lark-bots/internal/handler"
	"github.com/SunneeYang/lark-bots/internal/handler/matcher"
	"github.com/SunneeYang/lark-bots/internal/logger"
	"github.com/SunneeYang/lark-bots/internal/router"
	larkevent "github.com/larksuite/oapi-sdk-go/v3/event"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/spf13/cobra"
)

// 全局组件（供事件处理器使用）
var (
	globalRegistry    *bot.BotRegistry
	globalTaskLogger *logger.TaskLogger
	globalRouter     *router.MessageRouter
)

// MessagePoller 消息轮询器接口
type MessagePoller interface {
	Start(ctx context.Context)
	Stop()
}

// UnifiedExecutorHandler 统一的 executor handler，根据 bot 名称路由到具体的 handler
type UnifiedExecutorHandler struct {
	handlers map[string]*handler.ExecutorHandler
}

// Handle 实现 MessageHandler 接口
func (h *UnifiedExecutorHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	fmt.Printf("📨 [%s] UnifiedExecutorHandler 收到事件\n", botClient.Name)
	executorHandler, exists := h.handlers[botClient.Name]
	if !exists {
		return fmt.Errorf("未找到 executor handler: %s (可用: %v)", botClient.Name, mapKeys(h.handlers))
	}
	return executorHandler.Handle(ctx, event, botClient)
}

func mapKeys(m map[string]*handler.ExecutorHandler) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

var (
	cfgFile    string
	bots       string
)

var rootCmd = &cobra.Command{
	Use:   "bot-service",
	Short: "飞书机器人服务",
	Long:  `多机器人协作服务，支持配置文件管理和任务分发`,
}

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "启动服务",
	Run:   runStart,
}

var fetchOpenIDCmd = &cobra.Command{
	Use:   "fetch-openid [bot names...]",
	Short: "获取指定机器人的 OpenID",
	Args:  cobra.MinimumNArgs(1),
	Run:   runFetchOpenID,
}

func init() {
	cobra.OnInitialize(initConfig)

	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "configs/bots.yaml", "配置文件路径")
	startCmd.Flags().StringVarP(&bots, "bots", "b", "", "要启动的机器人列表（逗号分隔）")
	startCmd.Flags().Bool("all", false, "启动所有机器人")

	fetchOpenIDCmd.Flags().StringP("config", "c", "configs/bots.yaml", "配置文件路径")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(fetchOpenIDCmd)
}

func initConfig() {}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runStart(cmd *cobra.Command, args []string) {
	fmt.Println("========================================")
	fmt.Println("       飞书机器人服务启动中...")
	fmt.Println("========================================")

	// 1. 加载配置
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 配置加载成功")

	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 配置验证通过")

	// 2. 创建机器人注册表
	globalRegistry = bot.NewBotRegistry()

	fmt.Println("\n📝 注册机器人:")
	for _, botCfg := range cfg.Bots {
		botClient := bot.NewBotClient(botCfg.Name, botCfg.AppID, botCfg.AppSecret, botCfg.Role)
		// 为每个机器人初始化飞书 SDK 客户端
		botClient.InitLarkClient()
		if botCfg.Role == "executor" {
			botClient.AllowedDispatchers = botCfg.AllowedDispatchers
		}
		if err := globalRegistry.Register(botClient); err != nil {
			fmt.Printf("❌ 注册机器人失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("   ✅ %s (角色: %s, AppID: %s)\n", botCfg.Name, botCfg.Role, botCfg.AppID)
	}

	// 3. 过滤要启动的机器人
	var activeBots []*bot.BotClient
	botList, _ := cmd.Flags().GetString("bots")
	startAll, _ := cmd.Flags().GetBool("all")

	if startAll || botList == "" {
		activeBots = globalRegistry.GetAll()
	} else {
		activeBots = globalRegistry.FilterByName(parseBotList(botList))
	}

	if len(activeBots) == 0 {
		fmt.Println("❌ 没有要启动的机器人")
		os.Exit(1)
	}

	fmt.Printf("\n🚀 将启动 %d 个机器人:\n", len(activeBots))
	for _, b := range activeBots {
		fmt.Printf("   - %s (%s)\n", b.Name, b.Role)
	}

	// 4. 初始化组件
	globalTaskLogger = logger.NewTaskLogger()
	fmt.Println("\n✅ 任务日志管理器初始化完成")

	globalRouter = router.NewMessageRouter(globalRegistry)
	fmt.Println("✅ 消息路由器初始化完成")

	// 5. 注册 handlers
	fmt.Println("\n📋 注册消息处理器:")

	// 配置 dispatcher 相关
	var dispatcherCfg *config.BotConfig
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "dispatcher" {
			dispatcherCfg = &botCfg
			break
		}
	}

	// 创建 dispatcher handler
	dispatcherHandler := handler.NewDispatcherHandler(nil)
	dispatcherHandler.SetRobotGroupID(cfg.RobotGroupID)
	globalRouter.RegisterHandler("dispatcher", dispatcherHandler)
	fmt.Println("   ✅ dispatcher 处理器注册成功")

	// 构建 executor 配置（用于分层匹配）
	executorConfigs := buildExecutorConfigs(cfg)
	if len(executorConfigs) > 0 {
		// 有新配置格式，启用分层匹配模式
		var semanticProvider embedding.Provider
		if dispatcherCfg != nil && dispatcherCfg.SemanticMatch != nil && dispatcherCfg.SemanticMatch.Enabled {
			semanticProvider = embedding.NewGLMProvider(dispatcherCfg.SemanticMatch.APIKey, dispatcherCfg.SemanticMatch.Model)
		}
		layeredMatcher := matcher.NewLayeredMatcher(executorConfigs, semanticProvider)
		dispatcherHandler.SetLayeredMatcher(layeredMatcher)
		fmt.Printf("   ✅ 分层匹配模式已启用，共 %d 个任务配置\n", len(executorConfigs))
	} else {
		// 旧配置格式，使用任务白名单模式
		allTaskNames := make([]string, 0)
		for _, botCfg := range cfg.Bots {
			if botCfg.Role == "executor" {
				for taskName := range botCfg.TaskScripts {
					allTaskNames = append(allTaskNames, taskName)
				}
			}
		}
		dispatcherHandler.SetAllowedTasks(allTaskNames)
		fmt.Printf("   ✅ 传统匹配模式已启用，共 %d 个任务\n", len(allTaskNames))
	}

	// 设置 dispatcher 的用户白名单
	if dispatcherCfg != nil {
		fmt.Printf("📋 加载 dispatcher 配置: allowed_users=%v\n", dispatcherCfg.AllowedUsers)
		dispatcherHandler.SetAllowedUsers(dispatcherCfg.AllowedUsers)
		// 设置群组项目映射
		if dispatcherCfg.GroupProjectMap != nil && len(dispatcherCfg.GroupProjectMap) > 0 {
			dispatcherHandler.SetGroupProjectMap(dispatcherCfg.GroupProjectMap)
			fmt.Printf("📋 加载群组项目映射配置: %d 个群组\n", len(dispatcherCfg.GroupProjectMap))
		}
	} else {
		fmt.Println("⚠️ 未找到 dispatcher 配置")
	}

	executorHandlers := make(map[string]*handler.ExecutorHandler)
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			executorHandler := handler.NewExecutorHandler()
			executorHandler.SetRobotGroupID(cfg.RobotGroupID)
			executorHandler.SetAllowedDispatchers(botCfg.AllowedDispatchers)
			executorHandler.SetTasks(botCfg.Tasks)
			executorHandlers[botCfg.Name] = executorHandler
			fmt.Printf("   ✅ executor 处理器注册成功 (bot: %s)\n", botCfg.Name)
		}
	}

	unifiedExecutorHandler := &UnifiedExecutorHandler{handlers: executorHandlers}
	globalRouter.RegisterHandler("executor", unifiedExecutorHandler)

	// 6. 启动 executor 消息轮询器
	fmt.Println("\n🔄 启动消息轮询器:")
	var pollers []MessagePoller
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			// 获取对应的 ExecutorHandler
			executorHandler := executorHandlers[botCfg.Name]
			if executorHandler == nil {
				fmt.Printf("   ⚠️  %s 未找到对应的 handler，跳过\n", botCfg.Name)
				continue
			}

			// 获取 executor 的 BotClient
			var executorBot *bot.BotClient
			for _, bc := range activeBots {
				if bc.Name == botCfg.Name {
					executorBot = bc
					break
				}
			}
			if executorBot == nil {
				continue
			}

			// 将 []string 转换为 map[string]bool
			dispatchersMap := make(map[string]bool)
			for _, d := range botCfg.AllowedDispatchers {
				dispatchersMap[d] = true
			}

			// 构建任务名称到脚本的映射（用于轮询器兼容）
			taskNameToScript := make(map[string]string)
			for taskName, taskDetail := range botCfg.Tasks {
				taskNameToScript[taskName] = taskDetail.Script
			}

			// 创建轮询器
			pollInterval := config.ParsePollInterval(botCfg.PollInterval)
			maxTasks := botCfg.MaxTasks // 从配置读取最大并发任务数，0 或负数表示不限制
			poller := handler.NewMessagePoller(
				executorBot,
				cfg.RobotGroupID,
				dispatchersMap,
				executorHandler,  // 传递 ExecutorHandler 引用
				taskNameToScript,
				botCfg.Tasks,      // 传递完整 Tasks 配置（包含 params）
				pollInterval,
				maxTasks,
			)
			pollers = append(pollers, poller)
			fmt.Printf("   ✅ %s 轮询器已创建 (tasks: %d)\n", botCfg.Name, len(taskNameToScript))
		}
	}

	// 7. 为每个机器人创建 WebSocket 客户端并启动
	fmt.Println("\n🔌 建立飞书 WebSocket 连接:")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 先启动 WebSocket 连接
	for _, botClient := range activeBots {
		go startBotWSClient(ctx, botClient)
		fmt.Printf("   🌐 %s WebSocket 连接中...\n", botClient.Name)
	}

	// 等待 WebSocket 连接建立（SDK 的 Start 是阻塞的，给一点时间让连接建立）
	time.Sleep(2 * time.Second)
	fmt.Println("   ✅ 所有 WebSocket 连接已建立")

	// 启动所有轮询器（WebSocket 连接建立之后）
	for _, poller := range pollers {
		poller.Start(ctx)
	}

	fmt.Println("\n========================================")
	fmt.Println("       🎉 服务启动成功!")
	fmt.Println("========================================")
	fmt.Println("\n⏳ 正在监听飞书事件...")
	fmt.Println("提示: 按 Ctrl+C 退出")

	// 8. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 正在关闭服务...")
	cancel()

	// 停止所有轮询器
	for _, poller := range pollers {
		poller.Stop()
	}

	time.Sleep(2 * time.Second) // 等待连接关闭
	fmt.Println("✅ 服务已关闭")
}

// startBotWSClient 为单个机器人启动 WebSocket 客户端
func startBotWSClient(ctx context.Context, botClient *bot.BotClient) {
	// 创建事件分发器
	evtDispatcher := dispatcher.NewEventDispatcher("", "")

	// 统一处理 im.message.receive_v1 事件（同时覆盖 P1 和 P2）
	// 根据 payload 结构判断是 P1（私聊/群聊普通消息）还是 P2（@机器人消息）
	evtDispatcher.OnCustomizedEvent("im.message.receive_v1", func(c context.Context, event *larkevent.EventReq) error {
		return handleMessageReceive(c, event, botClient)
	})

	// 创建 WebSocket 客户端
	wsClient := ws.NewClient(
		botClient.AppID,
		botClient.AppSecret,
		ws.WithEventHandler(evtDispatcher),
	)

	fmt.Printf("[%s] 🔗 WebSocket 连接已建立\n", botClient.Name)

	// 启动客户端（阻塞，直到连接断开）
	if err := wsClient.Start(ctx); err != nil {
		fmt.Printf("[%s] ❌ WebSocket 连接失败: %v\n", botClient.Name, err)
	}
}

// handleMessageReceive 处理飞书消息接收事件（统一处理 P1 和 P2 格式）
// P1: 私聊/群聊普通消息；P2: @机器人消息
func handleMessageReceive(ctx context.Context, req *larkevent.EventReq, botClient *bot.BotClient) error {
	// 打印事件详情
	fmt.Printf("\n========================================\n")
	fmt.Printf("📨 收到飞书消息事件 [%s]\n", botClient.Name)
	fmt.Printf("========================================\n")

	tenantKey := ""
	if h := req.Header["Tenant-Key"]; len(h) > 0 {
		tenantKey = h[0]
	}
	fmt.Printf("🏢 TenantKey: %s\n", tenantKey)
	fmt.Printf("📦 AppID (接收者): %s\n", botClient.AppID)

	// 解析原始 JSON，判断是 P1 还是 P2 格式
	var raw map[string]interface{}
	if err := json.Unmarshal(req.Body, &raw); err != nil {
		return fmt.Errorf("解析事件 JSON 失败: %w", err)
	}

	// 打印原始事件的 sender 和 app_id 结构（用于调试）
	if sender, ok := raw["event"].(map[string]interface{})["sender"].(map[string]interface{}); ok {
		fmt.Printf("🔍 原始事件 sender: %+v\n", sender)
	}
	if appID, ok := raw["app_id"].(string); ok {
		fmt.Printf("🔍 原始事件 app_id (发送者): %s\n", appID)
	}

	event, ok := raw["event"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("事件中无 event 字段")
	}

	var senderID, messageID, chatID, chatType, msgType, content string
	var senderBotID string // 发送者的 bot_id (app_id)

	// P2 格式: event.sender + event.message（@机器人消息）
	if sender, ok := event["sender"].(map[string]interface{}); ok {
		if senderIDMap, ok := sender["sender_id"].(map[string]interface{}); ok {
			if openID, ok := senderIDMap["open_id"].(string); ok {
				senderID = openID
			}
		}
		// 提取发送者的 bot_id（用于识别消息来自哪个应用）
		if botID, ok := sender["bot_id"].(string); ok {
			senderBotID = botID
		}
		if msg, ok := event["message"].(map[string]interface{}); ok {
			messageID, _ = msg["message_id"].(string)
			chatID, _ = msg["chat_id"].(string)
			chatType, _ = msg["chat_type"].(string)
			msgType, _ = msg["message_type"].(string)
			if c, ok := msg["content"].(string); ok {
				content = c
			}
		}
	} else {
		// P1 格式: event.open_id + event.open_chat_id（所有消息）
		senderID, _ = event["open_id"].(string)
		messageID, _ = event["open_message_id"].(string)
		chatID, _ = event["open_chat_id"].(string)
		chatType, _ = event["chat_type"].(string)
		msgType, _ = event["msg_type"].(string)
		if text, ok := event["text"].(string); ok {
			content = fmt.Sprintf(`{"text":"%s"}`, text)
		}
		// P1 格式的 app_id 可能位于事件顶层
		if appID, ok := raw["app_id"].(string); ok {
			senderBotID = appID
		}
	}

	// 优先使用 sender.bot_id，如果为空则尝试 raw.app_id
	if senderBotID == "" {
		if appID, ok := raw["app_id"].(string); ok {
			senderBotID = appID
		}
	}

	fmt.Printf("\n💬 消息详情:\n")
	fmt.Printf("   - 消息ID: %s\n", messageID)
	fmt.Printf("   - 会话ID: %s (%s)\n", chatID, chatType)
	fmt.Printf("   - 消息类型: %s\n", msgType)
	fmt.Printf("   - 内容: %s\n", truncateString(content, 200))
	fmt.Printf("   - 发送者 open_id: %s\n", senderID)
	fmt.Printf("   - 发送者 app_id: %s\n", senderBotID)

	// 构建事件数据传递给路由
	handlerEvent := map[string]interface{}{
		"message": map[string]interface{}{
			"message_id": messageID,
			"chat_id":    chatID,
			"chat_type":  chatType,
			"content":    content,
			"msg_type":   msgType,
		},
		"sender": map[string]interface{}{
			"sender_id": map[string]interface{}{
				"open_id": senderID,
			},
			"bot_id": senderBotID,
		},
		"app_id":     senderBotID, // 使用发送者的 app_id，而不是接收者的
		"tenant_key": tenantKey,
	}

	// 创建任务记录
	taskRecord := &bot.TaskRecord{
		TaskName:  "message_receive",
		User:      senderID,
		StartTime: bot.Now(),
	}

	fmt.Printf("   - 目标机器人: %s (%s)\n", botClient.Name, botClient.Role)
	taskRecord.Dispatcher = botClient.Name
	taskRecord.Executor = botClient.Name

	// 路由到对应的处理器
	if err := globalRouter.Route(ctx, handlerEvent, botClient); err != nil {
		fmt.Printf("❌ 路由消息失败: %v\n", err)
		taskRecord.Status = "failed"
		taskRecord.Error = err.Error()
	} else {
		fmt.Println("✅ 消息处理成功")
		taskRecord.Status = "completed"
	}

	// 记录任务
	globalTaskLogger.CreateTask(taskRecord)

	fmt.Printf("========================================\n\n")
	return nil
}

func parseBotList(botList string) []string {
	var result []string
	for _, s := range strings.Split(botList, ",") {
		trimmed := strings.TrimSpace(s)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// botInfoResponse 飞书 bot info API 响应
type botInfoResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Bot  struct {
		AppID          string `json:"app_id"`
		AppName        string `json:"app_name"`
		OpenID         string `json:"open_id"`
		BotName        string `json:"bot_name"`
		ActivateStatus int    `json:"activate_status"`
	} `json:"bot"`
}

func runFetchOpenID(cmd *cobra.Command, args []string) {
	fmt.Println("========================================")
	fmt.Println("       获取机器人 OpenID")
	fmt.Println("========================================")

	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	targetNames := make(map[string]bool)
	for _, name := range args {
		targetNames[name] = true
	}

	var bots []config.BotConfig
	for _, b := range cfg.Bots {
		if targetNames[b.Name] {
			bots = append(bots, b)
		}
	}

	if len(bots) == 0 {
		fmt.Printf("❌ 未找到匹配的机器人: %v\n", args)
		os.Exit(1)
	}

	fmt.Printf("\n📋 正在获取 %d 个机器人的 OpenID:\n\n", len(bots))

	for _, botCfg := range bots {
		openID, err := fetchBotOpenID(botCfg.AppID, botCfg.AppSecret)
		if err != nil {
			fmt.Printf("  ❌ %s (%s): 获取失败 - %v\n", botCfg.Name, botCfg.AppID, err)
		} else {
			fmt.Printf("  ✅ %s: %s\n", botCfg.Name, openID)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("请将获取的 OpenID 填入配置文件的 open_id 字段")
	fmt.Println("========================================")
}

type tokenResp struct {
	Code              int    `json:"code"`
	TenantAccessToken string `json:"tenant_access_token"`
}

func fetchBotOpenID(appID, appSecret string) (string, error) {
	// 获取 tenant_access_token
	tokenReqBody := map[string]string{"app_id": appID, "app_secret": appSecret}
	tokenReqJSON, _ := json.Marshal(tokenReqBody)
	req, _ := http.NewRequest("POST", "https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		strings.NewReader(string(tokenReqJSON)))
	req.Header.Set("Content-Type", "application/json")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求 token 失败: %w", err)
	}
	defer resp.Body.Close()

	var token tokenResp
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
	if err := json.Unmarshal(body, &token); err != nil {
		return "", fmt.Errorf("解析 token 响应失败: %w, body: %s", err, string(body))
	}
	if token.Code != 0 {
		return "", fmt.Errorf("token API 错误: code=%d", token.Code)
	}

	// 调用 bot info API
	botReq, _ := http.NewRequest("GET", "https://open.feishu.cn/open-apis/bot/v3/info", nil)
	botReq.Header.Set("Authorization", "Bearer "+token.TenantAccessToken)

	botResp, err := httpClient.Do(botReq)
	if err != nil {
		return "", fmt.Errorf("请求 bot info 失败: %w", err)
	}
	defer botResp.Body.Close()

	botBody, _ := io.ReadAll(io.LimitReader(botResp.Body, 2048))

	var botInfo botInfoResponse
	if err := json.Unmarshal(botBody, &botInfo); err != nil {
		return "", fmt.Errorf("解析 bot info 响应失败: %w", err)
	}

	if botInfo.Code != 0 {
		return "", fmt.Errorf("bot info API 错误: code=%d, msg=%s", botInfo.Code, botInfo.Msg)
	}

	if botInfo.Bot.OpenID == "" {
		return "", fmt.Errorf("bot open_id 为空，请确认机器人在飞书开放平台已启用机器人功能")
	}

	return botInfo.Bot.OpenID, nil
}

// buildExecutorConfigs 从配置构建 executor 任务配置（用于分层匹配）
func buildExecutorConfigs(cfg *config.ServiceConfig) []matcher.ExecutorTaskConfig {
	var configs []matcher.ExecutorTaskConfig

	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			// 新格式：Tasks 配置（参数化任务，包含语义匹配信息）
			for taskName, taskDetail := range botCfg.Tasks {
				// 跳过没有 names 的任务（需要用于精确匹配）
				if len(taskDetail.Names) == 0 {
					continue
				}
				configs = append(configs, matcher.ExecutorTaskConfig{
					ExecutorID:      botCfg.AppID,
					ExecutorName:    botCfg.Name,
					Description:     botCfg.Description,
					RoutingKeywords: botCfg.RoutingKeywords,
					Keywords:        taskDetail.Keywords,
					TaskName:        taskName,
					DisplayName:     taskDetail.DisplayName,
					Script:          taskDetail.Script,
					Names: matcher.TaskNames{
						Primary: taskDetail.Names[0], // 主操作名
						Aliases: taskDetail.Names,    // 所有名称（含主名）
					},
				})
			}
		}
	}

	return configs
}
