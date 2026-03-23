package handler

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// ExecutorHandler 执行机器人处理器
type ExecutorHandler struct {
	*BaseHandler

	allowedDispatchers map[string]bool
	allowedScripts     map[string]bool
}

// NewExecutorHandler 创建执行机器人处理器
func NewExecutorHandler() *ExecutorHandler {
	return &ExecutorHandler{
		BaseHandler:        NewBaseHandler(),
		allowedDispatchers: make(map[string]bool),
		allowedScripts:     make(map[string]bool),
	}
}

// SetAllowedDispatchers 设置允许的 dispatcher 列表
func (h *ExecutorHandler) SetAllowedDispatchers(dispatchers []string) {
	h.allowedDispatchers = make(map[string]bool)
	for _, d := range dispatchers {
		h.allowedDispatchers[d] = true
	}
}

// SetAllowedScripts 设置允许执行的脚本列表
func (h *ExecutorHandler) SetAllowedScripts(scripts []string) {
	h.allowedScripts = make(map[string]bool)
	for _, script := range scripts {
		h.allowedScripts[script] = true
	}
}

// Handle 处理消息
func (h *ExecutorHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	// 解析发送者 bot_id
	senderBotID, err := extractSenderBotID(event)
	if err != nil {
		return fmt.Errorf("解析发送者失败: %w", err)
	}

	// 校验是否来自允许的 dispatcher
	if !h.allowedDispatchers[senderBotID] {
		return fmt.Errorf("未授权的 dispatcher: %s", senderBotID)
	}

	// 解析消息内容
	message, err := extractMessageContent(event)
	if err != nil {
		return fmt.Errorf("解析消息失败: %w", err)
	}

	// 解析脚本路径
	scriptPath, args, err := h.parseScriptCommand(message)
	if err != nil {
		return fmt.Errorf("解析脚本命令失败: %w", err)
	}

	// 校验脚本是否在白名单
	if !h.allowedScripts[scriptPath] {
		return fmt.Errorf("脚本不在白名单中: %s", scriptPath)
	}

	// 执行脚本
	output, err := h.executeScript(scriptPath, args)
	if err != nil {
		return fmt.Errorf("执行脚本失败: %w", err)
	}

	// TODO: 向 dispatcher 汇报结果
	_ = output

	return nil
}

// parseScriptCommand 解析脚本命令
func (h *ExecutorHandler) parseScriptCommand(message string) (scriptPath string, args []string, err error) {
	// 简化实现：假设格式为 "execute <script> [args...]"
	parts := strings.Fields(message)
	if len(parts) < 2 || parts[0] != "execute" {
		return "", nil, fmt.Errorf("命令格式错误，应为: execute <script> [args...]")
	}

	scriptPath = parts[1]
	args = parts[2:]

	return scriptPath, args, nil
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

// extractSenderBotID 从事件中提取发送者 bot_id
func extractSenderBotID(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if sender, ok := eventMap["sender"].(map[string]interface{}); ok {
			if botID, ok := sender["bot_id"].(string); ok {
				return botID, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 sender.bot_id")
}

// extractMessageContent 从事件中提取消息内容
func extractMessageContent(event interface{}) (string, error) {
	if eventMap, ok := event.(map[string]interface{}); ok {
		if message, ok := eventMap["message"].(map[string]interface{}); ok {
			if content, ok := message["content"].(string); ok {
				return content, nil
			}
		}
	}
	return "", fmt.Errorf("无法提取 message.content")
}
