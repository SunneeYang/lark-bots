package bot

import (
	"time"

	lark "github.com/larksuite/oapi-sdk-go/v3"
)

// Now 返回当前时间
func Now() time.Time {
	return time.Now()
}

// TaskRecord 任务记录
type TaskRecord struct {
	ID         string
	TaskName   string
	Dispatcher string
	Executor   string
	User       string
	Status     string // "pending" | "running" | "completed" | "failed"
	StartTime  time.Time
	EndTime    *time.Time
	Result     string
	Error      string
}

// BotClient 机器人客户端
type BotClient struct {
	Name      string
	AppID     string
	AppSecret string
	Role      string
	OpenID    string // 机器人的 open_id，用于 @ 提及

	// 飞书 SDK 客户端（每个机器人用自己的 client 发送消息）
	LarkClient *lark.Client

	// Executor 特有配置
	AllowedDispatchers []string
}

// NewBotClient 创建机器人客户端
func NewBotClient(name, appID, appSecret, role string) *BotClient {
	return &BotClient{
		Name:      name,
		AppID:     appID,
		AppSecret: appSecret,
		Role:      role,
	}
}

// InitLarkClient 初始化飞书 SDK 客户端
func (b *BotClient) InitLarkClient() {
	b.LarkClient = lark.NewClient(b.AppID, b.AppSecret)
}
