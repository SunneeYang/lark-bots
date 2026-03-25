package handler

import (
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
		name      string
		cmd       *TaskCommand
		wantStr   string
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
		name        string
		input       string
		wantTask    string
		wantErr     bool
	}{
		{
			name:     "标准格式",
			input:    "task:potato-dev-restart",
			wantTask: "potato-dev-restart",
			wantErr:   false,
		},
		{
			name:     "包含连字符",
			input:    "task:mist-test-update",
			wantTask: "mist-test-update",
			wantErr:   false,
		},
		{
			name:     "无task前缀",
			input:    "potato-dev-restart",
			wantErr:   true,
		},
		{
			name:     "空任务名",
			input:    "task:",
			wantErr:   true,
		},
		{
			name:     "只有前缀",
			input:    "task:",
			wantErr:   true,
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
