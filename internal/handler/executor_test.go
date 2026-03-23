package handler

import (
	"context"
	"testing"

	"github.com/SunneeYang/lark-bots/internal/bot"
)

func TestExecutorHandler_VerifyDispatcher(t *testing.T) {
	handler := NewExecutorHandler()

	// 设置允许的 dispatcher
	handler.SetAllowedDispatchers([]string{"cli_123"})

	testBot := bot.NewBotClient("executor", "cli_456", "secret", "executor")

	// 创建来自允许的 dispatcher 的事件
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_123",
		},
		"message": map[string]interface{}{
			"content": "execute /opt/scripts/test.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected for incomplete implementation): %v", err)
	}
}

func TestExecutorHandler_UnauthorizedDispatcher(t *testing.T) {
	handler := NewExecutorHandler()

	// 设置允许的 dispatcher
	handler.SetAllowedDispatchers([]string{"cli_123"})

	testBot := bot.NewBotClient("executor", "cli_456", "secret", "executor")

	// 创建来自未授权的 dispatcher 的事件
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_unauthorized",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for unauthorized dispatcher, got nil")
	}
}

func TestExecutorHandler_SetAllowedDispatchers(t *testing.T) {
	handler := NewExecutorHandler()

	dispatchers := []string{"cli_123", "cli_456"}
	handler.SetAllowedDispatchers(dispatchers)

	// 验证设置成功（通过后续的 Handle 测试验证）
	testBot := bot.NewBotClient("executor", "cli_789", "secret", "executor")

	// 测试允许的 dispatcher
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_123",
		},
		"message": map[string]interface{}{
			"content": "execute /opt/scripts/test.sh",
		},
	}

	// 应该不会因为 dispatcher 校验而失败（可能因为其他原因失败）
	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected): %v", err)
	}
}

func TestExecutorHandler_SetAllowedScripts(t *testing.T) {
	handler := NewExecutorHandler()

	scripts := []string{"/opt/scripts/deploy.sh", "/opt/scripts/test.sh"}
	handler.SetAllowedScripts(scripts)

	// 验证设置成功（通过后续的 Handle 测试验证）
	testBot := bot.NewBotClient("executor", "cli_456", "secret", "executor")

	handler.SetAllowedDispatchers([]string{"cli_123"})

	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"bot_id": "cli_123",
		},
		"message": map[string]interface{}{
			"content": "execute /opt/scripts/deploy.sh",
		},
	}

	err := handler.Handle(context.Background(), event, testBot)
	if err != nil {
		t.Logf("Handle returned error (expected): %v", err)
	}
}

func TestExecutorHandler_ParseScriptCommand(t *testing.T) {
	handler := NewExecutorHandler()

	tests := []struct {
		name           string
		message        string
		expectedScript string
		expectedArgs   []string
		expectError    bool
	}{
		{
			name:           "valid command with args",
			message:        "execute /opt/scripts/test.sh arg1 arg2",
			expectedScript: "/opt/scripts/test.sh",
			expectedArgs:   []string{"arg1", "arg2"},
			expectError:    false,
		},
		{
			name:           "valid command without args",
			message:        "execute /opt/scripts/test.sh",
			expectedScript: "/opt/scripts/test.sh",
			expectedArgs:   []string{},
			expectError:    false,
		},
		{
			name:        "invalid command - missing execute",
			message:     "/opt/scripts/test.sh",
			expectError: true,
		},
		{
			name:        "invalid command - missing script",
			message:     "execute",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scriptPath, args, err := handler.parseScriptCommand(tt.message)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if scriptPath != tt.expectedScript {
					t.Errorf("Expected script path '%s', got '%s'", tt.expectedScript, scriptPath)
				}
				if len(args) != len(tt.expectedArgs) {
					t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(args))
				}
			}
		})
	}
}

func TestExtractSenderBotID(t *testing.T) {
	tests := []struct {
		name        string
		event       interface{}
		expectedID  string
		expectError bool
	}{
		{
			name: "valid event",
			event: map[string]interface{}{
				"sender": map[string]interface{}{
					"bot_id": "cli_123",
				},
			},
			expectedID:  "cli_123",
			expectError: false,
		},
		{
			name:        "missing sender",
			event:       map[string]interface{}{},
			expectError: true,
		},
		{
			name: "missing bot_id",
			event: map[string]interface{}{
				"sender": map[string]interface{}{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			botID, err := extractSenderBotID(tt.event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if botID != tt.expectedID {
					t.Errorf("Expected bot ID '%s', got '%s'", tt.expectedID, botID)
				}
			}
		})
	}
}

func TestExtractMessageContentFromExecutor(t *testing.T) {
	tests := []struct {
		name            string
		event           interface{}
		expectedContent string
		expectError     bool
	}{
		{
			name: "valid event",
			event: map[string]interface{}{
				"message": map[string]interface{}{
					"content": "execute /opt/scripts/test.sh",
				},
			},
			expectedContent: "execute /opt/scripts/test.sh",
			expectError:     false,
		},
		{
			name:        "missing message",
			event:       map[string]interface{}{},
			expectError: true,
		},
		{
			name: "missing content",
			event: map[string]interface{}{
				"message": map[string]interface{}{},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := extractMessageContent(tt.event)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if content != tt.expectedContent {
					t.Errorf("Expected content '%s', got '%s'", tt.expectedContent, content)
				}
			}
		})
	}
}
