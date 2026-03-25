package handler

import (
	"fmt"
	"strings"
)

// TaskCommand 任务命令结构（Dispatcher → Executor 通信格式）
// 格式: task:potato-dev-restart（只发送唯一任务名称）
type TaskCommand struct {
	TaskName string // 唯一任务名称（全局唯一）
}

// BuildTaskCommand 构建任务命令（只包含任务名）
func BuildTaskCommand(taskName string) *TaskCommand {
	return &TaskCommand{
		TaskName: taskName,
	}
}

// String 序列化为字符串格式（发送到群消息）
func (c *TaskCommand) String() string {
	// 格式: task:potato-dev-restart
	return fmt.Sprintf("task:%s", c.TaskName)
}

// ParseTaskCommand 从字符串解析任务命令
// 输入格式: task:potato-dev-restart
func ParseTaskCommand(input string) (*TaskCommand, error) {
	// 检查是否是 task: 前缀
	if !strings.HasPrefix(input, "task:") {
		return nil, fmt.Errorf("无效的任务命令格式: %s", input)
	}

	// 提取任务名称
	taskName := strings.TrimPrefix(input, "task:")
	taskName = strings.TrimSpace(taskName)

	if taskName == "" {
		return nil, fmt.Errorf("任务名称为空: %s", input)
	}

	return &TaskCommand{
		TaskName: taskName,
	}, nil
}
