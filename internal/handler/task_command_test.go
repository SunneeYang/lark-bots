package handler

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestBuildTaskCommand 测试构建任务命令
func TestBuildTaskCommand(t *testing.T) {
	cmd := BuildTaskCommand("potato-dev-restart")

	if cmd.TaskName != "potato-dev-restart" {
		t.Errorf("TaskName = %v, want %v", cmd.TaskName, "potato-dev-restart")
	}
}

// TestTaskCommand_String 测试序列化任务命令
func TestTaskCommand_String(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *TaskCommand
		wantStr string
	}{
		{
			name: "标准格式",
			cmd: &TaskCommand{
				TaskName: "potato-dev-restart",
			},
			wantStr: "task:potato-dev-restart",
		},
		{
			name: "包含连字符",
			cmd: &TaskCommand{
				TaskName: "mist-test-update",
			},
			wantStr: "task:mist-test-update",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.String()
			if got != tt.wantStr {
				t.Errorf("String() = %v, want %v", got, tt.wantStr)
			}
		})
	}
}

// TestParseTaskCommand 测试解析任务命令
func TestParseTaskCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantTask string
		wantErr  bool
	}{
		{
			name:     "标准格式",
			input:    "task:potato-dev-restart",
			wantTask: "potato-dev-restart",
			wantErr:  false,
		},
		{
			name:     "包含连字符",
			input:    "task:mist-test-update",
			wantTask: "mist-test-update",
			wantErr:  false,
		},
		{
			name:    "无task前缀",
			input:   "potato-dev-restart",
			wantErr: true,
		},
		{
			name:    "空任务名",
			input:   "task:",
			wantErr: true,
		},
		{
			name:    "只有前缀",
			input:   "task:",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskCommand(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.TaskName != tt.wantTask {
				t.Errorf("TaskName = %v, want %v", got.TaskName, tt.wantTask)
			}
		})
	}
}

// TestTaskCommand_RoundTrip 测试序列化与反序列化的往返
func TestTaskCommand_RoundTrip(t *testing.T) {
	originalTaskName := "potato-dev-restart"

	// 序列化
	serialized := BuildTaskCommand(originalTaskName).String()

	// 反序列化
	parsedCmd, err := ParseTaskCommand(serialized)
	if err != nil {
		t.Fatalf("ParseTaskCommand() error = %v", err)
	}

	// 验证
	if parsedCmd.TaskName != originalTaskName {
		t.Errorf("TaskName = %v, want %v", parsedCmd.TaskName, originalTaskName)
	}
}

// TestTaskCommand_JSON 测试 JSON 序列化
func TestTaskCommand_JSON(t *testing.T) {
	tests := []struct {
		name     string
		cmd      *TaskCommand
		wantTask string
		wantReq  string
	}{
		{
			name: "标准格式",
			cmd: &TaskCommand{
				TaskName:  "miwu-dev-restart",
				Requester: "张三",
			},
			wantTask: "miwu-dev-restart",
			wantReq:  "张三",
		},
		{
			name: "英文名",
			cmd: &TaskCommand{
				TaskName:  "potato-test-update",
				Requester: "John Doe",
			},
			wantTask: "potato-test-update",
			wantReq:  "John Doe",
		},
		{
			name: "空发布者",
			cmd: &TaskCommand{
				TaskName:  "mist-deploy",
				Requester: "",
			},
			wantTask: "mist-deploy",
			wantReq:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cmd.JSON()

			// 解析 JSON 以验证格式
			var data map[string]interface{}
			if err := json.Unmarshal([]byte(got), &data); err != nil {
				t.Fatalf("JSON() produced invalid JSON: %v", err)
			}

			// 验证字段存在
			if task, ok := data["task"].(string); !ok || task != tt.wantTask {
				t.Errorf("task field = %v (type %T), want %v", data["task"], data["task"], tt.wantTask)
			}

			if requester, ok := data["requester"].(string); !ok || requester != tt.wantReq {
				t.Errorf("requester field = %v (type %T), want %v", data["requester"], data["requester"], tt.wantReq)
			}
		})
	}
}

// TestTaskCommand_JSON_Format 测试 JSON 格式正确性
func TestTaskCommand_JSON_Format(t *testing.T) {
	cmd := &TaskCommand{
		TaskName:  "test-task",
		Requester: "测试用户",
	}

	jsonStr := cmd.JSON()

	// 验证是有效的 JSON
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		t.Fatalf("Invalid JSON: %v", err)
	}

	// 验证包含必需字段
	if _, ok := data["task"]; !ok {
		t.Error("Missing 'task' field")
	}
	if _, ok := data["requester"]; !ok {
		t.Error("Missing 'requester' field")
	}

	// 验证没有多余字段（只有 task 和 requester）
	if len(data) != 2 {
		t.Errorf("Unexpected number of fields: got %d, want 2", len(data))
	}
}

// TestTaskCommand_JSON_Example 测试 JSON 输出示例
func TestTaskCommand_JSON_Example(t *testing.T) {
	cmd := &TaskCommand{
		TaskName:  "miwu-dev-restart",
		Requester: "张三",
	}

	jsonStr := cmd.JSON()

	// 验证包含预期的键和值
	if !strings.Contains(jsonStr, "miwu-dev-restart") {
		t.Error("JSON should contain task name")
	}
	if !strings.Contains(jsonStr, "张三") {
		t.Error("JSON should contain requester name")
	}
	if !strings.Contains(jsonStr, "task") {
		t.Error("JSON should contain 'task' key")
	}
	if !strings.Contains(jsonStr, "requester") {
		t.Error("JSON should contain 'requester' key")
	}
}

// TestParseTaskCommandJSON 测试从 JSON 字符串解析任务命令
func TestParseTaskCommandJSON(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantTask      string
		wantRequester string
		wantErr       bool
		errContains   string
	}{
		{
			name:          "标准JSON格式",
			input:         `{"task":"potato-dev-restart","requester":"张三"}`,
			wantTask:      "potato-dev-restart",
			wantRequester: "张三",
			wantErr:       false,
		},
		{
			name:          "英文名发布者",
			input:         `{"task":"mist-test-update","requester":"John Doe"}`,
			wantTask:      "mist-test-update",
			wantRequester: "John Doe",
			wantErr:       false,
		},
		{
			name:          "空发布者",
			input:         `{"task":"miwu-deploy","requester":""}`,
			wantTask:      "miwu-deploy",
			wantRequester: "",
			wantErr:       false,
		},
		{
			name:        "无效的JSON",
			input:       `{invalid json}`,
			wantErr:     true,
			errContains: "JSON 解析失败",
		},
		{
			name:        "缺少task字段",
			input:       `{"requester":"张三"}`,
			wantErr:     true,
			errContains: "缺少或无效的 task 字段",
		},
		{
			name:        "缺少requester字段",
			input:       `{"task":"potato-dev-restart"}`,
			wantErr:     true,
			errContains: "缺少或无效的 requester 字段",
		},
		{
			name:        "task字段为空字符串",
			input:       `{"task":"","requester":"张三"}`,
			wantErr:     true,
			errContains: "缺少或无效的 task 字段",
		},
		{
			name:        "task字段类型错误",
			input:       `{"task":123,"requester":"张三"}`,
			wantErr:     true,
			errContains: "缺少或无效的 task 字段",
		},
		{
			name:        "requester字段类型错误",
			input:       `{"task":"potato-dev-restart","requester":456}`,
			wantErr:     true,
			errContains: "缺少或无效的 requester 字段",
		},
		{
			name:        "空JSON对象",
			input:       `{}`,
			wantErr:     true,
			errContains: "缺少或无效的 task 字段",
		},
		{
			name:        "空字符串",
			input:       ``,
			wantErr:     true,
			errContains: "JSON 解析失败",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskCommandJSON(tt.input)

			if (err != nil) != tt.wantErr {
				t.Errorf("ParseTaskCommandJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error message = %v, want包含 %v", err.Error(), tt.errContains)
				}
				return
			}

			if got.TaskName != tt.wantTask {
				t.Errorf("TaskName = %v, want %v", got.TaskName, tt.wantTask)
			}

			if got.Requester != tt.wantRequester {
				t.Errorf("Requester = %v, want %v", got.Requester, tt.wantRequester)
			}
		})
	}
}

// TestParseTaskCommandJSON_RoundTrip 测试 JSON 序列化与反序列化的往返
func TestParseTaskCommandJSON_RoundTrip(t *testing.T) {
	original := &TaskCommand{
		TaskName:  "potato-dev-restart",
		Requester: "李四",
	}

	// 序列化
	jsonStr := original.JSON()

	// 反序列化
	parsed, err := ParseTaskCommandJSON(jsonStr)
	if err != nil {
		t.Fatalf("ParseTaskCommandJSON() error = %v", err)
	}

	// 验证
	if parsed.TaskName != original.TaskName {
		t.Errorf("TaskName = %v, want %v", parsed.TaskName, original.TaskName)
	}

	if parsed.Requester != original.Requester {
		t.Errorf("Requester = %v, want %v", parsed.Requester, original.Requester)
	}
}

// TestParseTaskCommandJSON_Whitespace 测试 JSON 格式中的空白字符处理
func TestParseTaskCommandJSON_Whitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "标准格式",
			input: `{"task":"test-task","requester":"user"}`,
		},
		{
			name:  "带空格",
			input: `{ "task" : "test-task" , "requester" : "user" }`,
		},
		{
			name: "带换行",
			input: `{
				"task": "test-task",
				"requester": "user"
			}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseTaskCommandJSON(tt.input)
			if err != nil {
				t.Fatalf("ParseTaskCommandJSON() error = %v", err)
			}

			if got.TaskName != "test-task" {
				t.Errorf("TaskName = %v, want test-task", got.TaskName)
			}

			if got.Requester != "user" {
				t.Errorf("Requester = %v, want user", got.Requester)
			}
		})
	}
}
