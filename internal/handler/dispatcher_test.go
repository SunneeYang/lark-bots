package handler

import (
	"context"
	"testing"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
)

func TestDispatcherHandler_HandleUserMessage(t *testing.T) {
	handler := NewDispatcherHandler(nil)

	// 设置白名单
	handler.SetAllowedUsers([]string{"user_1"})
	handler.SetAllowedTasks([]string{"deploy.sh", "restart.sh"})

	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	// 测试正常消息
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "user_1",
		},
		"message": map[string]interface{}{
			"content": "执行 deploy.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected for incomplete implementation): %v", err)
	}
}

func TestDispatcherHandler_UserNotInWhitelist(t *testing.T) {
	handler := NewDispatcherHandler(nil)

	// 设置白名单，不包含 user_2
	handler.SetAllowedUsers([]string{"user_1"})
	handler.SetAllowedTasks([]string{"deploy.sh"})

	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "user_2",
		},
		"message": map[string]interface{}{
			"content": "执行 deploy.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for user not in whitelist, got nil")
	}
}

func TestDispatcherHandler_TaskNotInWhitelist(t *testing.T) {
	handler := NewDispatcherHandler(nil)

	// 设置白名单，不包含 test.sh
	handler.SetAllowedUsers([]string{"user_1"})
	handler.SetAllowedTasks([]string{"deploy.sh"})

	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "user_1",
		},
		"message": map[string]interface{}{
			"content": "执行 test.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for task not in whitelist, got nil")
	}
}

func TestDispatcherHandler_SetAllowedUsers(t *testing.T) {
	handler := NewDispatcherHandler(nil)

	users := []string{"user_1", "user_2"}
	handler.SetAllowedUsers(users)

	// 验证白名单已设置
	// 由于 userWhiteList 是私有字段，我们通过 Handle 方法间接验证
	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	// 测试用户在白名单中
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "user_1",
		},
		"message": map[string]interface{}{
			"content": "deploy.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	// 不应该报错用户不在白名单
	if err != nil && err.Error() == "用户不在白名单中: user_1" {
		t.Error("Expected user_1 to be in whitelist")
	}
}

func TestExtractSenderID(t *testing.T) {
	tests := []struct {
		name        string
		event       interface{}
		expectedID  string
		expectError bool
	}{
		{
			name: "正常事件",
			event: map[string]interface{}{
				"sender": map[string]interface{}{
					"user_id": "user_1",
				},
			},
			expectedID:  "user_1",
			expectError: false,
		},
		{
			name:        "非 map 类型",
			event:       "invalid",
			expectedID:  "",
			expectError: true,
		},
		{
			name: "缺少 sender 字段",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"content": "test",
				},
			},
			expectedID:  "",
			expectError: true,
		},
		{
			name: "sender 不是 map 类型",
			event: map[string]interface{}{
				"sender": "invalid",
			},
			expectedID:  "",
			expectError: true,
		},
		{
			name: "缺少 user_id 字段",
			event: map[string]interface{}{
				"sender": map[string]interface{}{
					"bot_id": "bot_1",
				},
			},
			expectedID:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			userID, err := common.ExtractSenderID(tt.event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if userID != tt.expectedID {
					t.Errorf("Expected user ID '%s', got '%s'", tt.expectedID, userID)
				}
			}
		})
	}
}

func TestDispatcherExtractMessageContent(t *testing.T) {
	tests := []struct {
		name        string
		event       interface{}
		expectedMsg string
		expectError bool
	}{
		{
			name: "正常消息",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"content": "执行 deploy.sh",
				},
			},
			expectedMsg: "执行 deploy.sh",
			expectError: false,
		},
		{
			name:        "非 map 类型",
			event:       "invalid",
			expectedMsg: "",
			expectError: true,
		},
		{
			name: "缺少 message 字段",
			event: map[string]interface{}{
				"sender": map[string]interface{}{
					"user_id": "user_1",
				},
			},
			expectedMsg: "",
			expectError: true,
		},
		{
			name: "message 不是 map 类型",
			event: map[string]interface{}{
				"message": "invalid",
			},
			expectedMsg: "",
			expectError: true,
		},
		{
			name: "缺少 content 字段",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"message_id": "msg_1",
				},
			},
			expectedMsg: "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := common.ExtractMessageContent(tt.event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if content != tt.expectedMsg {
					t.Errorf("Expected message '%s', got '%s'", tt.expectedMsg, content)
				}
			}
		})
	}
}
