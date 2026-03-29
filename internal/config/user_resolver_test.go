package config

import (
	"strings"
	"testing"
)

func TestResolveUsers_Success(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "ou_abc123",
			"李四": "ou_def456",
		},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{"张三", "李四"},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	task := cfg.Bots[0].Tasks["restart"]
	if len(task.AllowedUsers) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(task.AllowedUsers))
	}
	if task.AllowedUsers[0] != "ou_abc123" {
		t.Errorf("Expected ou_abc123, got %s", task.AllowedUsers[0])
	}
	if task.AllowedUsers[1] != "ou_def456" {
		t.Errorf("Expected ou_def456, got %s", task.AllowedUsers[1])
	}
}

func TestResolveUsers_NameNotFound(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "ou_abc123",
		},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{"张三", "王五"},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err == nil {
		t.Fatal("Expected error for unknown user name, got nil")
	}
	if !strings.Contains(err.Error(), "王五") {
		t.Errorf("Error should mention unknown user name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "not found in global users mapping") {
		t.Errorf("Error should mention global users mapping, got: %v", err)
	}
}

func TestResolveUsers_EmptyAllowedUsers(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "ou_abc123",
		},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err != nil {
		t.Fatalf("Expected no error for empty allowed_users, got: %v", err)
	}

	task := cfg.Bots[0].Tasks["restart"]
	if len(task.AllowedUsers) != 0 {
		t.Errorf("Expected 0 users, got %d", len(task.AllowedUsers))
	}
}

func TestResolveUsers_EmptyUsersMap(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{"张三"},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err == nil {
		t.Fatal("Expected error when users map is empty but allowed_users has names, got nil")
	}
}

func TestResolveUsers_NilUsersMap(t *testing.T) {
	cfg := &ServiceConfig{
		Users: nil,
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{"张三"},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err == nil {
		t.Fatal("Expected error when users map is nil but allowed_users has names, got nil")
	}
}

func TestResolveUsers_NoAllowedUsers(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "ou_abc123",
		},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"health-check": {
						Script: "/opt/scripts/health.sh",
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestResolveUsers_NoTasks(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "ou_abc123",
		},
		Bots: []BotConfig{
			{
				Name:        "executor-1",
				Role:        "executor",
				TaskScripts: map[string]string{"deploy": "/opt/deploy.sh"},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
}

func TestResolveUsers_EmptyOpenIDValue(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "",
		},
		Bots: []BotConfig{
			{
				Name: "executor-1",
				Role: "executor",
				Tasks: map[string]TaskDetail{
					"restart": {
						Script:       "/opt/scripts/restart.sh",
						DisplayName:  "重启服务",
						AllowedUsers: []string{"张三"},
					},
				},
			},
		},
	}

	err := ResolveUsers(cfg)
	if err == nil {
		t.Fatal("Expected error when user has empty open_id, got nil")
	}
	if !strings.Contains(err.Error(), "张三") {
		t.Errorf("Error should mention user name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "empty open_id") {
		t.Errorf("Error should mention empty open_id, got: %v", err)
	}
}
