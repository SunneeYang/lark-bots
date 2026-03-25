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
	taskScripts        map[string]string // 旧格式：任务名 -> 脚本路径（向后兼容）
	taskNameToScript   map[string]string // 新格式：任务名称 -> 脚本路径
}

// NewExecutorHandler 创建执行机器人处理器
func NewExecutorHandler() *ExecutorHandler {
	return &ExecutorHandler{
		BaseHandler:        NewBaseHandler(),
		allowedDispatchers: make(map[string]bool),
		taskScripts:        make(map[string]string),
		taskNameToScript:   make(map[string]string),
	}
}

// SetAllowedDispatchers 设置允许的 dispatcher 列表
func (h *ExecutorHandler) SetAllowedDispatchers(dispatchers []string) {
	h.allowedDispatchers = make(map[string]bool)
	for _, d := range dispatchers {
		h.allowedDispatchers[d] = true
	}
}

// SetTaskScripts 设置任务脚本映射 (任务名 -> 脚本路径) - 旧格式
func (h *ExecutorHandler) SetTaskScripts(scripts map[string]string) {
	h.taskScripts = scripts
}

// SetTaskNameMapping 设置任务名称到脚本的映射 - 新格式
func (h *ExecutorHandler) SetTaskNameMapping(mapping map[string]string) {
	h.taskNameToScript = mapping
}

// Handle 处理消息（支持旧格式纯文本任务名和新格式 TaskCommand）
func (h *ExecutorHandler) Handle(_ context.Context, event interface{}, botClient *bot.BotClient) error {
	// executor 使用轮询机制获取任务，不处理 WebSocket 事件
	// 过滤：只处理来自机器人的消息，忽略用户消息
	if !common.IsFromApp(event) {
		return nil // 静默忽略用户消息
	}

	// 解析消息内容
	rawContent, _ := common.ExtractMessageContent(event)
	message, _ := common.ParseMessageContent(rawContent)

	// 通过 app_id 校验是否来自允许的 dispatcher
	senderAppID, err := common.ExtractSenderAppID(event)
	if err != nil {
		senderAppID = "解析失败-" + err.Error()
	}

	// 校验是否来自允许的 dispatcher（按 app_id）
	if !h.allowedDispatchers[senderAppID] {
		return fmt.Errorf("未授权的 dispatcher: %s", senderAppID)
	}

	// 解析任务（支持新旧两种格式）
	var scriptPath string
	var taskName string

	// 尝试解析为 TaskCommand 格式
	taskCmd, err := ParseTaskCommand(message)
	if err == nil {
		// 新格式：TaskCommand，只包含任务名称
		taskName = taskCmd.TaskName

		// 在自己的任务映射表中查找脚本
		var ok bool
		scriptPath, ok = h.taskNameToScript[taskName]
		if !ok {
			// 未找到，静默忽略（可能是发给其他 executor 的）
			return nil
		}
	} else {
		// 旧格式：纯文本任务名，从 taskScripts 查找
		taskName = strings.TrimSpace(message)
		if taskName == "" {
			return nil // 空消息，静默忽略
		}
		var ok bool
		scriptPath, ok = h.taskScripts[taskName]
		if !ok {
			return nil // 任务不在自己的任务列表中，静默忽略
		}
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

