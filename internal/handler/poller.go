package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
)

// MessagePoller 消息轮询器，定期从飞书 API 获取群消息
type MessagePoller struct {
	botClient       *bot.BotClient
	robotGroupID    string
	pollInterval    time.Duration
	lastRequestTime int64 // 上次请求时间（毫秒时间戳）

	allowedDispatchers map[string]bool
	taskScripts       map[string]string
	sender           *common.Sender
	stopCh           chan struct{}
	wg               sync.WaitGroup
}

// NewMessagePoller 创建消息轮询器
func NewMessagePoller(botClient *bot.BotClient, robotGroupID string, allowedDispatchers map[string]bool, taskScripts map[string]string, pollInterval time.Duration) *MessagePoller {
	sender := common.NewSender(botClient.LarkClient)
	sender.SetRobotGroupID(robotGroupID)

	return &MessagePoller{
		botClient:         botClient,
		robotGroupID:      robotGroupID,
		pollInterval:      pollInterval,
		allowedDispatchers: allowedDispatchers,
		taskScripts:      taskScripts,
		sender:           sender,
		stopCh:           make(chan struct{}),
	}
}

// Start 开始轮询
func (p *MessagePoller) Start(ctx context.Context) {
	p.wg.Add(1)
	go p.pollLoop(ctx)
	fmt.Printf("[%s] 消息轮询器已启动，间隔: %v\n", p.botClient.Name, p.pollInterval)
}

// Stop 停止轮询
func (p *MessagePoller) Stop() {
	close(p.stopCh)
	p.wg.Wait()
	fmt.Printf("[%s] 消息轮询器已停止\n", p.botClient.Name)
}

// pollLoop 轮询循环
func (p *MessagePoller) pollLoop(ctx context.Context) {
	defer p.wg.Done()

	ticker := time.NewTicker(p.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.fetchMessages(ctx)
		}
	}
}

// fetchMessages 获取消息列表
func (p *MessagePoller) fetchMessages(ctx context.Context) {
	// 如果没有上次请求时间，设置为当前时间的前 1 分钟
	startTime := p.lastRequestTime
	if startTime == 0 {
		startTime = time.Now().Add(-1 * time.Minute).UnixMilli()
	}

	// 构建请求
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("chat").
		ContainerId(p.robotGroupID).
		StartTime(fmt.Sprintf("%d", startTime/1000)). // API 需要秒时间戳
		PageSize(50).
		SortType(larkim.SortTypeListMessageByCreateTimeDesc).
		Build()

	// 调用 SDK 获取消息列表
	resp, err := p.botClient.LarkClient.Im.Message.List(ctx, req)
	if err != nil {
		fmt.Printf("[%s] 获取消息列表失败: %v\n", p.botClient.Name, err)
		return
	}

	if !resp.Success() {
		fmt.Printf("[%s] 获取消息列表失败: code=%d, msg=%s\n", p.botClient.Name, resp.Code, resp.Msg)
		return
	}

	// 更新请求时间
	p.lastRequestTime = time.Now().UnixMilli()

	// 处理消息
	if resp.Data != nil && resp.Data.Items != nil {
		p.processMessages(ctx, resp.Data.Items)
	}
}

// processMessages 处理消息列表
func (p *MessagePoller) processMessages(ctx context.Context, items []*larkim.Message) {
	for _, msg := range items {
		// 跳过系统消息
		if msg.MsgType != nil && *msg.MsgType == "system" {
			continue
		}

		// 解析 sender
		if msg.Sender == nil {
			continue
		}

		// 检查是否是 app（机器人）发送的消息
		if msg.Sender.SenderType == nil || *msg.Sender.SenderType != "app" {
			continue
		}

		// 检查 sender id 是否是允许的 dispatcher
		if msg.Sender.Id == nil || *msg.Sender.Id == "" {
			continue
		}
		senderAppID := *msg.Sender.Id

		if !p.allowedDispatchers[senderAppID] {
			continue
		}

		// 解析消息内容
		if msg.Body == nil || msg.Body.Content == nil {
			continue
		}

		content := *msg.Body.Content
		taskName, err := parseTaskContent(content)
		if err != nil {
			continue
		}

		// 检查任务是否在白名单中
		if _, ok := p.taskScripts[taskName]; !ok {
			continue
		}

		// 执行任务
		fmt.Printf("[%s] 轮询发现任务: sender=%s, task=%s\n", p.botClient.Name, senderAppID, taskName)
		if err := p.executeTask(ctx, taskName); err != nil {
			fmt.Printf("[%s] 执行任务失败: %v\n", p.botClient.Name, err)
		}
	}
}

// parseTaskContent 解析任务内容，提取任务名
func parseTaskContent(content string) (string, error) {
	// content 格式是 JSON: {"text":"task_name"}
	var data struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", err
	}
	return data.Text, nil
}

// executeTask 执行任务并发送结果到群
func (p *MessagePoller) executeTask(ctx context.Context, taskName string) error {
	scriptPath, ok := p.taskScripts[taskName]
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskName)
	}

	// 执行脚本
	cmd := exec.Command(scriptPath)
	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// 构建结果消息
	resultMsg := fmt.Sprintf("[%s] 任务 [%s] 执行%s", p.botClient.Name, taskName, map[bool]string{true: "成功", false: "失败"}[err == nil])
	if err != nil {
		resultMsg += fmt.Sprintf(": %v", err)
	} else {
		// 截断输出
		outputLines := strings.Split(strings.TrimSpace(outputStr), "\n")
		if len(outputLines) > 5 {
			outputStr = strings.Join(outputLines[:5], "\n") + "\n...(输出已截断)"
		}
		resultMsg += fmt.Sprintf("\n输出:\n%s", strings.TrimSpace(outputStr))
	}

	fmt.Printf("📤 [%s] 汇报结果: %s\n", p.botClient.Name, taskName)

	// 发送结果到机器人群
	if err := p.sender.SendToGroup(resultMsg); err != nil {
		fmt.Printf("❌ [%s] 汇报结果失败: %v\n", p.botClient.Name, err)
		return err
	}

	return nil
}
