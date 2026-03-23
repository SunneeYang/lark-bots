package common

import (
	"context"
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

	contentStr := fmt.Sprintf(`{"text":"%s"}`, message)
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

	contentStr := fmt.Sprintf(`{"text":"%s"}`, content)

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

