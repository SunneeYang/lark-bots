# 任务级用户白名单设计文档

**日期：** 2026-03-27
**作者：** Claude Code
**状态：** 设计阶段

## 1. 背景

当前系统中，Dispatcher 配置有全局的用户白名单（`allowed_users`），所有在白名单中的用户都可以触发任意任务。这种设计无法实现细粒度的权限控制。

**需求：** 将用户白名单从 Dispatcher 全局配置移到每个 Executor 的任务配置中，实现任务级别的用户权限控制。

## 2. 设计目标

- **细粒度权限控制：** 每个任务可以有独立的用户白名单
- **两层权限验证：** 先检查用户是否在系统中任何任务的白名单中，再检查是否在特定任务的白名单中
- **配置简化：** Dispatcher 不再维护独立的用户白名单
- **强制迁移：** 不兼容旧的配置方式，直接采用新设计

## 3. 架构设计

### 3.1 配置结构变更

#### TaskDetail 新增字段

```go
type TaskDetail struct {
    DisplayName  string   `yaml:"display_name,omitempty"`
    Keywords     []string `yaml:"keywords,omitempty"`
    Names        []string `yaml:"names,omitempty"`
    Script       string   `yaml:"script"`
    Params       []string `yaml:"params,omitempty"`
    AllowedUsers []string `yaml:"allowed_users,omitempty"`  // 新增：允许的用户列表
}
```

#### BotConfig 移除字段

```go
type BotConfig struct {
    Name      string `yaml:"name"`
    AppID     string `yaml:"app_id"`
    AppSecret string `yaml:"app_secret"`
    Role      string `yaml:"role"`

    // 移除：AllowedUsers []string `yaml:"allowed_users,omitempty"`

    GroupProjectMap map[string]string `yaml:"group_project_map,omitempty"`

    AllowedDispatchers []string          `yaml:"allowed_dispatchers,omitempty"`
    RoutingKeywords    []string          `yaml:"routing_keywords,omitempty"`
    Description        string            `yaml:"description,omitempty"`
    LegacyTasks        []ExecutorTask    `yaml:"legacy_tasks,omitempty"`
    Tasks              map[string]TaskDetail `yaml:"tasks,omitempty"`
    TaskScripts        map[string]string `yaml:"task_scripts,omitempty"`
    PollInterval       string            `yaml:"poll_interval,omitempty"`
    MaxTasks           int               `yaml:"max_tasks,omitempty"`
    SemanticMatch      *SemanticMatchConfig `yaml:"semantic_match,omitempty"`
}
```

### 3.2 配置文件示例

```yaml
# Dispatcher 配置（无 allowed_users）
- name: "task-dispatcher"
  role: "dispatcher"
  group_project_map:
    "oc_xxx_potato_dev": "土豆"

# Executor 配置（每个任务有独立的 allowed_users）
- name: "dev-executor"
  role: "executor"
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

### 3.3 Dispatcher 数据结构变更

```go
type DispatcherHandler struct {
    *BaseHandler

    userWhiteList        map[string]bool           // 所有任务用户白名单的并集（第一层检查）
    layeredMatcher       *matcher.LayeredMatcher
    groupProjectMap      map[string]string
    userInfoCache        map[string]string
    userInfoCacheMu      sync.RWMutex

    taskUserPermissions  map[string][]string       // 任务名 → 允许的用户列表（第二层检查）
}
```

**职责说明：**
- `userWhiteList` (map[string]bool): 第一层检查，用户是否在系统的任何任务白名单中
- `taskUserPermissions` (map[string][]string): 第二层检查，用户是否在特定任务的白名单中

### 3.4 初始化流程

#### NewDispatcherHandler 内部构建权限数据

```go
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
        layeredMatcher:      nil, // 稍后通过 SetLayeredMatcher 设置
        userInfoCache:       make(map[string]string),
    }
}
```

#### main.go 调用方式

```go
func main() {
    // 加载配置
    cfg, err := config.LoadConfig(configPath)
    if err != nil {
        log.Fatal(err)
    }

    // 创建 dispatcher（内部自动构建权限数据）
    dispatcherHandler := handler.NewDispatcherHandler(cfg, semanticCfg)

    // 启动机器人...
}

### 3.5 权限检查流程

#### 两层权限检查

```go
func (h *DispatcherHandler) HandleMessage(event *eventreceiver.V2Event) error {
    // 解析发送者
    senderID, err := h.extractSenderID(event)
    if err != nil {
        return fmt.Errorf("解析发送者失败: %w", err)
    }

    // 第一层：检查用户是否在任何任务的白名单中
    if !h.userWhiteList[senderID] {
        return fmt.Errorf("❌ 你没有执行任务的权限")
    }

    // 继续任务匹配
    userInput := extractUserInput(event)
    matchedTask, err := h.layeredMatcher.Match(userInput)
    if err != nil {
        return err
    }

    // 第二层：检查用户是否在匹配到的任务白名单中
    allowedUsers, exists := h.taskUserPermissions[matchedTask.Name]
    if !exists || !contains(allowedUsers, senderID) {
        return fmt.Errorf("❌ 你没有权限执行任务：%s", matchedTask.DisplayName)
    }

    // 发送到机器人群...
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

#### 边界情况处理

1. **任务没有配置 `allowed_users`**：拒绝所有用户（包括全局白名单中的用户）
2. **`allowed_users` 为空数组**：该任务不会向全局白名单贡献任何用户
3. **多个任务有相同用户**：全局白名单自动去重

### 3.6 配置验证

#### validator.go 新增验证

```go
func ValidateConfig(cfg *ServiceConfig) error {
    // ... 现有验证逻辑 ...

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

    return nil
}
```

## 4. 测试策略

### 4.1 单元测试

#### dispatcher_test.go

```go
func TestDispatcher_TwoLayerPermissionCheck(t *testing.T) {
    tests := []struct {
        name        string
        senderID    string
        userInput   string
        globalUsers map[string]bool
        taskPerms   map[string][]string
        expectedErr string
    }{
        {
            name:     "用户不在全局白名单中",
            senderID: "ou_999",
            userInput: "土豆重启",
            globalUsers: map[string]bool{
                "ou_111": true,
                "ou_222": true,
            },
            taskPerms: map[string][]string{
                "potato_restart": {"ou_111"},
            },
            expectedErr: "你没有执行任务的权限",
        },
        {
            name:     "用户在全局白名单但不在任务白名单",
            senderID: "ou_222",
            userInput: "土豆重启",
            globalUsers: map[string]bool{
                "ou_111": true,
                "ou_222": true,
            },
            taskPerms: map[string][]string{
                "potato_restart": {"ou_111"},
            },
            expectedErr: "你没有权限执行任务：土豆开发服重启",
        },
        {
            name:     "用户有权限",
            senderID: "ou_111",
            userInput: "土豆重启",
            globalUsers: map[string]bool{
                "ou_111": true,
            },
            taskPerms: map[string][]string{
                "potato_restart": {"ou_111"},
            },
            expectedErr: "",
        },
    }
    // ... 测试实现 ...
}

func TestDispatcher_TaskPermissionEdgeCases(t *testing.T) {
    tests := []struct {
        name        string
        senderID    string
        userInput   string
        globalUsers map[string]bool
        taskPerms   map[string][]string
        expectedErr string
    }{
        {
            name:     "任务没有配置 allowed_users，拒绝所有用户",
            senderID: "ou_111",
            userInput: "某个任务",
            globalUsers: map[string]bool{
                "ou_111": true,
            },
            taskPerms: map[string][]string{
                // 该任务没有权限配置
            },
            expectedErr: "你没有权限执行任务：某个任务",
        },
        {
            name:     "allowed_users 为空数组",
            senderID: "ou_111",
            userInput: "某个任务",
            globalUsers: map[string]bool{}, // 空数组不会加入全局白名单
            taskPerms: map[string][]string{
                "某个任务": {},
            },
            expectedErr: "你没有执行任务的权限",
        },
        {
            name:     "多个 executor 有相同用户，全局白名单去重",
            senderID: "ou_111",
            userInput: "任务1",
            globalUsers: map[string]bool{
                "ou_111": true,
            },
            taskPerms: map[string][]string{
                "任务1": {"ou_111", "ou_222"},
                "任务2": {"ou_111", "ou_333"},
            },
            expectedErr: "",
        },
    }
    // ... 测试实现 ...
}

func TestDispatcher_PermissionCheckConcurrency(t *testing.T) {
    // 模拟多个 goroutine 同时读取权限数据
    // 验证不会发生数据竞争
}
```

#### validator_test.go

```go
func TestValidateConfig_TaskAllowedUsers(t *testing.T) {
    tests := []struct {
        name      string
        config    string
        wantErr   bool
        errContains string
    }{
        {
            name: "任务有 allowed_users 但没有 display_name",
            config: `
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
`,
            wantErr:     true,
            errContains: "缺少 display_name",
        },
        {
            name: "任务有 allowed_users 且有 display_name",
            config: `
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
`,
            wantErr: false,
        },
    }
    // ... 测试实现 ...
}
```

### 4.2 集成测试

扩展 `integration_test.sh`，添加两层权限检查的端到端测试场景。

## 5. 实现步骤

### 涉及的文件

1. **internal/config/config.go**
   - `TaskDetail` 新增 `AllowedUsers []string`
   - `BotConfig` 移除 `AllowedUsers`

2. **internal/config/validator.go**
   - 新增验证：有 `allowed_users` 的任务必须有 `display_name`

3. **internal/handler/dispatcher.go**
   - `DispatcherHandler` 新增 `taskUserPermissions map[string][]string`
   - 移除 `SetAllowedUsers()` 方法
   - 移除 `SetAllowedTasks()` 方法
   - 修改 `HandleMessage()` 添加第二层权限检查

4. **cmd/bot-service/main.go**
   - 启动时收集所有任务的 `allowed_users`
   - 修改 `NewDispatcherHandler()` 调用

5. **configs/bots.yaml.example**
   - 移除 dispatcher 的 `allowed_users`
   - 在每个 executor 任务中添加 `allowed_users` 示例

6. **测试文件**
   - `internal/handler/dispatcher_test.go`
   - `internal/config/validator_test.go`
   - `internal/config/config_test.go`

### 实现顺序

1. 修改配置结构（config.go）
2. 修改配置验证（validator.go）
3. 修改 Dispatcher 数据结构（dispatcher.go）
4. 修改启动逻辑（main.go）
5. 更新配置示例（bots.yaml.example）
6. 更新测试

## 6. 风险和注意事项

### 风险点

1. **配置错误风险**
   - 如果所有任务都没有配置 `allowed_users`，全局白名单为空，所有用户都无法使用
   - **缓解措施**：启动时检查，如果全局白名单为空则警告

2. **误操作风险**
   - 管理员可能在配置时遗漏某个用户的权限
   - **缓解措施**：提供辅助脚本，检查用户是否在至少一个任务的白名单中

3. **性能影响**
   - 权限检查增加了一次 map 查询（性能影响可忽略）
   - 启动时构建权限数据（一次性开销）

### 注意事项

1. **并发安全**：`taskUserPermissions` 只在启动时写入，运行时只读，无需锁保护

2. **错误消息**：使用 `display_name` 而不是任务内部名称，提供更好的用户体验

3. **配置验证**：确保配置了 `allowed_users` 的任务必须有 `display_name`

4. **测试覆盖**：重点测试边界情况（空数组、未配置、并发读取）

## 7. 向后兼容性

**不提供向后兼容**：直接移除旧的 `dispatcher.allowed_users` 配置方式，强制使用新设计。

配置文件需要手动迁移：
1. 从 dispatcher 配置中移除 `allowed_users`
2. 将用户列表分配到各个 executor 任务的 `allowed_users` 中

## 8. 示例配置迁移

### 迁移前

```yaml
- name: "task-dispatcher"
  role: "dispatcher"
  allowed_users:              # 旧配置
    - "ou_admin"
    - "ou_dev1"
    - "ou_ops1"

- name: "dev-executor"
  role: "executor"
  tasks:
    restart:
      script: "/restart.sh"
    update:
      script: "/update.sh"
```

### 迁移后

```yaml
- name: "task-dispatcher"
  role: "dispatcher"
  # 移除 allowed_users

- name: "dev-executor"
  role: "executor"
  tasks:
    restart:
      display_name: "重启服务"
      script: "/restart.sh"
      allowed_users:          # 新配置
        - "ou_admin"
        - "ou_dev1"
        - "ou_ops1"
    update:
      display_name: "更新服务"
      script: "/update.sh"
      allowed_users:          # 只有管理员可以更新
        - "ou_admin"
```
