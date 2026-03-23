package handler

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

func TestHandlerInterface(t *testing.T) {
	// 测试接口实现
	var _ MessageHandler = (*MockHandler)(nil)
}

// MockHandler 测试用的 mock handler
type MockHandler struct{}

func (m *MockHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	return nil
}
