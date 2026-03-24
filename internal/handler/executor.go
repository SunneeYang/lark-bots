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

	allowedDispatchers map[string]bool
	taskScripts        map[string]string // 任务名 -> 脚本路径
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

	// 转换任务名到脚本路径（完整匹配）
	taskName := strings.TrimSpace(message)
	if taskName == "" {
		return nil // 空消息，静默忽略
	}
	scriptPath, ok := h.taskScripts[taskName]
	if !ok {
		return nil // 任务不在自己的任务列表中，静默忽略
	}

	// 执行脚本
	output, err := h.executeScript(scriptPath, nil)

	// 向机器人群汇报结果（纯文本）
	resultMsg := fmt.Sprintf("[%s] 任务 [%s] 执行%s", botClient.Name, taskName, map[bool]string{true: "成功", false: "失败"}[err == nil])
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

	fmt.Printf("📤 [%s] 汇报结果: %s\n", botClient.Name, taskName)
	if err := h.SendToGroup(resultMsg, botClient); err != nil {
		fmt.Printf("❌ [%s] 汇报结果失败: %v\n", botClient.Name, err)
	}

	return nil
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

