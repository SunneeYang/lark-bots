package router

import (
	"context"
	"testing"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// MockHandler 用于测试
type MockHandler struct {
	Called bool
}

func (m *MockHandler) Handle(ctx context.Context, event interface{}, botClient *bot.BotClient) error {
	m.Called = true
	return nil
}

func TestMessageRouter_RegisterHandler(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	mockHandler := &MockHandler{}

	router.RegisterHandler("dispatcher", mockHandler)

	// 验证 handler 已注册
	// 实际测试在 Route 测试中进行
}

func TestMessageRouter_Route(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 注册测试机器人
	testBot := bot.NewBotClient("test", "cli_123", "secret", "dispatcher")
	registry.Register(testBot)

	// 注册 handler
	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)

	// 创建模拟事件
	event := map[string]interface{}{
		"receiver": map[string]interface{}{
			"bot_id": "cli_123",
		},
	}

	// 路由
	err := router.Route(context.Background(), event)
	if err != nil {
		t.Fatalf("Route failed: %v", err)
	}

	// 验证 handler 被调用
	if !mockHandler.Called {
		t.Error("Expected handler to be called")
	}
}

func TestMessageRouter_Route_BotNotFound(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 注册 handler
	mockHandler := &MockHandler{}
	router.RegisterHandler("dispatcher", mockHandler)

	// 创建不存在的机器人事件
	event := map[string]interface{}{
		"receiver": map[string]interface{}{
			"bot_id": "cli_nonexistent",
		},
	}

	// 路由应该失败
	err := router.Route(context.Background(), event)
	if err == nil {
		t.Error("Expected error for nonexistent bot, got nil")
	}

	expectedMsg := "未找到机器人"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMessageRouter_Route_HandlerNotFound(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 注册测试机器人，但不注册对应 handler
	testBot := bot.NewBotClient("test", "cli_123", "secret", "executor")
	registry.Register(testBot)

	// 创建模拟事件
	event := map[string]interface{}{
		"receiver": map[string]interface{}{
			"bot_id": "cli_123",
		},
	}

	// 路由应该失败（没有 executor handler）
	err := router.Route(context.Background(), event)
	if err == nil {
		t.Error("Expected error for missing handler, got nil")
	}

	expectedMsg := "未找到角色"
	if err.Error()[:len(expectedMsg)] != expectedMsg {
		t.Errorf("Expected error message to start with '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestMessageRouter_Route_InvalidEvent(t *testing.T) {
	registry := bot.NewBotRegistry()
	router := NewMessageRouter(registry)

	// 测试各种无效事件格式
	invalidEvents := []interface{}{
		nil,
		"string",
		123,
		map[string]interface{}{},
		map[string]interface{}{
			"receiver": "invalid",
		},
		map[string]interface{}{
			"receiver": map[string]interface{}{},
		},
		map[string]interface{}{
			"receiver": map[string]interface{}{
				"bot_id": 123, // 类型错误
			},
		},
	}

	for i, event := range invalidEvents {
		err := router.Route(context.Background(), event)
		if err == nil {
			t.Errorf("Test case %d: Expected error for invalid event, got nil", i)
		}

		expectedMsg := "解析事件失败"
		if err.Error()[:len(expectedMsg)] != expectedMsg {
			t.Errorf("Test case %d: Expected error message to start with '%s', got '%s'", i, expectedMsg, err.Error())
		}
	}
}

func TestExtractReceiverBotID(t *testing.T) {
	tests := []struct {
		name        string
		event       interface{}
		expectedID  string
		expectError bool
	}{
		{
			name: "valid event",
			event: map[string]interface{}{
				"receiver": map[string]interface{}{
					"bot_id": "cli_123",
				},
			},
			expectedID:  "cli_123",
			expectError: false,
		},
		{
			name:        "nil event",
			event:       nil,
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "string event",
			event:       "invalid",
			expectedID:  "",
			expectError: true,
		},
		{
			name: "missing receiver field",
			event: map[string]interface{}{
				"other": "data",
			},
			expectedID:  "",
			expectError: true,
		},
		{
			name: "receiver is not a map",
			event: map[string]interface{}{
				"receiver": "invalid",
			},
			expectedID:  "",
			expectError: true,
		},
		{
			name: "missing bot_id field",
			event: map[string]interface{}{
				"receiver": map[string]interface{}{
					"other": "data",
				},
			},
			expectedID:  "",
			expectError: true,
		},
		{
			name: "bot_id is not a string",
			event: map[string]interface{}{
				"receiver": map[string]interface{}{
					"bot_id": 123,
				},
			},
			expectedID:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			botID, err := extractReceiverBotID(tt.event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
				if botID != tt.expectedID {
					t.Errorf("Expected bot_id '%s', got '%s'", tt.expectedID, botID)
				}
			}
		})
	}
}
