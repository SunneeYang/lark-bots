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
    allowed_users:
      - "user_1"
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

	if len(cfg.Bots[0].AllowedUsers) != 1 {
		t.Errorf("Expected 1 allowed user, got %d", len(cfg.Bots[0].AllowedUsers))
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
    allowed_users:
      - "user_1"
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
