package bot

import (
	"fmt"

	lark "github.com/larksuite/oapi-sdk-go/v3"
)

// LarkClient 飞书客户端封装
type LarkClient struct {
	BotID     string
	AppID     string
	AppSecret string

	// 飞书 SDK 客户端
	client *lark.Client
}

// NewLarkClient 创建飞书客户端
func NewLarkClient(botClient *BotClient) (*LarkClient, error) {
	if botClient == nil {
		return nil, fmt.Errorf("bot 不能为空")
	}

	c := lark.NewClient(botClient.AppID, botClient.AppSecret)

	return &LarkClient{
		BotID:     botClient.AppID,
		AppID:     botClient.AppID,
		AppSecret: botClient.AppSecret,
		client:    c,
	}, nil
}

// GetClient 获取飞书 SDK 客户端
func (c *LarkClient) GetClient() *lark.Client {
	return c.client
}

// DebugLog 打印调试日志
func (c *LarkClient) DebugLog(format string, args ...interface{}) {
	fmt.Printf("[DEBUG] LarkClient[%s]: "+format+"\n", append([]interface{}{c.BotID}, args...)...)
}

// LogInfo 打印信息日志
func (c *LarkClient) LogInfo(format string, args ...interface{}) {
	fmt.Printf("[INFO] LarkClient[%s]: "+format+"\n", append([]interface{}{c.BotID}, args...)...)
}

// LogError 打印错误日志
func (c *LarkClient) LogError(format string, args ...interface{}) {
	fmt.Printf("[ERROR] LarkClient[%s]: "+format+"\n", append([]interface{}{c.BotID}, args...)...)
}
