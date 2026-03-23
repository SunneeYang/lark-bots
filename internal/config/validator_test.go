package config

import (
	"strings"
	"testing"
)

func TestValidateConfig_Valid(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:              "executor",
				AppID:             "cli_456",
				AppSecret:         "secret",
				Role:              "executor",
				AllowedDispatchers: []string{"cli_123"},
			},
		},
		RobotGroupID:  "oc_test",
		TaskWhiteList: []string{"deploy"},
		UserWhiteList: []string{"user_1"},
	}

	err := ValidateConfig(cfg)
	if err != nil {
		t.Errorf("Expected valid config, got error: %v", err)
	}
}

func TestValidateConfig_NoDispatcher(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "executor",
				AppID:     "cli_456",
				AppSecret: "secret",
				Role:      "executor",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing dispatcher, got nil")
	}

	if !strings.Contains(err.Error(), "至少需要一个") {
		t.Errorf("Error message should mention dispatcher requirement, got: %v", err)
	}
}

func TestValidateConfig_InvalidDispatcherReference(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:              "executor",
				AppID:             "cli_456",
				AppSecret:         "secret",
				Role:              "executor",
				AllowedDispatchers: []string{"cli_nonexistent"}, // 不存在的 dispatcher
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid dispatcher reference, got nil")
	}

	if !strings.Contains(err.Error(), "不存在的") {
		t.Errorf("Error should mention nonexistent dispatcher, got: %v", err)
	}
}

func TestValidateConfig_DuplicateBotNames(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "duplicate",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
			{
				Name:      "duplicate", // 重复名称
				AppID:     "cli_456",
				AppSecret: "secret",
				Role:      "executor",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for duplicate bot names, got nil")
	}

	if !strings.Contains(err.Error(), "重复") {
		t.Errorf("Error should mention duplicate name, got: %v", err)
	}
}

func TestValidateConfig_MissingBotName(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing bot name, got nil")
	}

	if !strings.Contains(err.Error(), "缺少名称") {
		t.Errorf("Error should mention missing name, got: %v", err)
	}
}

func TestValidateConfig_MissingAppID(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "test-bot",
				AppID:     "",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing app_id, got nil")
	}

	if !strings.Contains(err.Error(), "缺少 app_id") {
		t.Errorf("Error should mention missing app_id, got: %v", err)
	}
}

func TestValidateConfig_MissingAppSecret(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "test-bot",
				AppID:     "cli_123",
				AppSecret: "",
				Role:      "dispatcher",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing app_secret, got nil")
	}

	if !strings.Contains(err.Error(), "缺少 app_secret") {
		t.Errorf("Error should mention missing app_secret, got: %v", err)
	}
}

func TestValidateConfig_InvalidRole(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "test-bot",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "invalid_role",
			},
		},
		RobotGroupID: "oc_test",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for invalid role, got nil")
	}

	if !strings.Contains(err.Error(), "角色") {
		t.Errorf("Error should mention invalid role, got: %v", err)
	}
}

func TestValidateConfig_MissingRobotGroupID(t *testing.T) {
	cfg := &ServiceConfig{
		Bots: []BotConfig{
			{
				Name:      "dispatcher",
				AppID:     "cli_123",
				AppSecret: "secret",
				Role:      "dispatcher",
			},
		},
		RobotGroupID: "",
	}

	err := ValidateConfig(cfg)
	if err == nil {
		t.Error("Expected error for missing robot_group_id, got nil")
	}

	if !strings.Contains(err.Error(), "robot_group_id") {
		t.Errorf("Error should mention robot_group_id, got: %v", err)
	}
}
