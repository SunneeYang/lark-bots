package config

import (
	"gopkg.in/yaml.v3"
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
allowed_scripts:
  - "/opt/scripts/test.sh"
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
}

func TestServiceConfig_UnmarshalYAML(t *testing.T) {
	yamlData := `
bots:
  - name: "dispatcher"
    app_id: "cli_123"
    app_secret: "secret"
    role: "dispatcher"
robot_group_id: "oc_xxx"
task_whitelist:
  - "deploy"
  - "restart"
user_whitelist:
  - "user_1"
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

	if len(cfg.TaskWhiteList) != 2 {
		t.Errorf("Expected 2 tasks in whitelist, got %d", len(cfg.TaskWhiteList))
	}
}
