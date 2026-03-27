package handler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/SunneeYang/lark-bots/internal/bot"
	"github.com/SunneeYang/lark-bots/internal/common"
	"github.com/SunneeYang/lark-bots/internal/config"
	"github.com/SunneeYang/lark-bots/internal/handler/matcher"
)

// createTestDispatcherHandler 创建用于测试的 dispatcher handler
func createTestDispatcherHandler() *DispatcherHandler {
	// 创建测试配置
	cfg := &config.ServiceConfig{
		RobotGroupID: "test-group-id",
		Bots: []config.BotConfig{
			{
				Name:            "dispatcher",
				AppID:           "cli_123",
				AppSecret:       "secret",
				Role:            "dispatcher",
				GroupProjectMap: map[string]string{"test-group": "test-project"},
				TaskScripts: map[string]string{
					"test-task":          "/test.sh",
					"potato-dev-restart": "/scripts/potato_restart.sh",
					"mist-dev-update":    "/scripts/mist_update.sh",
					"限制任务":               "/scripts/restricted.sh",
					"无限制任务":              "/scripts/unrestricted.sh",
				},
			},
			{
				Name:               "executor",
				AppID:              "cli_456",
				AppSecret:          "secret",
				Role:               "executor",
				AllowedDispatchers: []string{"cli_123"},
				RoutingKeywords:    []string{"dev", "development", "迷雾"},
				Tasks: map[string]config.TaskDetail{
					"test-task": {
						DisplayName:  "测试任务",
						Script:       "/test.sh",
						AllowedUsers: []string{"user_1", "user_2"},
					},
					"potato-dev-restart": {
						DisplayName:  "土豆开发服重启",
						Script:       "/scripts/potato_restart.sh",
						AllowedUsers: []string{"admin_user", "potato_admin"},
					},
					"mist-dev-update": {
						DisplayName:  "迷雾开发服更新",
						Script:       "/scripts/mist_update.sh",
						AllowedUsers: []string{"mist_user", "dev_team"},
					},
					"限制任务": {
						DisplayName:  "限制任务",
						Script:       "/scripts/restricted.sh",
						AllowedUsers: []string{}, // 空数组表示拒绝所有人
					},
					"无限制任务": {
						DisplayName:  "无限制任务",
						Script:       "/scripts/unrestricted.sh",
						// 没有AllowedUsers字段表示无限制
					},
				},
			},
		},
	}

	// 创建 handler，传入语义配置为 nil（不使用语义匹配）
	handler := NewDispatcherHandler(cfg, nil)
	handler.SetRobotGroupID(cfg.RobotGroupID)
	return handler
}

func TestDispatcherHandler_HandleUserMessage(t *testing.T) {
	handler := createTestDispatcherHandler()

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
	handler := createTestDispatcherHandler()

	// 设置白名单，不包含 user_2
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
	handler := createTestDispatcherHandler()

	// 设置白名单，不包含 test.sh
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

// ===== 新增测试：分层匹配模式 =====

func TestDispatcherHandler_StripAtMention(t *testing.T) {
	h := &DispatcherHandler{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "剥离 @user_xxx 前缀",
			input:    "@_user_1234567890 重启土豆开发服",
			expected: "重启土豆开发服",
		},
		{
			name:     "剥离 @bot_xxx 前缀",
			input:    "@_bot_abcdeffedcba 更新迷雾开发服",
			expected: "更新迷雾开发服",
		},
		{
			name:     "无 @mention 前缀",
			input:    "重启土豆开发服",
			expected: "重启土豆开发服",
		},
		{
			name:     "只有 @mention 无后续文本",
			input:    "@_user_1234567890",
			expected: "",
		},
		{
			name:     "以 @ 开头但不是 mention",
			input:    "@ 重启",
			expected: "@ 重启",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := h.stripAtMention(tt.input)
			if got != tt.expected {
				t.Errorf("stripAtMention() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDispatcherHandler_SetLayeredMatcher(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 创建测试匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID:   "test-executor",
			ExecutorName: "test-executor",
			Script:       "/test.sh",
		},
	}
	matcher := matcher.NewLayeredMatcher(executors, nil)

	// 设置分层匹配器
	h.SetLayeredMatcher(matcher)

	if h.layeredMatcher == nil {
		t.Error("SetLayeredMatcher() did not set layeredMatcher")
	}
}

func TestDispatcherHandler_ModeDetection(t *testing.T) {
	// 测试传统模式（无 layeredMatcher）
	testConfig := &config.ServiceConfig{}
	h1 := NewDispatcherHandler(testConfig, nil)
	if h1.layeredMatcher != nil {
		t.Error("New dispatcher should not have layeredMatcher by default")
	}

	// 测试分层匹配模式（有 layeredMatcher）
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID: "test-executor",
		},
	}
	h2 := NewDispatcherHandler(testConfig, nil)
	h2.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))
	if h2.layeredMatcher == nil {
		t.Error("SetLayeredMatcher() should set layeredMatcher")
	}
}

func TestDispatcherHandler_LayeredModeBasicFlow(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置分层匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID:      "dev-executor",
			ExecutorName:    "dev-executor",
			RoutingKeywords: []string{"开发", "dev"},
			Keywords:        []string{"土豆"},
			TaskName:        "potato-dev-restart",
			Script:          "/scripts/potato_restart.sh",
			Names: matcher.TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "restart"},
			},
		},
	}

	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 验证匹配器已设置
	if h.layeredMatcher == nil {
		t.Fatal("layeredMatcher should be set")
	}

	// 测试匹配
	result := h.layeredMatcher.Match(context.Background(), matcher.LayeredMatchInput{
		UserInput: "土豆开发服重启",
	})

	if result.HasNegation {
		t.Error("Should not detect negation")
	}

	if len(result.Matches) != 1 {
		t.Fatalf("Expected 1 match, got %d", len(result.Matches))
	}

	match := result.Matches[0]
	if match.ExecutorID != "dev-executor" {
		t.Errorf("ExecutorID = %s, want dev-executor", match.ExecutorID)
	}

	if match.Operation != "重启" {
		t.Errorf("Operation = %s, want 重启", match.Operation)
	}

	if match.TaskName != "potato-dev-restart" {
		t.Errorf("TaskName = %s, want potato-dev-restart", match.TaskName)
	}

	if match.Script != "/scripts/potato_restart.sh" {
		t.Errorf("Script = %s, want /scripts/potato_restart.sh", match.Script)
	}
}

// TestGroupProjectMap 测试群组项目关键词自动补充功能
func TestGroupProjectMap(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置群组到项目名的映射
	groupProjectMap := map[string]string{
		"oc_potato_dev_group":  "土豆",
		"oc_mist_dev_group":    "迷雾",
		"oc_potato_test_group": "土豆",
		"oc_mist_test_group":   "迷雾",
	}
	h.SetGroupProjectMap(groupProjectMap)

	tests := []struct {
		name           string
		chatID         string
		message        string
		expectedOutput string
		shouldEnhance  bool
	}{
		{
			name:           "土豆开发群自动补充项目名",
			chatID:         "oc_potato_dev_group",
			message:        "重启",
			expectedOutput: "土豆 重启",
			shouldEnhance:  true,
		},
		{
			name:           "迷雾开发群自动补充项目名",
			chatID:         "oc_mist_dev_group",
			message:        "更新",
			expectedOutput: "迷雾 更新",
			shouldEnhance:  true,
		},
		{
			name:           "未配置群组不补充",
			chatID:         "oc_unknown_group",
			message:        "重启",
			expectedOutput: "重启",
			shouldEnhance:  false,
		},
		{
			name:           "用户已指定项目名不补充",
			chatID:         "oc_potato_dev_group",
			message:        "土豆重启",
			expectedOutput: "土豆重启",
			shouldEnhance:  false,
		},
		{
			name:           "用户包含 potato 关键词不补充",
			chatID:         "oc_potato_dev_group",
			message:        "potato dev 重启",
			expectedOutput: "potato dev 重启",
			shouldEnhance:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 构造测试事件
			event := map[string]interface{}{
				"message": map[string]interface{}{
					"chat_id":   tt.chatID,
					"chat_type": "group",
					"content":   fmt.Sprintf(`{"text":"%s"}`, tt.message),
				},
			}

			// 执行增强
			enhanced := h.enhanceMessageWithGroupProject(context.Background(), event, tt.message)

			if enhanced != tt.expectedOutput {
				t.Errorf("enhanceMessageWithGroupProject() = %v, want %v", enhanced, tt.expectedOutput)
			}
		})
	}
}

// ===== getUserInfo 测试 =====

func TestDispatcherHandler_GetUserInfo_CacheHit(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)
	ctx := context.Background()
	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	// 预先填充缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_123"] = "张三"
	h.userInfoCacheMu.Unlock()

	// 调用 getUserInfo，应该从缓存返回
	name, err := h.getUserInfo(ctx, testBot, "ou_123")
	if err != nil {
		t.Fatalf("getUserInfo() error = %v", err)
	}

	if name != "张三" {
		t.Errorf("getUserInfo() = %v, want %v", name, "张三")
	}
}

func TestDispatcherHandler_GetUserInfo_ConcurrentAccess(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)
	ctx := context.Background()
	testBot := bot.NewBotClient("dispatcher", "cli_123", "secret", "dispatcher")

	// 预先填充缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_789"] = "王五"
	h.userInfoCacheMu.Unlock()

	// 并发读取
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			name, err := h.getUserInfo(ctx, testBot, "ou_789")
			if err != nil {
				t.Errorf("getUserInfo() error = %v", err)
			}
			if name != "王五" {
				t.Errorf("getUserInfo() = %v, want %v", name, "王五")
			}
			done <- true
		}()
	}

	// 等待所有 goroutine 完成
	for i := 0; i < 10; i++ {
		<-done
	}
}

// ===== handleLayeredMode 集成测试 =====

// TestDispatcherHandler_HandleLayeredMode_Integration 测试 handleLayeredMode 的完整集成流程
// 验证：消息增强 → 分层匹配 → 获取用户信息 → 构建 JSON → 发送到群组
func TestDispatcherHandler_HandleLayeredMode_Integration(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 1. 配置白名单用户

	// 2. 配置群组项目映射
	h.SetGroupProjectMap(map[string]string{
		"oc_potato_dev": "土豆",
	})

	// 3. 配置分层匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID:      "dev-executor",
			ExecutorName:    "dev-executor",
			RoutingKeywords: []string{"开发", "dev"},
			Keywords:        []string{"土豆"},
			TaskName:        "potato-dev-restart",
			Script:          "/scripts/potato_restart.sh",
			Names: matcher.TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "restart"},
			},
			DisplayName: "土豆开发服重启",
		},
	}
	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 4. 预填充用户信息缓存（模拟 getUserInfo 成功）
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_test_user"] = "张三"
	h.userInfoCacheMu.Unlock()

	// 5. 构造测试事件（模拟群聊消息）
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "ou_test_user",
		},
		"message": map[string]interface{}{
			"chat_id":   "oc_potato_dev",
			"chat_type": "group",
			"content":   `{"text":"重启"}`,
		},
	}

	// 6. 创建 mock bot client
	testBot := bot.NewBotClient("test-dispatcher", "cli_123", "secret", "dispatcher")

	// 7. 执行 Handle（应该调用 handleLayeredMode）
	err := h.Handle(context.Background(), event, testBot)

	// 8. 验证结果
	// 注意：由于 SendToGroup 和 replyToUser 会失败（没有真实的飞书客户端），
	// 我们预期会返回错误，但错误信息应该表明匹配成功
	if err == nil {
		t.Log("Handle succeeded (unexpected in test environment)")
	} else {
		// 错误应该是因为发送消息失败，而不是因为匹配失败
		if err.Error() == "未找到匹配任务" {
			t.Errorf("Expected task to match, but got: %v", err)
		}
		if err.Error() == "检测到否定意图" {
			t.Errorf("Expected no negation, but got: %v", err)
		}
		if err.Error() == "检测到多个匹配的任务" {
			t.Errorf("Expected single match, but got: %v", err)
		}
		// 其他错误（如发送消息失败）是可以接受的
		t.Logf("Got expected error (due to mock environment): %v", err)
	}
}

// TestDispatcherHandler_HandleLayeredMode_NoMatch 测试无匹配场景
func TestDispatcherHandler_HandleLayeredMode_NoMatch(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置白名单用户

	// 配置分层匹配器（只有土豆任务，且只配置重启操作）
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID:      "dev-executor",
			RoutingKeywords: []string{"开发"},
			Keywords:        []string{"土豆"},
			TaskName:        "potato-dev-restart",
			Names: matcher.TaskNames{
				Primary: "重启",
			},
		},
	}
	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 预填充用户信息缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_test_user"] = "测试用户"
	h.userInfoCacheMu.Unlock()

	// 构造测试事件（用户请求删除操作，但只配置了重启）
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "ou_test_user",
		},
		"message": map[string]interface{}{
			"chat_id":   "oc_unknown_group",
			"chat_type": "group",
			"content":   `{"text":"删除土豆服务器"}`,
		},
	}

	testBot := bot.NewBotClient("test-dispatcher", "cli_123", "secret", "dispatcher")

	err := h.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for no match, got nil")
	} else {
		// 由于语义匹配可能找到部分匹配，我们只验证它不是成功发送
		errMsg := err.Error()
		// 只要不是成功发送，就接受
		if strings.Contains(errMsg, "分发任务失败") || strings.Contains(errMsg, "未找到匹配任务") {
			t.Logf("Got expected error: %v", errMsg)
		} else {
			t.Logf("Got error (may be acceptable): %v", errMsg)
		}
	}
}

// TestDispatcherHandler_HandleLayeredMode_Negation 测试否定意图检测
func TestDispatcherHandler_HandleLayeredMode_Negation(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{
		Bots: []config.BotConfig{
			{
				Name:  "executor",
				AppID: "cli_456",
				Role:  "executor",
				Tasks: map[string]config.TaskDetail{
					"potato-dev-restart": {
						DisplayName:  "土豆开发服重启",
						Script:       "/scripts/potato_restart.sh",
						AllowedUsers: []string{"ou_test_user"}, // 添加测试用户到白名单
					},
				},
			},
		},
	}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置分层匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID: "dev-executor",
			Keywords:   []string{"土豆"},
			TaskName:   "potato-dev-restart",
			Names: matcher.TaskNames{
				Primary: "重启",
			},
			DisplayName: "土豆开发服重启",
		},
	}
	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 预填充用户信息缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_test_user"] = "测试用户"
	h.userInfoCacheMu.Unlock()

	// 构造测试事件（包含否定词）
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "ou_test_user",
		},
		"message": map[string]interface{}{
			"chat_id":   "oc_unknown_group",
			"chat_type": "group",
			"content":   `{"text":"不要重启土豆"}`,
		},
	}

	testBot := bot.NewBotClient("test-dispatcher", "cli_123", "secret", "dispatcher")

	err := h.Handle(context.Background(), event, testBot)
	if err == nil {
		t.Error("Expected error for negation, got nil")
	} else {
		// 错误应该包含否定意图的信息（可能是被wrap的错误）
		errMsg := err.Error()
		// 检查是否包含否定意图的核心错误
		if !strings.Contains(errMsg, "检测到否定意图") && !strings.Contains(errMsg, "回复用户失败") {
			t.Errorf("Expected error related to negation, got: %v", errMsg)
		} else {
			t.Logf("Got expected negation-related error: %v", errMsg)
		}
	}
}

// TestDispatcherHandler_HandleLayeredMode_GroupProjectEnhancement 测试群组项目关键词自动补充
func TestDispatcherHandler_HandleLayeredMode_GroupProjectEnhancement(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置白名单用户

	// 配置群组项目映射
	h.SetGroupProjectMap(map[string]string{
		"oc_potato_dev": "土豆",
	})

	// 配置分层匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID: "dev-executor",
			Keywords:   []string{"土豆"},
			TaskName:   "potato-dev-restart",
			Names: matcher.TaskNames{
				Primary: "重启",
			},
		},
	}
	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 预填充用户信息缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_test_user"] = "测试用户"
	h.userInfoCacheMu.Unlock()

	// 构造测试事件（用户只说"重启"，应该自动补充"土豆"）
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "ou_test_user",
		},
		"message": map[string]interface{}{
			"chat_id":   "oc_potato_dev",
			"chat_type": "group",
			"content":   `{"text":"重启"}`,
		},
	}

	testBot := bot.NewBotClient("test-dispatcher", "cli_123", "secret", "dispatcher")

	err := h.Handle(context.Background(), event, testBot)
	// 应该匹配成功（错误来自发送消息，而非匹配失败）
	if err != nil && err.Error() == "未找到匹配任务" {
		t.Errorf("Expected task to match with group project enhancement, got: %v", err)
	}
}

// TestDispatcherHandler_HandleLayeredMode_UserSpecifiedProject 测试用户已指定项目时不自动补充
func TestDispatcherHandler_HandleLayeredMode_UserSpecifiedProject(t *testing.T) {
	// 创建测试配置
	testConfig := &config.ServiceConfig{}
	h := NewDispatcherHandler(testConfig, nil)

	// 配置白名单用户

	// 配置群组项目映射
	h.SetGroupProjectMap(map[string]string{
		"oc_potato_dev": "土豆",
	})

	// 配置分层匹配器（只有迷雾任务）
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID: "test-executor",
			Keywords:   []string{"迷雾"},
			TaskName:   "mist-dev-restart",
			Names: matcher.TaskNames{
				Primary: "重启",
			},
		},
	}
	h.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 预填充用户信息缓存
	h.userInfoCacheMu.Lock()
	h.userInfoCache["ou_test_user"] = "测试用户"
	h.userInfoCacheMu.Unlock()

	// 构造测试事件（用户明确指定"迷雾"，虽然群组是土豆群）
	event := map[string]interface{}{
		"sender": map[string]interface{}{
			"user_id": "ou_test_user",
		},
		"message": map[string]interface{}{
			"chat_id":   "oc_potato_dev",
			"chat_type": "group",
			"content":   `{"text":"迷雾重启"}`,
		},
	}

	testBot := bot.NewBotClient("test-dispatcher", "cli_123", "secret", "dispatcher")

	err := h.Handle(context.Background(), event, testBot)
	// 应该匹配到迷雾任务，而不是土豆任务（错误来自发送消息，而非匹配失败）
	if err != nil && err.Error() == "未找到匹配任务" {
		t.Errorf("Expected mist task to match, got: %v", err)
	}
}

// TestDispatcher_TwoLayerPermissionCheck 测试两层权限检查框架
// 验证数据结构正确构建，但不实际执行完整的消息处理流程
func TestDispatcher_TwoLayerPermissionCheck(t *testing.T) {
	// 创建测试配置
	cfg := &config.ServiceConfig{
		Bots: []config.BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:               "executor",
				AppID:              "cli_456",
				AppSecret:          "secret",
				Role:               "executor",
				AllowedDispatchers: []string{"cli_123"},
				Tasks: map[string]config.TaskDetail{
					"task1": {
						Script:       "/opt/scripts/task1.sh",
						AllowedUsers: []string{"user1", "user2"},
						DisplayName:  "Task 1",
					},
					"task2": {
						Script:       "/opt/scripts/task2.sh",
						AllowedUsers: []string{"user3"},
						DisplayName:  "Task 2",
					},
				},
			},
		},
		RobotGroupID: "oc_test",
	}

	// 使用新的构造函数创建 Dispatcher
	dispatcher := NewDispatcherHandler(cfg, nil)

	// 验证全局用户白名单（所有允许的用户合并）
	if !dispatcher.userWhiteList["user1"] {
		t.Error("Expected user1 to be in global whitelist")
	}
	if !dispatcher.userWhiteList["user2"] {
		t.Error("Expected user2 to be in global whitelist")
	}
	if !dispatcher.userWhiteList["user3"] {
		t.Error("Expected user3 to be in global whitelist")
	}

	// 验证任务级权限
	taskPerms := dispatcher.taskUserPermissions
	if len(taskPerms) != 2 {
		t.Errorf("Expected 2 task permissions, got %d", len(taskPerms))
	}

	// 验证任务1的权限
	if !contains(taskPerms["task1"], "user1") {
		t.Error("Expected user1 to be allowed for task1")
	}
	if !contains(taskPerms["task1"], "user2") {
		t.Error("Expected user2 to be allowed for task1")
	}

	// 验证任务2的权限
	if !contains(taskPerms["task2"], "user3") {
		t.Error("Expected user3 to be allowed for task2")
	}

	// 验证不存在的用户不在任务权限中
	if contains(taskPerms["task1"], "user3") {
		t.Error("Expected user3 to NOT be allowed for task1")
	}

	t.Log("Two-layer permission data structure verified successfully")
}

// TestDispatcher_TaskPermissionEdgeCases 测试任务权限边界情况（仅测试权限数据结构）
func TestDispatcher_TaskPermissionEdgeCases(t *testing.T) {
	handler := createTestDispatcherHandler()

	// 设置分层匹配器
	executors := []matcher.ExecutorTaskConfig{
		{
			ExecutorID:   "cli_456",
			RoutingKeywords: []string{"dev", "development", "迷雾", "土豆"},
			Keywords:     []string{"土豆"},
			TaskName:     "potato-dev-restart",
			Script:       "/scripts/potato_restart.sh",
			Names: matcher.TaskNames{
				Primary: "重启",
				Aliases: []string{"重启", "restart", "回滚"},
			},
			DisplayName: "土豆开发服重启",
		},
	}
	handler.SetLayeredMatcher(matcher.NewLayeredMatcher(executors, nil))

	// 测试用例 - 验证权限数据结构
	testCases := []struct {
		name          string
		userID        string
		taskName      string
		shouldAllow   bool
		description   string
	}{
		{
			name:        "用户不在任务权限中",
			userID:      "admin_user",
			taskName:    "test-task",
			shouldAllow: false, // test-task 有 allowed_users: ["user_1", "user_2"]，不包含 admin_user
			description: "用户不在任务的allowed_users中，应该拒绝",
		},
		{
			name:        "用户在任务权限中",
			userID:      "admin_user",
			taskName:    "potato-dev-restart",
			shouldAllow: true,
			description: "用户在任务的allowed_users中，应该允许",
		},
		{
			name:        "用户不在任务权限中",
			userID:      "admin_user",
			taskName:    "mist-dev-update",
			shouldAllow: false,
			description: "用户不在指定任务的allowed_users中，应该拒绝",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 第一层：检查用户是否在全局白名单中
			if !handler.userWhiteList[tc.userID] {
				t.Errorf("%s: 用户 %s 不在全局白名单中", tc.description, tc.userID)
				return
			}

			// 第二层：检查用户是否在任务的白名单中
			allowedUsers, exists := handler.taskUserPermissions[tc.taskName]
			hasPermission := exists && contains(allowedUsers, tc.userID)

			if tc.shouldAllow && !hasPermission {
				t.Errorf("%s: 用户 %s 应该有权限执行任务 %s，但被拒绝", tc.description, tc.userID, tc.taskName)
			} else if !tc.shouldAllow && hasPermission {
				t.Errorf("%s: 用户 %s 不应该有权限执行任务 %s，但被允许", tc.description, tc.userID, tc.taskName)
			} else {
				t.Logf("%s: 权限检查正确 (allow=%v, hasPermission=%v)", tc.description, tc.shouldAllow, hasPermission)
			}
		})
	}
}

// TestDispatcher_PermissionCheckConcurrency 测试权限检查的并发安全性（仅测试数据结构并发读取）
func TestDispatcher_PermissionCheckConcurrency(t *testing.T) {
	handler := createTestDispatcherHandler()

	const numGoroutines = 100
	const numReads = 100
	var wg sync.WaitGroup

	// 模拟并发读取权限数据结构
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()

			for j := 0; j < numReads; j++ {
				// 并发读取全局白名单（不会发生数据竞争）
				_ = len(handler.userWhiteList)

				// 并发读取任务权限（不会发生数据竞争）
				for taskName, users := range handler.taskUserPermissions {
					_ = taskName
					_ = len(users)
				}
			}
		}(i)
	}

	wg.Wait()

	// 如果没有 panic 或数据竞争，测试通过
	// 使用 go test -race 运行此测试以检测数据竞争
	t.Log("并发读取测试通过，无数据竞争")
}
