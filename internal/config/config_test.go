package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
	"testing"
)

func TestBotConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
name: "test-bot"
app_id: "cli_123"
app_secret: "secret"
role: "executor"
allowed_dispatchers:
  - "cli_456"
task_scripts:
  test_script: "/opt/scripts/test.sh"
`

	var cfg BotConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if cfg.Name != "test-bot" {
		t.Errorf("Expected Name 'test-bot', got '%s'", cfg.Name)
	}

	if cfg.Role != "executor" {
		t.Errorf("Expected Role 'executor', got '%s'", cfg.Role)
	}

	if len(cfg.AllowedDispatchers) != 1 {
		t.Errorf("Expected 1 allowed dispatcher, got %d", len(cfg.AllowedDispatchers))
	}

	if len(cfg.TaskScripts) != 1 {
		t.Errorf("Expected 1 task script, got %d", len(cfg.TaskScripts))
	}
}

func TestServiceConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
bots:
  - name: "dispatcher"
    app_id: "cli_123"
    app_secret: "secret"
    role: "dispatcher"
    allowed_tasks:
      - "deploy"
      - "restart"
robot_group_id: "oc_xxx"
`

	var cfg ServiceConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if len(cfg.Bots) != 1 {
		t.Errorf("Expected 1 bot, got %d", len(cfg.Bots))
	}

	if cfg.RobotGroupID != "oc_xxx" {
		t.Errorf("Expected RobotGroupID 'oc_xxx', got '%s'", cfg.RobotGroupID)
	}
}

func TestLoadConfig_Success(t *testing.T) {
	// Create temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	yamlContent := `
bots:
  - name: "dispatcher"
    app_id: "cli_123"
    app_secret: "secret"
    role: "dispatcher"
  - name: "executor"
    app_id: "cli_456"
    app_secret: "secret2"
    role: "executor"
    allowed_dispatchers:
      - "cli_123"
    task_scripts:
      deploy: "/opt/scripts/deploy.sh"
robot_group_id: "oc_xxx"
`

	err := os.WriteFile(configPath, []byte(yamlContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	// Load config
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config
	if len(cfg.Bots) != 2 {
		t.Errorf("Expected 2 bots, got %d", len(cfg.Bots))
	}

	if cfg.Bots[0].Name != "dispatcher" {
		t.Errorf("Expected first bot name 'dispatcher', got '%s'", cfg.Bots[0].Name)
	}

	if cfg.Bots[1].Role != "executor" {
		t.Errorf("Expected second bot role 'executor', got '%s'", cfg.Bots[1].Role)
	}

	if cfg.RobotGroupID != "oc_xxx" {
		t.Errorf("Expected RobotGroupID 'oc_xxx', got '%s'", cfg.RobotGroupID)
	}
}

func TestLoadConfig_FileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for nonexistent file, got nil")
	}

	if !errors.Is(err, ErrConfigNotFound) {
		t.Errorf("Expected ErrConfigNotFound, got %v", err)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	// Write invalid YAML
	invalidYAML := `
bots:
  - name: "test"
    app_id: "cli_123
invalid yaml content [[[
`
	err := os.WriteFile(configPath, []byte(invalidYAML), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if !errors.Is(err, ErrInvalidYAML) {
		t.Errorf("Expected ErrInvalidYAML, got %v", err)
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.yaml")

	err := os.WriteFile(configPath, []byte(""), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for empty file, got nil")
	}
}

func TestBotConfig_TaskDetails(t *testing.T) {
	yamlData := `
name: "test-executor"
app_id: "cli_123"
app_secret: "secret"
role: "executor"
allowed_dispatchers:
  - "cli_456"
tasks:
  miwu-dev-restart:
    script: "/opt/scripts/build.sh"
    params: ["dev", "zh"]
  potato-prod-deploy:
    script: "/opt/scripts/deploy.sh"
    params: ["prod", "us"]
  check-logs:
    script: "/opt/scripts/check_logs.sh"
    params: []
`

	var cfg BotConfig
	err := yaml.Unmarshal([]byte(yamlData), &cfg)

	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Verify basic fields
	if cfg.Name != "test-executor" {
		t.Errorf("Expected Name 'test-executor', got '%s'", cfg.Name)
	}

	if cfg.Role != "executor" {
		t.Errorf("Expected Role 'executor', got '%s'", cfg.Role)
	}

	// Verify Tasks field
	if cfg.Tasks == nil {
		t.Fatal("Tasks should not be nil")
	}

	if len(cfg.Tasks) != 3 {
		t.Errorf("Expected 3 task details, got %d", len(cfg.Tasks))
	}

	// Verify first task
	task1, ok := cfg.Tasks["miwu-dev-restart"]
	if !ok {
		t.Error("Task 'miwu-dev-restart' not found in Tasks")
	} else {
		if task1.Script != "/opt/scripts/build.sh" {
			t.Errorf("Expected script '/opt/scripts/build.sh', got '%s'", task1.Script)
		}
		if len(task1.Params) != 2 {
			t.Errorf("Expected 2 params, got %d", len(task1.Params))
		} else {
			if task1.Params[0] != "dev" || task1.Params[1] != "zh" {
				t.Errorf("Expected params ['dev', 'zh'], got %v", task1.Params)
			}
		}
	}

	// Verify second task
	task2, ok := cfg.Tasks["potato-prod-deploy"]
	if !ok {
		t.Error("Task 'potato-prod-deploy' not found in Tasks")
	} else {
		if task2.Script != "/opt/scripts/deploy.sh" {
			t.Errorf("Expected script '/opt/scripts/deploy.sh', got '%s'", task2.Script)
		}
		if len(task2.Params) != 2 {
			t.Errorf("Expected 2 params, got %d", len(task2.Params))
		} else {
			if task2.Params[0] != "prod" || task2.Params[1] != "us" {
				t.Errorf("Expected params ['prod', 'us'], got %v", task2.Params)
			}
		}
	}

	// Verify third task with empty params
	task3, ok := cfg.Tasks["check-logs"]
	if !ok {
		t.Error("Task 'check-logs' not found in Tasks")
	} else {
		if task3.Script != "/opt/scripts/check_logs.sh" {
			t.Errorf("Expected script '/opt/scripts/check_logs.sh', got '%s'", task3.Script)
		}
		if len(task3.Params) != 0 {
			t.Errorf("Expected 0 params, got %d", len(task3.Params))
		}
	}
}
