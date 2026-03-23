package handler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
)

// ExecutorHandler 执行机器人处理器
type ExecutorHandler struct {
	*BaseHandler

	allowedDispatchers  map[string]bool
	taskScripts        map[string]string // 任务名 -> 脚本路径
	dispatcherOpenID   string          // Dispatcher 的 open_id，用于 @ 汇报结果
}

// NewExecutorHandler 创建执行机器人处理器
func NewExecutorHandler() *ExecutorHandler {
	return &ExecutorHandler{
		BaseHandler:        NewBaseHandler(),
		allowedDispatchers: make(map[string]bool),
		taskScripts:        make(map[string]string),
	}
}

// SetAllowedDispatchers 设置允许的 dispatcher 列表
func (h *ExecutorHandler) SetAllowedDispatchers(dispatchers []string) {
	h.allowedDispatchers = make(map[string]bool)
	for _, d := range dispatchers {
		h.allowedDispatchers[d] = true
	}
}

// SetTaskScripts 设置任务脚本映射 (任务名 -> 脚本路径)
func (h *ExecutorHandler) SetTaskScripts(scripts map[string]string) {
	h.taskScripts = scripts
}

// SetDispatcherOpenID 设置 Dispatcher 的 open_id（用于 @ 汇报结果）
func (h *ExecutorHandler) SetDispatcherOpenID(openID string) {
	h.dispatcherOpenID = openID
}

// Handle 处理消息
func (h *ExecutorHandler) Handle(_ context.Context, event interface{}, botClient *bot.BotClient) error {
	// 解析发送者 bot_id
	senderBotID, err := common.ExtractSenderBotID(event)
	if err != nil {
		senderBotID = "解析失败-" + err.Error()
	}

	// 解析消息内容
	rawContent, _ := common.ExtractMessageContent(event)
	message, _ := common.ParseMessageContent(rawContent)

	// 调试：打印完整事件sender结构
	fmt.Printf("📨 [%s] 收到消息: senderBotID=%s, content=%s\n", botClient.Name, senderBotID, message)

	// 校验是否来自允许的 dispatcher
	if !h.allowedDispatchers[senderBotID] {
		return fmt.Errorf("未授权的 dispatcher: %s", senderBotID)
	}

	// 解析任务名
	taskName, err := h.parseTaskName(message)
	if err != nil {
		return fmt.Errorf("解析任务名失败: %w", err)
	}

	// 转换任务名到脚本路径
	scriptPath, ok := h.taskScripts[taskName]
	if !ok {
		return fmt.Errorf("任务 %s 未配置", taskName)
	}

	// 执行脚本
	output, err := h.executeScript(scriptPath, nil)

	// 向 dispatcher 汇报结果
	resultMsg := fmt.Sprintf("任务 [%s] 执行%s", taskName, map[bool]string{true: "成功", false: "失败"}[err == nil])
	if err != nil {
		resultMsg += fmt.Sprintf(": %v", err)
	} else {
		// 截断输出
		outputLines := strings.Split(strings.TrimSpace(output), "\n")
		if len(outputLines) > 5 {
			output = strings.Join(outputLines[:5], "\n") + "\n...(输出已截断)"
		}
		resultMsg += fmt.Sprintf("\n输出:\n%s", strings.TrimSpace(output))
	}

	fmt.Printf("📤 [%s] 汇报结果: @%s %s\n", botClient.Name, h.dispatcherOpenID, taskName)
	if err := h.SendAtToGroup(h.dispatcherOpenID, resultMsg, botClient); err != nil {
		fmt.Printf("❌ [%s] 汇报结果失败: %v\n", botClient.Name, err)
	}

	return nil
}

// parseTaskName 解析任务名（格式: @开发服专员 check_logs）
func (h *ExecutorHandler) parseTaskName(message string) (string, error) {
	message = strings.TrimSpace(message)

	// 移除 @提及部分（如果有）
	if strings.HasPrefix(message, "@") {
		parts := strings.Fields(message)
		if len(parts) < 2 {
			return "", fmt.Errorf("消息格式错误")
		}
		return parts[1], nil
	}

	return message, nil
}

// executeScript 执行脚本
func (h *ExecutorHandler) executeScript(scriptPath string, args []string) (string, error) {
	cmd := exec.Command(scriptPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("脚本执行失败: %w", err)
	}
	return string(output), nil
}

