package handler

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
	"github.com/SunneeYang/lark-bots/internal/config"
)

// ExecutorHandler 执行机器人处理器
type ExecutorHandler struct {
	*BaseHandler

	allowedDispatchers map[string]bool
	tasks              map[string]config.TaskDetail // 任务名 → 任务详细配置
}

// NewExecutorHandler 创建执行机器人处理器
func NewExecutorHandler() *ExecutorHandler {
	return &ExecutorHandler{
		BaseHandler:        NewBaseHandler(),
		allowedDispatchers: make(map[string]bool),
		tasks:              make(map[string]config.TaskDetail),
	}
}

// SetAllowedDispatchers 设置允许的 dispatcher 列表
func (h *ExecutorHandler) SetAllowedDispatchers(dispatchers []string) {
	h.allowedDispatchers = make(map[string]bool)
	for _, d := range dispatchers {
		h.allowedDispatchers[d] = true
	}
}

// SetTasks 设置任务配置
func (h *ExecutorHandler) SetTasks(tasks map[string]config.TaskDetail) {
	h.tasks = tasks
}

// Handle 处理消息（支持新格式 TaskCommand）
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

	// 解析 JSON 格式的任务命令
	taskCmd, err := ParseTaskCommandJSON(message)
	if err != nil {
		return fmt.Errorf("解析任务命令失败: %w", err)
	}

	fmt.Printf("📋 [%s] 解析任务: taskName=%s, requester=%s\n", botClient.Name, taskCmd.TaskName, taskCmd.Requester)

	// 执行任务
	return h.executeTask(taskCmd, botClient)
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

// executeTask 执行任务并汇报结果
func (h *ExecutorHandler) executeTask(taskCmd *TaskCommand, botClient *bot.BotClient) error {
	// 1. 提取任务名和发布者
	taskName := taskCmd.TaskName
	requester := taskCmd.Requester

	// 2. 从配置中查找任务详细配置
	taskDetail, ok := h.tasks[taskName]
	if !ok {
		// 任务不存在，静默忽略（可能是发给其他 executor 的）
		fmt.Printf("⚠️  [%s] 任务不存在: %s (可用任务: %v)\n", botClient.Name, taskName, mapKeys(h.tasks))
		return nil
	}

	fmt.Printf("✅ [%s] 找到任务配置: script=%s, params=%v\n", botClient.Name, taskDetail.Script, taskDetail.Params)

	// 3. 执行脚本
	output, err := h.executeScript(taskDetail.Script, taskDetail.Params)

	// 4. 构建汇报消息
	resultMsg := fmt.Sprintf("[%s] [%s] 任务 [%s] 执行成功", requester, botClient.Name, taskName)
	if err != nil {
		resultMsg = fmt.Sprintf("[%s] [%s] 任务 [%s] 执行失败: %v", requester, botClient.Name, taskName, err)
	} else {
		// 成功时追加输出（截断到 5 行）
		outputLines := fmt.Sprintf("%s", output)
		truncatedOutput := truncateOutput(outputLines, 5)
		resultMsg += fmt.Sprintf("\n输出:\n%s", truncatedOutput)
	}

	// 5. 发送汇报消息到机器人群
	return h.SendToGroup(resultMsg, botClient)
}

// truncateOutput 截断输出到指定行数
func truncateOutput(output string, maxLines int) string {
	lines := splitLines(output)
	if len(lines) <= maxLines {
		return output
	}
	return joinLines(lines[:maxLines]) + "\n...(输出已截断)"
}

// splitLines 按行分割字符串
func splitLines(s string) []string {
	if s == "" {
		return []string{}
	}
	lines := make([]string, 0)
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

// joinLines 将行数组合并为字符串
func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	result := lines[0]
	for i := 1; i < len(lines); i++ {
		result += "\n" + lines[i]
	}
	return result
}


func mapKeys(m map[string]config.TaskDetail) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
