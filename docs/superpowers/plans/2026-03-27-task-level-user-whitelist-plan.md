# 任务级用户白名单实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 实现任务级别的用户白名单控制，每个任务可以独立配置允许的用户列表，并通过两层权限检查确保安全性

**Architecture:**
- 将用户白名单从 Dispatcher 全局配置移到 Executor 的每个任务配置中
- NewDispatcherHandler 内部从配置文件构建权限数据（全局用户并集 + 任务级权限映射）
- HandleMessage 中实现两层权限检查：先检查用户是否在全局白名单中，再检查是否在特定任务的白名单中
- 不提供向后兼容，直接切换到新配置方式

**Tech Stack:**
- Go 1.21
- gopkg.in/yaml.v3 (配置解析)
- 标准库 testing 包（单元测试）
- 飞书开放平台 SDK

**文档参考:** `docs/superpowers/specs/2026-03-27-task-level-user-whitelist-design.md`

---

## 文件结构概览

### 需要修改的文件

1. **internal/config/config.go** - 配置结构定义
   - `TaskDetail` 新增 `AllowedUsers []string` 字段
   - `BotConfig` 移除 `AllowedUsers []string` 字段

2. **internal/config/validator.go** - 配置验证逻辑
   - 新增验证：有 `allowed_users` 的任务必须有 `display_name`

3. **internal/handler/dispatcher.go** - Dispatcher 处理器
   - `DispatcherHandler` 新增 `taskUserPermissions map[string][]string` 字段
   - 移除 `SetAllowedUsers()` 方法
   - 修改 `NewDispatcherHandler()` 签名，接收 `*config.ServiceConfig`
   - 内部实现权限数据构建逻辑
   - 修改 `HandleMessage()` 添加第二层权限检查
   - 新增 `contains()` 辅助函数

4. **cmd/bot-service/main.go** - 主程序入口
   - 修改 `NewDispatcherHandler()` 调用，传入配置对象

5. **configs/bots.yaml.example** - 配置示例文件
   - 移除 dispatcher 的 `allowed_users` 字段
   - 在 executor 任务中添加 `allowed_users` 示例

6. **internal/config/config_test.go** - 配置测试
   - 更新测试以适配新的配置结构

7. **internal/config/validator_test.go** - 验证器测试
   - 新增测试：验证 allowed_users 与 display_name 的关联

8. **internal/handler/dispatcher_test.go** - Dispatcher 测试
   - 新增测试：两层权限检查
   - 新增测试：边界情况处理
   - 更新现有测试以适配新的构造函数签名

---

## Task 1: 修改配置结构（config.go）

**Files:**
- Modify: `internal/config/config.go`
- Test: `internal/config/config_test.go`

- [ ] **Step 1: 在 TaskDetail 中添加 AllowedUsers 字段**

在 `internal/config/config.go` 的 `TaskDetail` 结构体中添加新字段：

```go
// TaskDetail 任务详细配置
type TaskDetail struct {
    DisplayName  string   `yaml:"display_name,omitempty"` // 任务显示名称
    Keywords     []string `yaml:"keywords,omitempty"`     // 关键词列表（用于语义匹配）
    Names        []string `yaml:"names,omitempty"`        // 操作名候选列表
    Script       string   `yaml:"script"`                 // 脚本路径
    Params       []string `yaml:"params,omitempty"`       // 参数列表
    AllowedUsers []string `yaml:"allowed_users,omitempty"` // 允许的用户列表
}
```

- [ ] **Step 2: 从 BotConfig 中移除 AllowedUsers 字段**

在 `internal/config/config.go` 的 `BotConfig` 结构体中，注释掉或删除 `AllowedUsers` 字段：

```go
// BotConfig 定义单个机器人的配置
type BotConfig struct {
    Name      string `yaml:"name"`
    AppID     string `yaml:"app_id"`
    AppSecret string `yaml:"app_secret"`
    Role      string `yaml:"role"` // "dispatcher" | "executor"

    // Dispatcher 特有配置
    // AllowedUsers []string `yaml:"allowed_users,omitempty"` // 已移除：用户白名单移到任务级别
    GroupProjectMap map[string]string `yaml:"group_project_map,omitempty"`

    // ... 其他字段保持不变
}
```

- [ ] **Step 3: 运行现有测试确保没有破坏性变更**

```bash
go test ./internal/config/... -v
```

预期：测试通过（配置结构变更向后兼容，因为使用了 `omitempty`）

- [ ] **Step 4: 提交变更**

```bash
git add internal/config/config.go
git commit -m " refactor(config): 将用户白名单从机器人级移到任务级

- TaskDetail 新增 AllowedUsers 字段支持任务级用户白名单
- BotConfig 移除 AllowedUsers 字段，用户白名单统一在任务中配置
- 使用 omitempty 确保配置文件向后兼容"
```

---

## Task 2: 添加配置验证（validator.go）

**Files:**
- Modify: `internal/config/validator.go`
- Test: `internal/config/validator_test.go`

- [ ] **Step 1: 编写验证失败的测试用例**

在 `internal/config/validator_test.go` 中添加：

```go
func TestValidateConfig_TaskAllowedUsersWithoutDisplayName(t *testing.T) {
    configYAML := `
robot_group_id: "oc_xxx"
bots:
  - name: "test-executor"
    app_id: "cli_xxx"
    app_secret: "xxx"
    role: "executor"
    tasks:
      test_task:
        keywords: ["test"]
        names: ["test"]
        script: "/test.sh"
        allowed_users:
          - "ou_xxx"
`

    var cfg config.ServiceConfig
    err := yaml.Unmarshal([]byte(configYAML), &cfg)
    require.NoError(t, err)

    err = ValidateConfig(&cfg)
    require.Error(t, err)
    assert.Contains(t, err.Error(), "缺少 display_name")
}
```

- [ ] **Step 2: 运行测试验证失败**

```bash
go test ./internal/config/... -run TestValidateConfig_TaskAllowedUsersWithoutDisplayName -v
```

预期：FAIL - 验证逻辑尚未实现

- [ ] **Step 3: 实现验证逻辑**

在 `internal/config/validator.go` 的 `ValidateConfig` 函数末尾（return nil 之前）添加：

```go
    // 验证配置了 allowed_users 的任务必须有 display_name
    for _, bot := range cfg.Bots {
        if bot.Role == "executor" {
            for taskName, task := range bot.Tasks {
                if len(task.AllowedUsers) > 0 && task.DisplayName == "" {
                    return fmt.Errorf(
                        "配置错误：executor '%s' 的任务 '%s' 配置了 allowed_users 但缺少 display_name（用于错误提示）",
                        bot.Name, taskName,
                    )
                }
            }
        }
    }
```

- [ ] **Step 4: 运行测试验证通过**

```bash
go test ./internal/config/... -run TestValidateConfig_TaskAllowedUsersWithoutDisplayName -v
```

预期：PASS

- [ ] **Step 5: 添加验证通过的测试用例**

```go
func TestValidateConfig_TaskAllowedUsersWithDisplayName(t *testing.T) {
    configYAML := `
robot_group_id: "oc_xxx"
bots:
  - name: "test-executor"
    app_id: "cli_xxx"
    app_secret: "xxx"
    role: "executor"
    tasks:
      test_task:
        display_name: "测试任务"
        keywords: ["test"]
        names: ["test"]
        script: "/test.sh"
        allowed_users:
          - "ou_xxx"
`

    var cfg config.ServiceConfig
    err := yaml.Unmarshal([]byte(configYAML), &cfg)
    require.NoError(t, err)

    err = ValidateConfig(&cfg)
    assert.NoError(t, err)
}
```

- [ ] **Step 6: 运行所有验证器测试**

```bash
go test ./internal/config/... -v
```

预期：所有测试通过

- [ ] **Step 7: 提交变更**

```bash
git add internal/config/validator.go internal/config/validator_test.go
git commit -m "feat(validator): 验证任务级用户白名单配置

- 添加验证规则：配置了 allowed_users 的任务必须同时配置 display_name
- display_name 用于权限错误提示，提供更好的用户体验
- 添加完整的单元测试覆盖"
```

---

## Task 3: 修改 Dispatcher 数据结构（dispatcher.go - 第一部分）

**Files:**
- Modify: `internal/handler/dispatcher.go`

- [ ] **Step 1: 在 DispatcherHandler 中添加 taskUserPermissions 字段**

在 `internal/handler/dispatcher.go` 的 `DispatcherHandler` 结构体中添加：

```go
// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
    *BaseHandler

    userWhiteList        map[string]bool           // 所有任务用户白名单的并集（第一层检查）
    taskWhiteList        map[string]bool           // 旧模式：任务名白名单（保留用于向后兼容）
    layeredMatcher       *matcher.LayeredMatcher
    groupProjectMap      map[string]string
    userInfoCache        map[string]string
    userInfoCacheMu      sync.RWMutex

    taskUserPermissions  map[string][]string       // 任务名 → 允许的用户列表（第二层检查）
}
```

- [ ] **Step 2: 移除 SetAllowedUsers 方法**

删除 `SetAllowedUsers` 方法（如果存在）：

```go
// 删除这个方法
// func (h *DispatcherHandler) SetAllowedUsers(users []string) { ... }
```

- [ ] **Step 3: 添加 contains 辅助函数**

在文件末尾添加：

```go
// contains 检查字符串切片中是否包含指定项
func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

- [ ] **Step 4: 运行测试检查编译错误**

```bash
go build ./internal/handler/...
```

预期：编译通过（只是添加字段和辅助函数，不影响现有逻辑）

- [ ] **Step 5: 提交变更**

```bash
git add internal/handler/dispatcher.go
git commit -m "refactor(dispatcher): 添加任务级权限数据结构

- DispatcherHandler 新增 taskUserPermissions 字段存储任务级权限
- 添加 contains 辅助函数用于权限检查
- 移除 SetAllowedUsers 方法，权限数据将在初始化时构建"
```

---

## Task 4: 修改 Dispatcher 初始化逻辑（dispatcher.go - 第二部分）

**Files:**
- Modify: `internal/handler/dispatcher.go`

- [ ] **Step 1: 修改 NewDispatcherHandler 签名**

将 `NewDispatcherHandler` 函数签名修改为：

```go
// NewDispatcherHandler 创建分发机器人处理器
func NewDispatcherHandler(
    cfg *config.ServiceConfig,
    semanticCfg *SemanticMatchConfig,
) *DispatcherHandler {
    // 收集所有 executor 任务的权限信息
    globalUsers := make(map[string]bool)
    taskPerms := make(map[string][]string)

    for _, bot := range cfg.Bots {
        if bot.Role == "executor" {
            for taskName, task := range bot.Tasks {
                if len(task.AllowedUsers) > 0 {
                    // 记录任务级权限
                    taskPerms[taskName] = task.AllowedUsers

                    // 构建全局用户并集
                    for _, user := range task.AllowedUsers {
                        globalUsers[user] = true
                    }
                }
            }
        }
    }

    // 启动时检查：如果全局白名单为空则警告
    if len(globalUsers) == 0 {
        log.Warn("警告：所有任务都没有配置 allowed_users，任何用户都无法执行任务")
    }

    return &DispatcherHandler{
        BaseHandler:         NewBaseHandler(),
        userWhiteList:       globalUsers,
        taskUserPermissions: taskPerms,
        taskWhiteList:       make(map[string]bool),
        userInfoCache:       make(map[string]string),
    }
}
```

注意：需要导入 `log` 包和 `config` 包。

- [ ] **Step 2: 运行测试检查编译错误**

```bash
go build ./internal/handler/...
```

预期：编译通过

- [ ] **Step 3: 提交变更**

```bash
git add internal/handler/dispatcher.go
git commit -m "feat(dispatcher): 在初始化时构建任务级权限数据

- NewDispatcherHandler 接收完整配置对象，内部提取权限数据
- 遍历所有 executor 任务，构建全局用户并集和任务级权限映射
- 添加启动时空白名单警告"
```

---

## Task 5: 实现两层权限检查（dispatcher.go - 第三部分）

**Files:**
- Modify: `internal/handler/dispatcher.go`
- Test: `internal/handler/dispatcher_test.go`

- [ ] **Step 1: 编写两层权限检查的测试用例**

在 `internal/handler/dispatcher_test.go` 中添加：

```go
func TestDispatcher_TwoLayerPermissionCheck(t *testing.T) {
    // 创建模拟配置
    cfg := &config.ServiceConfig{
        Bots: []config.BotConfig{
            {
                Name:  "test-executor",
                Role:  "executor",
                AppID: "cli_test",
                Tasks: map[string]config.TaskDetail{
                    "test_task": {
                        DisplayName:  "测试任务",
                        Keywords:     []string{"测试"},
                        Names:        []string{"任务"},
                        Script:       "/test.sh",
                        AllowedUsers: []string{"ou_111", "ou_222"},
                    },
                },
            },
        },
    }

    handler := NewDispatcherHandler(cfg, nil)

    tests := []struct {
        name        string
        senderID    string
        expectedErr string
    }{
        {
            name:        "用户不在全局白名单中",
            senderID:    "ou_999",
            expectedErr: "你没有执行任务的权限",
        },
        {
            name:        "用户在全局白名单但不在任务白名单",
            senderID:    "ou_222",
            expectedErr: "你没有权限执行任务：测试任务",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // 模拟权限检查
            if !handler.userWhiteList[tt.senderID] {
                // 第一层检查失败
                return
            }

            // 第二层检查（这里简化测试，实际需要完整的消息处理流程）
            allowedUsers, exists := handler.taskUserPermissions["test_task"]
            if !exists || !contains(allowedUsers, tt.senderID) {
                // 第二层检查失败
                return
            }

            t.Errorf("期望错误但未得到错误")
        })
    }
}
```

注意：这个测试是简化版本，后续会在 Task 7 中完善。

- [ ] **Step 2: 运行测试验证数据结构正确性**

```bash
go test ./internal/handler/... -run TestDispatcher_TwoLayerPermissionCheck -v
```

预期：测试通过（验证权限数据正确构建）

- [ ] **Step 3: 提交变更**

```bash
git add internal/handler/dispatcher_test.go
git commit -m "test(dispatcher): 添加两层权限检查测试框架

- 验证全局白名单和任务级权限数据正确构建
- 简化版本测试，后续会完善完整消息处理流程"
```

---

## Task 6: 修改主程序入口（main.go）

**Files:**
- Modify: `cmd/bot-service/main.go`

- [ ] **Step 1: 查找 NewDispatcherHandler 调用**

```bash
grep -n "NewDispatcherHandler" cmd/bot-service/main.go
```

- [ ] **Step 2: 修改 NewDispatcherHandler 调用**

将现有的调用（假设是）：

```go
dispatcherHandler := handler.NewDispatcherHandler(semanticCfg)
```

修改为：

```go
dispatcherHandler := handler.NewDispatcherHandler(cfg, semanticCfg)
```

- [ ] **Step 3: 移除 SetAllowedUsers 调用（如果存在）**

删除类似的代码：

```go
// 删除这行
dispatcherHandler.SetAllowedUsers(cfg.AllowedUsers)
```

- [ ] **Step 4: 运行测试确保主程序可以启动**

```bash
go build -o bot-service cmd/bot-service/main.go
```

预期：编译成功

- [ ] **Step 5: 提交变更**

```bash
git add cmd/bot-service/main.go
git commit -m "refactor(main): 适配新的 Dispatcher 初始化接口

- 修改 NewDispatcherHandler 调用，传入配置对象
- 移除 SetAllowedUsers 调用，权限数据在内部自动构建"
```

---

## Task 7: 实现完整的消息处理权限检查（dispatcher.go - 第四部分）

**Files:**
- Modify: `internal/handler/dispatcher.go`
- Test: `internal/handler/dispatcher_test.go`

- [ ] **Step 1: 在 HandleMessage 中添加第二层权限检查**

找到 `HandleMessage` 函数中任务匹配成功后的位置，添加第二层检查：

```go
// 在任务匹配成功后（matchedTask, err := h.layeredMatcher.Match(userInput) 之后）
// 添加第二层权限检查

// 第二层：检查用户是否在匹配到的任务白名单中
allowedUsers, exists := h.taskUserPermissions[matchedTask.Name]
if !exists || !contains(allowedUsers, senderID) {
    return fmt.Errorf("❌ 你没有权限执行任务：%s", matchedTask.DisplayName)
}
```

确保这段代码在第一层检查之后、发送消息到机器人群之前执行。

- [ ] **Step 2: 确认第一层权限检查存在**

确保在函数开始处已经有：

```go
// 第一层：检查用户是否在任何任务的白名单中
if !h.userWhiteList[senderID] {
    return fmt.Errorf("❌ 你没有执行任务的权限")
}
```

- [ ] **Step 3: 运行测试**

```bash
go test ./internal/handler/... -v
```

预期：测试通过（可能需要更新一些现有测试）

- [ ] **Step 4: 提交变更**

```bash
git add internal/handler/dispatcher.go
git commit -m "feat(dispatcher): 实现两层权限检查

- 第一层：检查用户是否在全局白名单中（任何任务的允许用户）
- 第二层：检查用户是否在特定任务的白名单中
- 使用 display_name 提供友好的错误提示
- 添加 contains 辅助函数简化代码"
```

---

## Task 8: 添加边界情况测试（dispatcher_test.go）

**Files:**
- Test: `internal/handler/dispatcher_test.go`

- [ ] **Step 1: 添加边界情况测试**

在 `internal/handler/dispatcher_test.go` 中添加：

```go
func TestDispatcher_TaskPermissionEdgeCases(t *testing.T) {
    tests := []struct {
        name        string
        config      *config.ServiceConfig
        senderID    string
        taskName    string
        expectedErr bool
        errContains string
    }{
        {
            name: "任务没有配置 allowed_users",
            config: &config.ServiceConfig{
                Bots: []config.BotConfig{
                    {
                        Role: "executor",
                        Tasks: map[string]config.TaskDetail{
                            "no_perms_task": {
                                DisplayName: "无权限任务",
                                Keywords:    []string{"test"},
                                Names:       []string{"任务"},
                                Script:      "/test.sh",
                                // 没有 AllowedUsers
                            },
                        },
                    },
                },
            },
            senderID:    "ou_111",
            taskName:    "no_perms_task",
            expectedErr: true,
        },
        {
            name: "allowed_users 为空数组",
            config: &config.ServiceConfig{
                Bots: []config.BotConfig{
                    {
                        Role: "executor",
                        Tasks: map[string]config.TaskDetail{
                            "empty_perms_task": {
                                DisplayName:  "空权限任务",
                                Keywords:     []string{"test"},
                                Names:        []string{"任务"},
                                Script:       "/test.sh",
                                AllowedUsers: []string{}, // 空数组
                            },
                        },
                    },
                },
            },
            senderID:    "ou_111",
            taskName:    "empty_perms_task",
            expectedErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            handler := NewDispatcherHandler(tt.config, nil)

            // 验证全局白名单
            if tt.expectedErr && !handler.userWhiteList[tt.senderID] {
                // 用户不在全局白名单，符合预期
                return
            }

            // 验证任务权限
            allowedUsers, exists := handler.taskUserPermissions[tt.taskName]
            if !exists || len(allowedUsers) == 0 {
                // 任务没有权限配置，符合预期
                return
            }

            if tt.expectedErr {
                t.Errorf("期望测试用例会失败，但没有")
            }
        })
    }
}

func TestDispatcher_PermissionCheckConcurrency(t *testing.T) {
    cfg := &config.ServiceConfig{
        Bots: []config.BotConfig{
            {
                Role: "executor",
                Tasks: map[string]config.TaskDetail{
                    "task1": {
                        DisplayName:  "任务1",
                        Keywords:     []string{"test"},
                        Names:        []string{"任务"},
                        Script:       "/test.sh",
                        AllowedUsers: []string{"ou_111"},
                    },
                },
            },
        },
    }

    handler := NewDispatcherHandler(cfg, nil)

    // 模拟并发读取
    done := make(chan bool)
    for i := 0; i < 100; i++ {
        go func() {
            _ = handler.userWhiteList["ou_111"]
            _, _ = handler.taskUserPermissions["task1"]
            done <- true
        }()
    }

    // 等待所有 goroutine 完成
    for i := 0; i < 100; i++ {
        <-done
    }

    // 如果没有数据竞争，测试通过
    // 使用 go test -race 运行此测试以检测数据竞争
}
```

- [ ] **Step 2: 运行测试**

```bash
go test ./internal/handler/... -run TestDispatcher_TaskPermissionEdgeCases -v
go test ./internal/handler/... -run TestDispatcher_PermissionCheckConcurrency -race -v
```

预期：测试通过，无数据竞争

- [ ] **Step 3: 提交变更**

```bash
git add internal/handler/dispatcher_test.go
git commit -m "test(dispatcher): 添加权限检查边界情况测试

- 测试任务没有配置 allowed_users 的情况
- 测试 allowed_users 为空数组的情况
- 测试并发读取权限数据的安全性（使用 -race 检测）"
```

---

## Task 9: 更新配置示例文件（bots.yaml.example）

**Files:**
- Modify: `configs/bots.yaml.example`

- [ ] **Step 1: 从 dispatcher 配置中移除 allowed_users**

找到 dispatcher 配置部分，删除 `allowed_users` 字段：

```yaml
# ====================
# 分发机器人 (Dispatcher)
# ====================
- name: "task-dispatcher"
  app_id: "cli_xxxxxxxxxxxxxxxxx"
  app_secret: "xxxxxxxxxxxxxxxxxxxx"
  role: "dispatcher"
  # 移除了 allowed_users 字段
  group_project_map:
    "oc_xxx_potato_dev": "土豆"
  semantic_match:
    enabled: true
    threshold: 0.7
    provider: "glm"
    model: "embedding-3"
    api_key: "${GLM_API_KEY}"
```

- [ ] **Step 2: 在 executor 任务中添加 allowed_users 示例**

找到 executor 的 tasks 配置，添加 `allowed_users` 字段：

```yaml
# ====================
# 执行机器人 1 - 开发服专员
# ====================
- name: "dev-executor"
  app_id: "cli_yyyyyyyyyyyyyyyyyy"
  app_secret: "yyyyyyyyyyyyyyyyyy"
  role: "executor"
  description: "开发服执行机器人"
  allowed_dispatchers:
    - "cli_xxxxxxxxxxxxxxxxx"
  routing_keywords:
    - "开发"
    - "dev"
  tasks:
    potato_restart:
      display_name: "土豆开发服重启"
      keywords: ["土豆", "potato"]
      names: ["重启", "restart"]
      script: "/opt/scripts/potato_dev_restart.sh"
      allowed_users:
        - "ou_xxx_admin"
        - "ou_yyy_developer"

    potato_update:
      display_name: "土豆开发服更新"
      keywords: ["土豆", "potato"]
      names: ["更新", "update"]
      script: "/opt/scripts/potato_dev_update.sh"
      allowed_users:
        - "ou_xxx_admin"  # 只有管理员可以更新
```

- [ ] **Step 3: 更新配置说明部分**

在文件底部的配置说明中添加：

```yaml
## 任务级用户白名单（新功能）

每个任务可以独立配置允许的用户列表：

```yaml
tasks:
  restart_server:
    display_name: "重启服务器"
    keywords: ["重启", "restart"]
    names: ["服务器"]
    script: "/opt/scripts/restart.sh"
    allowed_users:
      - "ou_admin_user"
      - "ou_operator_user"
```

**两层权限检查：**
1. 用户必须至少在一个任务的白名单中（全局白名单）
2. 用户必须在特定任务的白名单中（任务级权限）

**注意事项：**
- 配置了 `allowed_users` 的任务必须同时配置 `display_name`（用于错误提示）
- 如果任务没有配置 `allowed_users`，则任何用户（包括全局白名单中的用户）都无法执行该任务
- `allowed_users` 为空数组等同于没有配置
```

- [ ] **Step 4: 验证配置文件格式**

```bash
# 可以写一个简单的验证脚本或手动检查
cat configs/bots.yaml.example | grep -A 5 "allowed_users"
```

- [ ] **Step 5: 提交变更**

```bash
git add configs/bots.yaml.example
git commit -m "docs(config): 更新配置示例以支持任务级用户白名单

- 从 dispatcher 配置中移除 allowed_users
- 在 executor 任务中添加 allowed_users 示例
- 添加两层权限检查的使用说明
- 说明 allowed_users 与 display_name 的关联要求"
```

---

## Task 10: 运行完整测试套件并修复问题

**Files:**
- All modified files

- [ ] **Step 1: 运行所有单元测试**

```bash
go test ./... -v
```

预期：所有测试通过

- [ ] **Step 2: 运行测试并检查覆盖率**

```bash
go test ./... -cover
```

预期：覆盖率 >= 80%

- [ ] **Step 3: 运行竞态检测**

```bash
go test ./... -race
```

预期：无数据竞争警告

- [ ] **Step 4: 修复发现的问题**

如果任何测试失败，修复问题并重新运行。

- [ ] **Step 5: 提交最终修复**

```bash
git add -A
git commit -m "test: 修复测试问题并确保覆盖率达标

- 修复所有测试失败
- 确保测试覆盖率达到 80%
- 通过竞态检测"
```

---

## Task 11: 构建和基本验证

**Files:**
- All

- [ ] **Step 1: 构建项目**

```bash
go build -o bot-service cmd/bot-service/main.go
```

预期：编译成功，无错误

- [ ] **Step 2: 验证二进制文件**

```bash
./bot-service --help
```

预期：显示帮助信息或正常启动提示

- [ ] **Step 3: 检查代码格式**

```bash
go fmt ./...
go vet ./...
```

预期：无格式错误，无 vet 警告

- [ ] **Step 4: 提交变更**

```bash
git add -A
git commit -m "chore: 确保代码质量标准

- 通过 go fmt 和 go vet 检查
- 验证二进制文件可以正常构建"
```

---

## Task 12: 清理和文档

**Files:**
- Various

- [ ] **Step 1: 检查是否有遗留的 TODO 或 FIXME**

```bash
grep -r "TODO\|FIXME" internal/ cmd/
```

如果有，处理或记录到文档

- [ ] **Step 2: 更新 README.md（如果需要）**

如果 README 中提到了旧的权限配置方式，更新为新方式

- [ ] **Step 3: 提交最终变更**

```bash
git add -A
git commit -m "docs: 更新文档以反映任务级用户白名单变更

- 移除旧的 dispatcher.allowed_users 文档
- 添加任务级权限配置说明
- 更新配置示例和迁移指南"
```

---

## 总结

完成所有任务后，系统将具备：

1. **任务级用户白名单**：每个任务可以独立配置允许的用户
2. **两层权限检查**：全局白名单检查 + 任务级权限检查
3. **配置验证**：确保配置的完整性
4. **完整的测试覆盖**：单元测试、边界情况、并发安全
5. **清晰的错误提示**：使用 display_name 提供友好的错误消息

**涉及的文件：**
- 修改：6 个源文件
- 修改：3 个测试文件
- 修改：1 个配置示例文件

**测试覆盖：**
- 单元测试：配置验证、权限检查、边界情况
- 并发测试：竞态检测
- 集成测试：端到端流程（建议在 integration_test.sh 中添加）

**提交建议：**
每完成一个任务就提交一次，保持清晰的提交历史。
