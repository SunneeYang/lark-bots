package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/config"
	"github.com/SunneeYang/lark-bots/internal/handler"
	"github.com/SunneeYang/lark-bots/internal/logger"
	"github.com/SunneeYang/lark-bots/internal/router"
	"github.com/spf13/cobra"
)

// UnifiedExecutorHandler 统一的 executor handler，根据 bot 名称路由到具体的 handler
type UnifiedExecutorHandler struct {
	handlers map[string]*handler.ExecutorHandler
}

// Handle 实现 MessageHandler 接口
func (h *UnifiedExecutorHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	// 根据 bot 名称找到对应的 handler
	executorHandler, exists := h.handlers[botClient.Name]
	if !exists {
		return fmt.Errorf("未找到 executor handler: %s", botClient.Name)
	}

	// 委托给具体的 handler
	return executorHandler.Handle(ctx, event, botClient)
}

var (
	cfgFile string
	bots    string
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

func initConfig() {
	// 配置已在 runStart 中手动加载
}

func runStart(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// 1. 加载配置
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("❌ 加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 验证配置
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ 配置加载成功")

	// 3. 创建机器人注册表
	registry := bot.NewBotRegistry()

	// 4. 注册所有机器人
	for _, botCfg := range cfg.Bots {
		botClient := bot.NewBotClient(botCfg.Name, botCfg.AppID, botCfg.AppSecret, botCfg.Role)

		// 为 executor 机器人设置额外的配置
		if botCfg.Role == "executor" {
			botClient.AllowedDispatchers = botCfg.AllowedDispatchers
			botClient.AllowedScripts = botCfg.AllowedScripts
		}

		if err := registry.Register(botClient); err != nil {
			fmt.Printf("❌ 注册机器人失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  - 注册机器人: %s (%s)\n", botCfg.Name, botCfg.Role)
	}

	// 5. 过滤要启动的机器人
	var activeBots []*bot.BotClient
	botList, _ := cmd.Flags().GetString("bots")
	startAll, _ := cmd.Flags().GetBool("all")

	if startAll || botList == "" {
		activeBots = registry.GetAll()
	} else {
		activeBots = registry.FilterByName(parseBotList(botList))
	}

	if len(activeBots) == 0 {
		fmt.Println("❌ 没有要启动的机器人")
		os.Exit(1)
	}

	fmt.Printf("✅ 将启动 %d 个机器人\n", len(activeBots))
	for _, b := range activeBots {
		fmt.Printf("  - %s (%s)\n", b.Name, b.Role)
	}

	// 6. 创建任务日志
	taskLogger := logger.NewTaskLogger()

	// 7. 创建路由器
	messageRouter := router.NewMessageRouter(registry)

	// 8. 创建并注册 handlers
	dispatcherHandler := handler.NewDispatcherHandler()
	dispatcherHandler.SetWhiteLists(cfg.UserWhiteList, cfg.TaskWhiteList)
	messageRouter.RegisterHandler("dispatcher", dispatcherHandler)

	// 为 executor 创建一个统一的 handler，内部根据 bot 名称分发到具体的执行逻辑
	// 创建 executor handler 映射
	executorHandlers := make(map[string]*handler.ExecutorHandler)
	for _, botCfg := range cfg.Bots {
		if botCfg.Role == "executor" {
			executorHandler := handler.NewExecutorHandler()
			executorHandler.SetAllowedDispatchers(botCfg.AllowedDispatchers)
			executorHandler.SetAllowedScripts(botCfg.AllowedScripts)
			executorHandlers[botCfg.Name] = executorHandler
		}
	}

	// 创建一个统一的 executor handler，根据 bot 名称路由到对应的 handler
	unifiedExecutorHandler := &UnifiedExecutorHandler{
		handlers: executorHandlers,
	}
	messageRouter.RegisterHandler("executor", unifiedExecutorHandler)

	fmt.Println("✅ 服务初始化完成")

	// 9. 启动事件监听（TODO: 实现飞书事件监听）
	fmt.Println("⏳ 等待飞书事件...")
	fmt.Println("提示: 按 Ctrl+C 退出")

	// 10. 等待中断信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	fmt.Println("\n👋 正在关闭服务...")
	_ = taskLogger
	_ = ctx
	_ = messageRouter
	fmt.Println("✅ 服务已关闭")
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

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
