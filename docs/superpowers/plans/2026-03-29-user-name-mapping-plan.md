# 用户姓名映射表 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在配置文件中新增全局 `users` 映射表（姓名 → open_id），让 `allowed_users` 字段直接配置用户姓名，启动时自动解析为 open_id。

**Architecture:** 在 `ServiceConfig` 中新增 `Users map[string]string` 字段。新增 `ResolveUsers` 函数在启动时（`ValidateConfig` 之后）将所有 `TaskDetail.AllowedUsers` 中的姓名解析为 open_id。业务逻辑代码（dispatcher、executor）完全不变。

**Tech Stack:** Go, yaml.v3, 标准 testing 包

**Spec:** `docs/superpowers/specs/2026-03-29-user-name-mapping-design.md`

---

## File Structure

| 文件 | 操作 | 职责 |
|------|------|------|
| `internal/config/config.go` | 修改 | `ServiceConfig` 新增 `Users` 字段 |
| `internal/config/user_resolver.go` | 新增 | `ResolveUsers` 函数：将姓名解析为 open_id |
| `internal/config/user_resolver_test.go` | 新增 | `ResolveUsers` 的单元测试 |
| `cmd/bot-service/main.go` | 修改 | 在 `ValidateConfig` 后调用 `ResolveUsers` |
| `configs/bots.test.yaml` | 修改 | 添加 `users` 映射，迁移 `allowed_users` |
| `configs/bots.yaml.example` | 修改 | 添加 `users` 示例和注释 |

---

### Task 1: 新增 `Users` 字段到 `ServiceConfig`

**Files:**
- Modify: `internal/config/config.go:82-85`

- [ ] **Step 1: 修改 `ServiceConfig` 结构体**

在 `internal/config/config.go` 的 `ServiceConfig` 中新增 `Users` 字段：

```go
// ServiceConfig 定义服务配置
type ServiceConfig struct {
	Bots         []BotConfig       `yaml:"bots"`
	RobotGroupID string            `yaml:"robot_group_id"`
	Users        map[string]string `yaml:"users,omitempty"` // 姓名 → open_id 全局映射
}
```

- [ ] **Step 2: 验证现有测试通过**

Run: `go test ./internal/config/... -v`
Expected: 所有测试 PASS

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): 在 ServiceConfig 中新增 Users 姓名映射字段"
```

---

### Task 2: 实现 `ResolveUsers` 函数（TDD）

**Files:**
- Create: `internal/config/user_resolver.go`
- Create: `internal/config/user_resolver_test.go`

- [ ] **Step 1: 写失败测试 — 解析成功场景**

创建 `internal/config/user_resolver_test.go`：

```go
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
```

- [ ] **Step 2: 写失败测试 — 姓名不存在**

在 `user_resolver_test.go` 中追加：

```go
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
						AllowedUsers: []string{"张三", "王五"}, // 王五不在映射表中
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
```

- [ ] **Step 3: 写失败测试 — 空 allowed_users**

在 `user_resolver_test.go` 中追加：

```go
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
						AllowedUsers: []string{}, // 空 allowed_users
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
```

- [ ] **Step 4: 写失败测试 — 空 Users 映射表**

在 `user_resolver_test.go` 中追加：

```go
func TestResolveUsers_EmptyUsersMap(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{}, // 空映射表
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
```

- [ ] **Step 5: 写失败测试 — Users 为 nil**

在 `user_resolver_test.go` 中追加：

```go
func TestResolveUsers_NilUsersMap(t *testing.T) {
	cfg := &ServiceConfig{
		Users: nil, // nil 映射表
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
```

- [ ] **Step 6: 写失败测试 — 无 allowed_users 的任务被忽略**

在 `user_resolver_test.go` 中追加：

```go
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
						// 没有 AllowedUsers
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
```

- [ ] **Step 7: 写失败测试 — 没有 Tasks 的 executor 被忽略**

在 `user_resolver_test.go` 中追加：

```go
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
```

- [ ] **Step 8: 写失败测试 — users 映射表中 value 为空字符串**

在 `user_resolver_test.go` 中追加：

```go
func TestResolveUsers_EmptyOpenIDValue(t *testing.T) {
	cfg := &ServiceConfig{
		Users: map[string]string{
			"张三": "", // open_id 为空
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
```

- [ ] **Step 9: 运行测试确认全部失败**

Run: `go test ./internal/config/ -run TestResolveUsers -v`
Expected: 编译失败（`ResolveUsers` 未定义）

- [ ] **Step 10: 实现 `ResolveUsers` 函数**

创建 `internal/config/user_resolver.go`：

```go
package config

import "fmt"

// ResolveUsers 将所有 TaskDetail.AllowedUsers 中的姓名解析为 open_id
// 必须在 ValidateConfig 之后调用
func ResolveUsers(cfg *ServiceConfig) error {
	for i := range cfg.Bots {
		for taskName, task := range cfg.Bots[i].Tasks {
			if len(task.AllowedUsers) == 0 {
				continue
			}
			resolved := make([]string, 0, len(task.AllowedUsers))
			for _, name := range task.AllowedUsers {
				if cfg.Users == nil {
					return fmt.Errorf("bot %q task %q: user %q not found in global users mapping (users map is empty)",
						cfg.Bots[i].Name, taskName, name)
				}
				openID, ok := cfg.Users[name]
				if !ok {
					return fmt.Errorf("bot %q task %q: user %q not found in global users mapping",
						cfg.Bots[i].Name, taskName, name)
				}
				if openID == "" {
					return fmt.Errorf("bot %q task %q: user %q has empty open_id in global users mapping",
						cfg.Bots[i].Name, taskName, name)
				}
				resolved = append(resolved, openID)
			}
			task.AllowedUsers = resolved
			cfg.Bots[i].Tasks[taskName] = task
		}
	}
	return nil
}
```

- [ ] **Step 11: 运行测试确认全部通过**

Run: `go test ./internal/config/ -run TestResolveUsers -v`
Expected: 所有 8 个测试 PASS

- [ ] **Step 12: Commit**

```bash
git add internal/config/user_resolver.go internal/config/user_resolver_test.go
git commit -m "feat(config): 实现 ResolveUsers 姓名解析函数及测试"
```

---

### Task 3: 集成 `ResolveUsers` 到 `main.go`

**Files:**
- Modify: `cmd/bot-service/main.go:124-128`

- [ ] **Step 1: 在 `ValidateConfig` 之后添加 `ResolveUsers` 调用**

在 `cmd/bot-service/main.go` 中，在 `ValidateConfig` 成功后（第 128 行之后）添加：

```go
		if err := config.ResolveUsers(cfg); err != nil {
			fmt.Printf("❌ 用户名解析失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("✅ 用户名解析完成")
```

完整的调用顺序变为：
```go
	// 1. 加载配置
	cfg, err := config.LoadConfig(configPath)
	// ...
	if err := config.ValidateConfig(cfg); err != nil {
		fmt.Printf("❌ 配置验证失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 配置验证通过")

	// 新增：解析用户姓名映射
	if err := config.ResolveUsers(cfg); err != nil {
		fmt.Printf("❌ 用户名解析失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("✅ 用户名解析完成")

	// 2. 创建机器人注册表
```

- [ ] **Step 2: 验证编译通过**

Run: `go build ./cmd/bot-service/`
Expected: 编译成功，无错误

- [ ] **Step 3: Commit**

```bash
git add cmd/bot-service/main.go
git commit -m "feat: 在启动流程中集成 ResolveUsers 姓名解析"
```

---

### Task 4: 更新测试配置 `bots.test.yaml`

**Files:**
- Modify: `configs/bots.test.yaml`

- [ ] **Step 1: 添加 `users` 映射表**

在 `configs/bots.test.yaml` 文件顶部（`robot_group_id` 之前）添加：

```yaml
# 全局用户映射表：姓名 → open_id
users:
  测试用户1: test_user_001
  测试用户2: test_user_002
  管理员: admin_user
```

- [ ] **Step 2: 迁移 `test-dev-executor` 的 `allowed_users`**

将所有 `allowed_users` 从 open_id 改为姓名引用：

```yaml
    tasks:
      - name: "mist-dev-restart"
        # ...
        allowed_users:
          - "测试用户1"
      - name: "mist-dev-update"
        # ...
        allowed_users:
          - "测试用户2"
      - name: "potato-dev-restart"
        # ...
        allowed_users:
          - "管理员"
```

注意：`test-dispatcher` 的 `allowed_users` 已废弃（代码中被注释掉），保持不动。

- [ ] **Step 3: 运行全部测试验证**

Run: `go test ./... -v`
Expected: 所有测试 PASS

- [ ] **Step 4: Commit**

```bash
git add configs/bots.test.yaml
git commit -m "test(config): 迁移测试配置到姓名映射表"
```

---

### Task 5: 更新配置示例 `bots.yaml.example`

**Files:**
- Modify: `configs/bots.yaml.example`

- [ ] **Step 1: 在文件顶部添加 `users` 映射示例**

在 `robot_group_id` 之前添加：

```yaml
# 全局用户映射表：姓名 → open_id
# 用于 allowed_users 字段中用姓名代替 open_id，提升配置可读性
# allowed_users 中填写的姓名必须在此映射表中存在，否则启动时报错
users:
  # "张三": "ou_xxxxxxxxxxxxxxxxx"
  # "李四": "ou_yyyyyyyyyyyyyyyyyy"
```

- [ ] **Step 2: 更新 `allowed_users` 注释和示例**

将所有 `allowed_users` 中的 open_id 示例改为姓名引用：

```yaml
        allowed_users:
          - "张三"  # 在 users 映射表中注册的用户
```

同时更新底部「任务级用户权限」文档区域中的示例，将 `ou_user1` 等改为姓名。

- [ ] **Step 3: Commit**

```bash
git add configs/bots.yaml.example
git commit -m "docs(config): 更新配置示例，使用姓名映射表替代 open_id"
```

---

### Task 6: 最终验证

- [ ] **Step 1: 运行全部测试**

Run: `go test ./... -v`
Expected: 所有测试 PASS

- [ ] **Step 2: 运行覆盖率检查**

Run: `go test -cover ./internal/config/...`
Expected: 覆盖率不低于之前

- [ ] **Step 3: 代码格式化与检查**

Run: `go fmt ./... && go vet ./...`
Expected: 无输出（无格式或静态分析问题）
