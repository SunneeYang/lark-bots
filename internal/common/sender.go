package common

import (
	"context"
	"encoding/json"
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// Sender 消息发送器
type Sender struct {
	RobotGroupID string
	larkClient   *lark.Client
}

// NewSender 创建消息发送器
func NewSender(larkClient *lark.Client) *Sender {
	return &Sender{larkClient: larkClient}
}

// SetRobotGroupID 设置机器人群 ID
func (s *Sender) SetRobotGroupID(groupID string) {
	s.RobotGroupID = groupID
}

// SendToGroup 发送文本消息到指定群
func (s *Sender) SendToGroup(message string) error {
	if s.larkClient == nil {
		return fmt.Errorf("飞书客户端未初始化")
	}
	if s.RobotGroupID == "" {
		return fmt.Errorf("机器人群 ID 未配置")
	}

	// 使用 json.Marshal 正确转义特殊字符
	contentData := map[string]string{"text": message}
	contentBytes, err := json.Marshal(contentData)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	contentStr := string(contentBytes)
	msgType := "text"

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &s.RobotGroupID,
			MsgType:   &msgType,
			Content:   &contentStr,
		}).Build()

	resp, err := s.larkClient.Im.V1.Message.Create(context.Background(), req)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("发送消息失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

// SendToChatID 发送消息到指定会话（支持群聊和私聊）
func (s *Sender) SendToChatID(chatID, msgType, content string) error {
	if s.larkClient == nil {
		return fmt.Errorf("飞书客户端未初始化")
	}
	if chatID == "" {
		return fmt.Errorf("chatID 不能为空")
	}

	// 使用 json.Marshal 正确转义特殊字符
	contentData := map[string]string{"text": content}
	contentBytes, err := json.Marshal(contentData)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	contentStr := string(contentBytes)

	req := larkim.NewCreateMessageReqBuilder().
		ReceiveIdType("chat_id").
		Body(&larkim.CreateMessageReqBody{
			ReceiveId: &chatID,
			MsgType:   &msgType,
			Content:   &contentStr,
		}).Build()

	resp, err := s.larkClient.Im.V1.Message.Create(context.Background(), req)
	if err != nil {
		return fmt.Errorf("发送消息失败: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("发送消息失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

// ReplyToMessage 回复指定消息（支持群聊和私聊）
// parentMessageID: 要回复的消息 ID
// msgType: 消息类型（通常为 "text"）
// content: 消息内容
func (s *Sender) ReplyToMessage(parentMessageID, msgType, content string) error {
	if s.larkClient == nil {
		return fmt.Errorf("飞书客户端未初始化")
	}
	if parentMessageID == "" {
		return fmt.Errorf("parentMessageID 不能为空")
	}

	// 使用 json.Marshal 正确转义特殊字符
	contentData := map[string]string{"text": content}
	contentBytes, err := json.Marshal(contentData)
	if err != nil {
		return fmt.Errorf("序列化消息失败: %w", err)
	}
	contentStr := string(contentBytes)

	req := larkim.NewReplyMessageReqBuilder().
		MessageId(parentMessageID).
		Body(&larkim.ReplyMessageReqBody{
			MsgType: &msgType,
			Content: &contentStr,
		}).Build()

	resp, err := s.larkClient.Im.V1.Message.Reply(context.Background(), req)
	if err != nil {
		return fmt.Errorf("回复消息失败: %w", err)
	}

	if !resp.Success() {
		return fmt.Errorf("回复消息失败: code=%d, msg=%s", resp.Code, resp.Msg)
	}

	return nil
}

