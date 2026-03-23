package logger

import (
	"fmt"
	"testing"
	"time"

	"github.com/SunneeYang/lark-bots/internal/bot"
)

func TestTaskLogger_CreateTask(t *testing.T) {
	logger := NewTaskLogger()

	record := &bot.TaskRecord{
		ID:         "task-1",
		TaskName:   "deploy.sh",
		Dispatcher: "task-dispatcher",
		Executor:   "shell-executor-1",
		User:       "user_1",
		Status:     "pending",
		StartTime:  time.Now(),
	}

	err := logger.CreateTask(record)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	// 验证任务已创建
	retrieved, err := logger.GetTask("task-1")
	if err != nil {
		t.Fatalf("GetTask failed: %v", err)
	}

	if retrieved.TaskName != "deploy.sh" {
		t.Errorf("Expected task name 'deploy.sh', got '%s'", retrieved.TaskName)
	}
}

func TestTaskLogger_CreateTask_DuplicateID(t *testing.T) {
	logger := NewTaskLogger()

	record := &bot.TaskRecord{
		ID:        "task-dup",
		TaskName:  "test.sh",
		Status:    "pending",
		StartTime: time.Now(),
	}

	logger.CreateTask(record)

	// 尝试创建重复 ID
	err := logger.CreateTask(record)
	if err == nil {
		t.Error("Expected error for duplicate task ID, got nil")
	}
}

func TestTaskLogger_GetTask_NotFound(t *testing.T) {
	logger := NewTaskLogger()

	_, err := logger.GetTask("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent task, got nil")
	}
}

func TestTaskLogger_UpdateTaskStatus(t *testing.T) {
	logger := NewTaskLogger()

	record := &bot.TaskRecord{
		ID:        "task-2",
		TaskName:  "test.sh",
		Status:    "pending",
		StartTime: time.Now(),
	}

	logger.CreateTask(record)

	// 更新状态
	now := time.Now()
	err := logger.UpdateTaskStatus("task-2", "completed", &now, "执行成功", "")
	if err != nil {
		t.Fatalf("UpdateTaskStatus failed: %v", err)
	}

	// 验证更新
	retrieved, _ := logger.GetTask("task-2")
	if retrieved.Status != "completed" {
		t.Errorf("Expected status 'completed', got '%s'", retrieved.Status)
	}

	if retrieved.Result != "执行成功" {
		t.Errorf("Expected result '执行成功', got '%s'", retrieved.Result)
	}

	if retrieved.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestTaskLogger_UpdateTaskStatus_NotFound(t *testing.T) {
	logger := NewTaskLogger()

	now := time.Now()
	err := logger.UpdateTaskStatus("nonexistent", "completed", &now, "", "")
	if err == nil {
		t.Error("Expected error for nonexistent task, got nil")
	}
}

func TestTaskLogger_QueryTasks(t *testing.T) {
	logger := NewTaskLogger()

	// 创建多个任务
	for i := 0; i < 3; i++ {
		record := &bot.TaskRecord{
			ID:         fmt.Sprintf("task-%d", i),
			TaskName:   "test.sh",
			Status:     "completed",
			StartTime:  time.Now(),
			Dispatcher: "task-dispatcher",
			Executor:   "executor-1",
			User:       "user_1",
		}
		logger.CreateTask(record)
	}

	// 添加一个不同状态的任务
	record := &bot.TaskRecord{
		ID:         "task-pending",
		TaskName:   "test.sh",
		Status:     "pending",
		StartTime:  time.Now(),
		Dispatcher: "task-dispatcher",
		Executor:   "executor-1",
		User:       "user_1",
	}
	logger.CreateTask(record)

	// 查询已完成任务
	tasks, err := logger.QueryTasks(TaskFilter{Status: "completed"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 completed tasks, got %d", len(tasks))
	}

	// 验证所有返回的任务都是已完成状态
	for _, task := range tasks {
		if task.Status != "completed" {
			t.Errorf("Expected status 'completed', got '%s'", task.Status)
		}
	}
}

func TestTaskLogger_QueryTasks_ByUser(t *testing.T) {
	logger := NewTaskLogger()

	// 创建不同用户的任务
	record1 := &bot.TaskRecord{
		ID:         "task-user1",
		TaskName:   "test.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		User:       "user_1",
	}
	logger.CreateTask(record1)

	record2 := &bot.TaskRecord{
		ID:         "task-user2",
		TaskName:   "test.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		User:       "user_2",
	}
	logger.CreateTask(record2)

	// 查询 user_1 的任务
	tasks, err := logger.QueryTasks(TaskFilter{User: "user_1"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task for user_1, got %d", len(tasks))
	}

	if tasks[0].User != "user_1" {
		t.Errorf("Expected user 'user_1', got '%s'", tasks[0].User)
	}
}

func TestTaskLogger_QueryTasks_ByExecutor(t *testing.T) {
	logger := NewTaskLogger()

	// 创建不同执行器的任务
	record1 := &bot.TaskRecord{
		ID:         "task-exec1",
		TaskName:   "test.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		Executor:   "executor-1",
	}
	logger.CreateTask(record1)

	record2 := &bot.TaskRecord{
		ID:         "task-exec2",
		TaskName:   "test.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		Executor:   "executor-2",
	}
	logger.CreateTask(record2)

	// 查询 executor-1 的任务
	tasks, err := logger.QueryTasks(TaskFilter{Executor: "executor-1"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task for executor-1, got %d", len(tasks))
	}

	if tasks[0].Executor != "executor-1" {
		t.Errorf("Expected executor 'executor-1', got '%s'", tasks[0].Executor)
	}
}

func TestTaskLogger_QueryTasks_ByTaskName(t *testing.T) {
	logger := NewTaskLogger()

	// 创建不同任务名的任务
	record1 := &bot.TaskRecord{
		ID:         "task-deploy",
		TaskName:   "deploy.sh",
		Status:     "completed",
		StartTime:  time.Now(),
	}
	logger.CreateTask(record1)

	record2 := &bot.TaskRecord{
		ID:         "task-restart",
		TaskName:   "restart.sh",
		Status:     "completed",
		StartTime:  time.Now(),
	}
	logger.CreateTask(record2)

	// 查询 deploy.sh 任务
	tasks, err := logger.QueryTasks(TaskFilter{TaskName: "deploy.sh"})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task for deploy.sh, got %d", len(tasks))
	}

	if tasks[0].TaskName != "deploy.sh" {
		t.Errorf("Expected task name 'deploy.sh', got '%s'", tasks[0].TaskName)
	}
}

func TestTaskLogger_QueryTasks_MultipleFilters(t *testing.T) {
	logger := NewTaskLogger()

	// 创建多个任务
	record1 := &bot.TaskRecord{
		ID:         "task-match",
		TaskName:   "deploy.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		User:       "user_1",
		Executor:   "executor-1",
	}
	logger.CreateTask(record1)

	record2 := &bot.TaskRecord{
		ID:         "task-nomatch-status",
		TaskName:   "deploy.sh",
		Status:     "pending",
		StartTime:  time.Now(),
		User:       "user_1",
		Executor:   "executor-1",
	}
	logger.CreateTask(record2)

	record3 := &bot.TaskRecord{
		ID:         "task-nomatch-user",
		TaskName:   "deploy.sh",
		Status:     "completed",
		StartTime:  time.Now(),
		User:       "user_2",
		Executor:   "executor-1",
	}
	logger.CreateTask(record3)

	// 查询：completed + user_1 + deploy.sh
	tasks, err := logger.QueryTasks(TaskFilter{
		TaskName: "deploy.sh",
		Status:   "completed",
		User:     "user_1",
	})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task matching all filters, got %d", len(tasks))
	}

	if tasks[0].ID != "task-match" {
		t.Errorf("Expected task 'task-match', got '%s'", tasks[0].ID)
	}
}

func TestTaskLogger_QueryTasks_EmptyFilter(t *testing.T) {
	logger := NewTaskLogger()

	// 创建任务
	for i := 0; i < 3; i++ {
		record := &bot.TaskRecord{
			ID:         fmt.Sprintf("task-%d", i),
			TaskName:   "test.sh",
			Status:     "completed",
			StartTime:  time.Now(),
		}
		logger.CreateTask(record)
	}

	// 空过滤器应返回所有任务
	tasks, err := logger.QueryTasks(TaskFilter{})
	if err != nil {
		t.Fatalf("QueryTasks failed: %v", err)
	}

	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks with empty filter, got %d", len(tasks))
	}
}
