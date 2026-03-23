package bot

import (
	"testing"
	"time"
)

func TestTaskRecord_StatusTransition(t *testing.T) {
	record := &TaskRecord{
		ID:        "task-1",
		Status:    "pending",
		StartTime: time.Now(),
	}

	// 测试状态转换
	if record.Status != "pending" {
		t.Errorf("Expected initial status 'pending', got '%s'", record.Status)
	}

	record.Status = "running"
	if record.Status != "running" {
		t.Errorf("Expected status 'running', got '%s'", record.Status)
	}

	now := time.Now()
	record.EndTime = &now
	record.Status = "completed"

	if record.Status != "completed" {
		t.Errorf("Expected final status 'completed', got '%s'", record.Status)
	}

	if record.EndTime == nil {
		t.Error("Expected EndTime to be set")
	}
}

func TestTaskRecord_AllFields(t *testing.T) {
	now := time.Now()
	endTime := now.Add(time.Hour)

	record := &TaskRecord{
		ID:         "task-2",
		TaskName:   "deploy.sh",
		Dispatcher: "task-dispatcher",
		Executor:   "shell-executor-1",
		User:       "user_123",
		Status:     "completed",
		StartTime:  now,
		EndTime:    &endTime,
		Result:     "Deployment successful",
		Error:      "",
	}

	if record.ID != "task-2" {
		t.Errorf("Expected ID 'task-2', got '%s'", record.ID)
	}

	if record.TaskName != "deploy.sh" {
		t.Errorf("Expected TaskName 'deploy.sh', got '%s'", record.TaskName)
	}

	if record.Dispatcher != "task-dispatcher" {
		t.Errorf("Expected Dispatcher 'task-dispatcher', got '%s'", record.Dispatcher)
	}

	if record.Executor != "shell-executor-1" {
		t.Errorf("Expected Executor 'shell-executor-1', got '%s'", record.Executor)
	}

	if record.User != "user_123" {
		t.Errorf("Expected User 'user_123', got '%s'", record.User)
	}

	if record.Status != "completed" {
		t.Errorf("Expected Status 'completed', got '%s'", record.Status)
	}

	if record.Result != "Deployment successful" {
		t.Errorf("Expected Result 'Deployment successful', got '%s'", record.Result)
	}

	if record.Error != "" {
		t.Errorf("Expected Error to be empty, got '%s'", record.Error)
	}
}

func TestTaskRecord_FailedStatus(t *testing.T) {
	record := &TaskRecord{
		ID:        "task-3",
		TaskName:  "test.sh",
		Status:    "failed",
		StartTime: time.Now(),
		Error:     "Script not found",
	}

	if record.Status != "failed" {
		t.Errorf("Expected Status 'failed', got '%s'", record.Status)
	}

	if record.Error != "Script not found" {
		t.Errorf("Expected Error 'Script not found', got '%s'", record.Error)
	}

	if record.EndTime != nil {
		t.Error("Expected EndTime to be nil for failed task without end time")
	}
}

func TestNewBotClient(t *testing.T) {
	bot := NewBotClient("test-bot", "cli_123", "secret123", "dispatcher")

	if bot == nil {
		t.Fatal("Expected non-nil BotClient")
	}

	if bot.Name != "test-bot" {
		t.Errorf("Expected Name 'test-bot', got '%s'", bot.Name)
	}

	if bot.AppID != "cli_123" {
		t.Errorf("Expected AppID 'cli_123', got '%s'", bot.AppID)
	}

	if bot.AppSecret != "secret123" {
		t.Errorf("Expected AppSecret 'secret123', got '%s'", bot.AppSecret)
	}

	if bot.Role != "dispatcher" {
		t.Errorf("Expected Role 'dispatcher', got '%s'", bot.Role)
	}
}

func TestBotClient_ExecutorConfig(t *testing.T) {
	bot := NewBotClient("executor-bot", "cli_456", "secret456", "executor")

	bot.AllowedDispatchers = []string{"cli_123", "cli_789"}
	bot.AllowedScripts = []string{"/opt/scripts/deploy.sh", "/opt/scripts/restart.sh"}

	if len(bot.AllowedDispatchers) != 2 {
		t.Errorf("Expected 2 allowed dispatchers, got %d", len(bot.AllowedDispatchers))
	}

	if len(bot.AllowedScripts) != 2 {
		t.Errorf("Expected 2 allowed scripts, got %d", len(bot.AllowedScripts))
	}

	if bot.AllowedDispatchers[0] != "cli_123" {
		t.Errorf("Expected first dispatcher 'cli_123', got '%s'", bot.AllowedDispatchers[0])
	}

	if bot.AllowedScripts[1] != "/opt/scripts/restart.sh" {
		t.Errorf("Expected second script '/opt/scripts/restart.sh', got '%s'", bot.AllowedScripts[1])
	}
}
