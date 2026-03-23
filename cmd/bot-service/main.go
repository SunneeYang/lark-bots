package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/config"
	"github.com/SunneeYang/lark-bots/internal/handler"
	"github.com/SunneeYang/lark-bots/internal/logger"
	"github.com/SunneeYang/lark-bots/internal/router"
	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/larksuite/oapi-sdk-go/v3/ws"
	"github.com/spf13/cobra"
)

// 全局组件（供事件处理器使用）
var (
	globalRegistry    *bot.BotRegistry
	globalTaskLogger *logger.TaskLogger
	globalRouter     *router.MessageRouter
)

// UnifiedExecutorHandler 统一的 executor handler，根据 bot 名称路由到具体的 handler
type UnifiedExecutorHandler struct {
	handlers map[string]*handler.ExecutorHandler
}

// Handle 实现 MessageHandler 接口
func (h *UnifiedExecutorHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	executorHandler, exists := h.handlers[botClient.Name]
	if !exists {
		return fmt.Errorf("未找到 executor handler: %s", botClient.Name)
	}
	return executorHandler.Handle(ctx, event, botClient)
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

func init() {
	cobra.OnInitialize(initConfig)

	startCmd.Flags().StringVarP(&cfgFile, "config", "c", "configs/bots.yaml", "配置文件路径")
	startCmd.Flags().StringVarP(&bots, "bots", "b", "", "要启动的机器人列表（逗号分隔）")
	startCmd.Flags().Bool("all", false, "启动所有机器人")

	rootCmd.AddCommand(startCmd)
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
		if botCfg.Role == "executor" {
			botClient.AllowedDispatchers = botCfg.AllowedDispatchers
			botClient.AllowedScripts = botCfg.AllowedScripts
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

	dispatcherHandler := handler.NewDispatcherHandler()
	dispatcherHandler.SetWhiteLists(cfg.UserWhiteList, cfg.TaskWhiteList)
	globalRouter.RegisterHandler("dispatcher", dispatcherHandler)
	fmt.Println("   ✅ dispatcher 处理器注册成功")

	executorHandlers := make(map[string]*handler.ExecutorHandler)
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			executorHandler := handler.NewExecutorHandler()
			executorHandler.SetAllowedDispatchers(botCfg.AllowedDispatchers)
			executorHandler.SetAllowedScripts(botCfg.AllowedScripts)
			executorHandlers[botCfg.Name] = executorHandler
			fmt.Printf("   ✅ executor 处理器注册成功 (bot: %s)\n", botCfg.Name)
		}
	}

	unifiedExecutorHandler := &UnifiedExecutorHandler{handlers: executorHandlers}
	globalRouter.RegisterHandler("executor", unifiedExecutorHandler)

	// 6. 为每个机器人创建 WebSocket 客户端并启动
	fmt.Println("\n🔌 建立飞书 WebSocket 连接:")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	for _, botClient := range activeBots {
		go startBotWSClient(ctx, botClient)
		fmt.Printf("   🌐 %s 连接中...\n", botClient.Name)
	}

	fmt.Println("\n========================================")
	fmt.Println("       🎉 服务启动成功!")
	fmt.Println("========================================")
	fmt.Println("\n⏳ 正在监听飞书事件...")
	fmt.Println("提示: 按 Ctrl+C 退出")

	// 7. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 正在关闭服务...")
	cancel()
	time.Sleep(2 * time.Second) // 等待连接关闭
	fmt.Println("✅ 服务已关闭")
}

// startBotWSClient 为单个机器人启动 WebSocket 客户端
func startBotWSClient(ctx context.Context, botClient *bot.BotClient) {
	// 创建事件分发器
	evtDispatcher := dispatcher.NewEventDispatcher("", "")

	// 注册消息接收处理器
	evtDispatcher.OnP2MessageReceiveV1(func(c context.Context, event *larkim.P2MessageReceiveV1) error {
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

// handleMessageReceive 处理飞书消息接收事件
func handleMessageReceive(ctx context.Context, msgEvent *larkim.P2MessageReceiveV1, botClient *bot.BotClient) error {
	// 打印事件详情
	fmt.Printf("\n========================================\n")
	fmt.Printf("📨 收到飞书消息事件 [%s]\n", botClient.Name)
	fmt.Printf("========================================\n")

	// 获取 tenant_key
	tenantKey := ""
	if msgEvent.Event != nil && msgEvent.Event.Sender != nil && msgEvent.Event.Sender.TenantKey != nil {
		tenantKey = *msgEvent.Event.Sender.TenantKey
	}
	fmt.Printf("🏢 TenantKey: %s\n", tenantKey)
	fmt.Printf("📦 AppID: %s\n", botClient.AppID)

	// 获取消息详情
	if msgEvent.Event != nil && msgEvent.Event.Message != nil {
		msg := msgEvent.Event.Message

		messageID := getStringPtr(msg.MessageId)
		chatID := getStringPtr(msg.ChatId)
		chatType := getStringPtr(msg.ChatType)
		content := getStringPtr(msg.Content)
		msgType := getStringPtr(msg.MessageType)

		fmt.Printf("\n💬 消息详情:\n")
		fmt.Printf("   - 消息ID: %s\n", messageID)
		fmt.Printf("   - 会话ID: %s (%s)\n", chatID, chatType)
		fmt.Printf("   - 消息类型: %s\n", msgType)
		fmt.Printf("   - 内容: %s\n", truncateString(content, 200))

		// 获取发送者信息
		if msgEvent.Event.Sender != nil {
			sender := msgEvent.Event.Sender
			senderID := ""
			if sender.SenderId != nil {
				if sender.SenderId.OpenId != nil {
					senderID = *sender.SenderId.OpenId
				} else if sender.SenderId.UserId != nil {
					senderID = *sender.SenderId.UserId
				}
			}
			senderType := getStringPtr(sender.SenderType)
			fmt.Printf("   - 发送者: %s (%s)\n", senderID, senderType)

			// 创建任务记录
			taskRecord := &bot.TaskRecord{
				TaskName:  "message_receive",
				User:      senderID,
				StartTime: bot.Now(),
			}

			// 构建事件数据传递给路由
			handlerEvent := map[string]interface{}{
				"message_id": messageID,
				"chat_id":    chatID,
				"content":    content,
				"msg_type":   msgType,
				"sender":     senderID,
				"sender_type": senderType,
				"app_id":    botClient.AppID,
				"tenant_key": tenantKey,
			}

			fmt.Printf("   - 目标机器人: %s (%s)\n", botClient.Name, botClient.Role)
			taskRecord.Dispatcher = botClient.Name
			taskRecord.Executor = botClient.Name

			// 路由到对应的处理器
			if err := globalRouter.Route(ctx, handlerEvent); err != nil {
				fmt.Printf("❌ 路由消息失败: %v\n", err)
				taskRecord.Status = "failed"
				taskRecord.Error = err.Error()
			} else {
				fmt.Println("✅ 消息处理成功")
				taskRecord.Status = "completed"
			}

			// 记录任务
			globalTaskLogger.CreateTask(taskRecord)
		}
	}

	fmt.Printf("========================================\n\n")
	return nil
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
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
