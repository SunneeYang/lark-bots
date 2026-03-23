package bot

import (
	"time"
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

	// Executor 特有配置
	AllowedDispatchers []string
	AllowedScripts     []string
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
