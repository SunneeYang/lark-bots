package handler

import (
	"encoding/json"
	"fmt"
	"strings"
)

// TaskCommand 任务命令结构（Dispatcher → Executor 通信格式）
// 格式: task:potato-dev-restart（只发送唯一任务名称）
// 将迁移到 JSON 格式以支持更多元数据（如发布者信息）
type TaskCommand struct {
	TaskName  string // 唯一任务名称（全局唯一）
	Requester string // 发布者姓名（谁发起的任务）
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

// JSON 序列化为 JSON 格式
func (c *TaskCommand) JSON() string {
	data := map[string]interface{}{
		"task":      c.TaskName,
		"requester": c.Requester,
	}
	bytes, _ := json.Marshal(data)
	return string(bytes)
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
