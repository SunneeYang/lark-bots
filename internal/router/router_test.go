package router

import (
	"context"
	"strings"
	"testing"

	"github.com/SunneeYang/lark-bots/internal/bot"
)

// MockHandler 用于测试
type MockHandler struct {
	Called      bool
	ReceivedBot *bot.BotClient
}

func (m *MockHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	m.Called = true
	m.ReceivedBot = botClient
	return nil
}

func TestMessageRouter_RegisterHandler(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)
}

func TestMessageRouter_Route(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	testBot := bot.NewBotClient("test", "cli_123", "secret", "dispatcher")
	registry.Register(testBot)

	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)

	event := map[string]interface{}{
		"content": "hello",
	}

	err := router.Route(context.Background(), event, testBot)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	if !mockHandler.Called {
		t.Error("Expected handler to be called")
	}

	if mockHandler.ReceivedBot != testBot {
		t.Errorf("Expected bot %v, got %v", testBot, mockHandler.ReceivedBot)
	}
}

func TestMessageRouter_Route_BotNotFound(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)

	event := map[string]interface{}{}

	err := router.Route(context.Background(), event, nil)
	if err == nil {
		t.Error("Expected error for nil bot, got nil")
	}

	if err.Error() != "机器人不能为空" {
		t.Errorf("Expected error '机器人不能为空', got '%s'", err.Error())
	}
}

func TestMessageRouter_Route_HandlerNotFound(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	testBot := bot.NewBotClient("test", "cli_123", "secret", "executor")
	registry.Register(testBot)

	event := map[string]interface{}{}

	err := router.Route(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for missing handler, got nil")
	}

	expectedInMsg := "的处理器"
	if !strings.Contains(err.Error(), expectedInMsg) {
		t.Errorf("Expected error message to contain '%s', got '%s'", expectedInMsg, err.Error())
	}
}

func TestMessageRouter_Route_ByBotName(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 注册两个相同角色的 bot
	bot1 := bot.NewBotClient("bot-one", "cli_1", "secret", "executor")
	bot2 := bot.NewBotClient("bot-two", "cli_2", "secret", "executor")
	registry.Register(bot1)
	registry.Register(bot2)

	handler1 := &MockHandler{}
	handler2 := &MockHandler{}
	router.RegisterHandler("bot-one", handler1)
	router.RegisterHandler("bot-two", handler2)

	event := map[string]interface{}{}

	router.Route(context.Background(), event, bot1)
	if !handler1.Called || handler2.Called {
		t.Error("Expected only handler1 to be called")
	}

	handler1.Called = false
	handler2.Called = false

	router.Route(context.Background(), event, bot2)
	if handler1.Called || !handler2.Called {
		t.Error("Expected only handler2 to be called")
	}
}
