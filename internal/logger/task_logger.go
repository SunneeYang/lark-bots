package logger

import (
	"fmt"
	"sync"
	"time"

	"github.com/yourname/lark-bot-service/internal/bot"
)

// TaskFilter 任务查询过滤器
type TaskFilter struct {
	TaskName string
	Status   string
	User     string
	Executor string
	// 可以添加更多过滤条件
}

// TaskLogger 任务日志记录器
type TaskLogger struct {
	mu    sync.RWMutex
	tasks map[string]*bot.TaskRecord // key: task_id
}

// NewTaskLogger 创建任务日志记录器
func NewTaskLogger() *TaskLogger {
	return &TaskLogger{
		tasks: make(map[string]*bot.TaskRecord),
	}
}

// CreateTask 创建任务记录
func (l *TaskLogger) CreateTask(record *bot.TaskRecord) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.tasks[record.ID]; exists {
		return fmt.Errorf("任务 ID 已存在: %s", record.ID)
	}

	l.tasks[record.ID] = record
	return nil
}

// GetTask 获取任务记录
func (l *TaskLogger) GetTask(taskID string) (*bot.TaskRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	record, exists := l.tasks[taskID]
	if !exists {
		return nil, fmt.Errorf("任务不存在: %s", taskID)
	}

	return record, nil
}

// UpdateTaskStatus 更新任务状态
func (l *TaskLogger) UpdateTaskStatus(taskID, status string, endTime *time.Time, result, errorMsg string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	record, exists := l.tasks[taskID]
	if !exists {
		return fmt.Errorf("任务不存在: %s", taskID)
	}

	record.Status = status
	record.EndTime = endTime
	record.Result = result
	record.Error = errorMsg

	return nil
}

// QueryTasks 查询任务记录
func (l *TaskLogger) QueryTasks(filter TaskFilter) ([]*bot.TaskRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	var results []*bot.TaskRecord

	for _, record := range l.tasks {
		if l.matchFilter(record, filter) {
			results = append(results, record)
		}
	}

	return results, nil
}

// matchFilter 检查任务是否匹配过滤器
func (l *TaskLogger) matchFilter(record *bot.TaskRecord, filter TaskFilter) bool {
	if filter.TaskName != "" && record.TaskName != filter.TaskName {
		return false
	}

	if filter.Status != "" && record.Status != filter.Status {
		return false
	}

	if filter.User != "" && record.User != filter.User {
		return false
	}

	if filter.Executor != "" && record.Executor != filter.Executor {
		return false
	}

	return true
}
