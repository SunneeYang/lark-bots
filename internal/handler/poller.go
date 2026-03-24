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

	// 已处理消息 ID 去重（使用 sync.Map 支持并发安全读写）
	processedMsgs     sync.Map
	dedupWindow      time.Duration // 去重时间窗口
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
		dedupWindow:      2 * time.Minute, // 去重时间窗口，保留最近 2 分钟的消息 ID
	}
}

// Start 开始轮询
func (p *MessagePoller) Start(ctx context.Context) {
	p.wg.Add(2) // 轮询 + 清理
	go p.pollLoop(ctx)
	go p.cleanupLoop(ctx)
	fmt.Printf("[%s] 消息轮询器已启动，间隔: %v\n", p.botClient.Name, p.pollInterval)
}

// cleanupLoop 定期清理已处理消息记录，避免内存无限增长
func (p *MessagePoller) cleanupLoop(ctx context.Context) {
	defer p.wg.Done()
	cleanupTicker := time.NewTicker(p.dedupWindow)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-p.stopCh:
			return
		case <-cleanupTicker.C:
			// 清理一半的旧记录（下次清理时再清另一半）
			keys := make([]string, 0, 100)
			p.processedMsgs.Range(func(key, _ any) bool {
				keys = append(keys, key.(string))
				return true
			})
			for i, k := range keys {
				if i >= 100 { // 每次清理最多 100 个
					break
				}
				p.processedMsgs.Delete(k)
			}
		}
	}
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
	currentTime := time.Now().UnixMilli()

	// 计算本次请求的时间窗口 [startTime, endTime]，严格不重叠
	startTime := p.lastRequestTime
	if startTime == 0 {
		startTime = currentTime - 60*1000 // 首次请求：向前取 1 分钟
	}
	endTime := currentTime // 本次请求的截止时间

	// 必须在 API 调用之前更新 lastRequestTime，下次轮询从 endTime 开始
	p.lastRequestTime = endTime

	// 构建请求，时间窗口 [startTime, endTime]，严格不重叠
	// 使用升序排列（ByCreateTimeAsc），消息按时间顺序返回，便于理解
	req := larkim.NewListMessageReqBuilder().
		ContainerIdType("chat").
		ContainerId(p.robotGroupID).
		StartTime(fmt.Sprintf("%d", startTime/1000)). // API 需要秒时间戳
		EndTime(fmt.Sprintf("%d", endTime/1000)).   // 截止时间（不含）
		SortType(larkim.SortTypeListMessageByCreateTimeAsc). // 升序：最旧的消息在前
		PageSize(50).
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

		// 消息 ID 去重：同一消息只处理一次
		if msg.MessageId == nil || *msg.MessageId == "" {
			continue
		}
		msgID := *msg.MessageId
		if _, loaded := p.processedMsgs.LoadOrStore(msgID, struct{}{}); loaded {
			continue // 已处理过，跳过
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
		fmt.Printf("[%s] 轮询发现任务: sender=%s, task=%s, msg_id=%s\n", p.botClient.Name, senderAppID, taskName, msgID)
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

// executeTask 执行任务并发送结果到群（流式输出，单条消息实时更新）
func (p *MessagePoller) executeTask(ctx context.Context, taskName string) error {
	scriptPath, ok := p.taskScripts[taskName]
	if !ok {
		return fmt.Errorf("任务不存在: %s", taskName)
	}

	// 1. 发送初始卡片（带 update_multi: true，允许后续更新）
	initialCard := p.buildTaskCard(taskName, "running", "正在执行...")
	msgID, err := p.sendCardMessage(ctx, initialCard)
	if err != nil {
		fmt.Printf("❌ [%s] 发送卡片失败: %v\n", p.botClient.Name, err)
		return err
	}
	fmt.Printf("📤 [%s] 任务开始: %s, msg_id=%s\n", p.botClient.Name, taskName, msgID)

	// 2. 执行脚本并实时更新卡片
	cmd := exec.Command(scriptPath)
	stdout, err := cmd.StdoutPipe()
	stderr, err := cmd.StderrPipe()
	if err != nil {
		p.patchCardMessage(ctx, msgID, p.buildTaskCard(taskName, "failed", fmt.Sprintf("启动失败: %v", err)))
		return err
	}

	if err := cmd.Start(); err != nil {
		p.patchCardMessage(ctx, msgID, p.buildTaskCard(taskName, "failed", fmt.Sprintf("启动失败: %v", err)))
		return err
	}

	// 实时读取并更新卡片（每 0.5 秒）
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	var outputBuf, errorBuf strings.Builder
	var lastUpdate time.Time
	buf := make([]byte, 1024)

	// 读取 stdout
	go func() {
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				outputBuf.Write(buf[:n])
			}
			if err != nil {
				break
			}
		}
	}()

	// 读取 stderr
	for {
		n, err := stderr.Read(buf)
		if n > 0 {
			errorBuf.Write(buf[:n])
		}
		if err != nil {
			break
		}

		// 定期更新卡片显示进度
		if time.Since(lastUpdate) > 500*time.Millisecond {
			p.updateCardProgress(ctx, msgID, taskName, "running", outputBuf.String(), errorBuf.String())
			lastUpdate = time.Now()
		}
	}

	// 等待命令结束
	cmd.Wait()

	// 3. 更新最终结果
	if cmd.ProcessState.Success() {
		p.updateCardProgress(ctx, msgID, taskName, "success", outputBuf.String(), errorBuf.String())
		fmt.Printf("✅ [%s] 任务完成: %s\n", p.botClient.Name, taskName)
	} else {
		p.updateCardProgress(ctx, msgID, taskName, "failed", outputBuf.String(), errorBuf.String())
		fmt.Printf("❌ [%s] 任务失败: %s\n", p.botClient.Name, taskName)
	}

	return nil
}

// buildTaskCard 构建任务卡片（带 update_multi: true）
func (p *MessagePoller) buildTaskCard(taskName, status, output string) string {
	statusText := map[string]string{
		"running": "⏳ 执行中",
		"success": "✅ 执行成功",
		"failed":  "❌ 执行失败",
	}[status]

	statusColor := map[string]string{
		"running": "grey",
		"success": "green",
		"failed":  "red",
	}[status]

	// 格式化输出
	formattedOutput := strings.TrimSpace(output)
	if formattedOutput == "" {
		formattedOutput = "正在执行..."
	}

	// 飞书交互式卡片 JSON 格式，必须包含 update_multi: true 才能更新
	card := map[string]interface{}{
		"config": map[string]bool{
			"wide_screen_mode": true,
			"update_multi":    true, // 允许更新，对所有用户可见
		},
		"header": map[string]interface{}{
			"title": map[string]string{
				"content": fmt.Sprintf("%s [%s]", statusText, taskName),
				"tag":     "plain_text",
			},
			"template": statusColor,
		},
		"elements": []map[string]interface{}{
			{
				"tag": "div",
				"text": map[string]string{
					"content": fmt.Sprintf("```\n%s\n```", formattedOutput),
					"tag":     "lark_md",
				},
			},
		},
	}

	// 序列化为 JSON
	cardBytes, _ := json.Marshal(card)
	return string(cardBytes)
}

// sendCardMessage 发送卡片消息
func (p *MessagePoller) sendCardMessage(ctx context.Context, card string) (string, error) {
	msgType := "interactive"
	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &p.robotGroupID,
			MsgType:   &msgType,
			Content:   &card,
		}).
		Build()

	resp, err := p.botClient.LarkClient.Im.V1.Message.Create(ctx, req)
	if err != nil {
		return "", err
	}
	if !resp.Success() {
		return "", fmt.Errorf("code=%d, msg=%s", resp.Code, resp.Msg)
	}

	if resp.Data == nil || resp.Data.MessageId == nil {
		return "", fmt.Errorf("no message id in response")
	}

	return *resp.Data.MessageId, nil
}

// updateCardProgress 更新卡片进度
func (p *MessagePoller) updateCardProgress(ctx context.Context, messageID, taskName, status, output, errorMsg string) {
	formattedOutput := strings.TrimSpace(output)
	if formattedOutput == "" {
		formattedOutput = "正在执行..."
	}

	if errorMsg != "" {
		formattedOutput += fmt.Sprintf("\n\n**错误:**\n```\n%s\n```", strings.TrimSpace(errorMsg))
	}

	card := p.buildTaskCard(taskName, status, formattedOutput)
	p.patchCardMessage(ctx, messageID, card)
}

// patchCardMessage 使用 PATCH 更新卡片（不会显示"已编辑"）
func (p *MessagePoller) patchCardMessage(ctx context.Context, messageID, card string) {
	req := larkim.NewPatchMessageReqBuilder().
		MessageId(messageID).
		Body(larkim.NewPatchMessageReqBodyBuilder().
			Content(card).
			Build()).
		Build()

	resp, err := p.botClient.LarkClient.Im.V1.Message.Patch(ctx, req)
	if err != nil {
		fmt.Printf("⚠️ [%s] 更新卡片失败: %v\n", p.botClient.Name, err)
		return
	}
	if !resp.Success() {
		fmt.Printf("⚠️ [%s] 更新卡片失败: code=%d, msg=%s\n", p.botClient.Name, resp.Code, resp.Msg)
	}
}
