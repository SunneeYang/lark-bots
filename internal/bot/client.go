package bot

import (
	"fmt"
)

// LarkClient 飞书客户端封装
type LarkClient struct {
	BotID     string
	AppID     string
	AppSecret string

	// 飞书 SDK 客户端（懒加载）
	// actual SDK client will be initialized later
	client interface{}
}

// NewLarkClient 创建飞书客户端
func NewLarkClient(bot *BotClient) (*LarkClient, error) {
	if bot == nil {
		return nil, fmt.Errorf("bot 不能为空")
	}

	return &LarkClient{
		BotID:     bot.AppID,
		AppID:     bot.AppID,
		AppSecret: bot.AppSecret,
	}, nil
}

// Init 初始化飞书 SDK 客户端
func (c *LarkClient) Init() error {
	// 创建飞书客户端
	// 实际 SDK 初始化将在后续任务中完成
	return nil
}

// SendMessage 发送消息（待实现）
func (c *LarkClient) SendMessage(chatID, message string) error {
	// 待实现
	return nil
}
