# Design: 用户姓名映射表

> 日期：2026-03-29

## 背景

当前 `TaskDetail.AllowedUsers` 字段直接存储飞书 `open_id`（如 `ou_xxx`），配置可读性差，维护人员难以识别谁有权限。

## 目标

新增全局用户映射表（姓名 → open_id），让 `allowed_users` 字段直接配置用户姓名，提升配置可读性。

## 方案：启动时静态解析

在配置加载阶段，将 `allowed_users` 中的姓名通过全局映射表解析为 `open_id`，后续业务逻辑完全不变。

选择静态解析而非运行时动态解析的理由：
- 当前项目没有热更新机制，YAGNI
- 启动时报错比运行时静默失败更安全
- 运行时零开销

## 约束

- `allowed_users` **只支持姓名**，不再支持直接填写 open_id。所有 open_id 必须通过全局 `users` 映射表注册。
- 旧格式任务（`LegacyTasks`、`TaskScripts`）没有 `AllowedUsers` 字段，无需处理。
- `users` 映射表的 value 必须是非空字符串，启动时校验。

## 变更详情

### 1. 配置模型

在 `ServiceConfig` 中新增 `Users` 字段（`internal/config/config.go`）：

```go
type ServiceConfig struct {
    Bots         []BotConfig       `yaml:"bots"`
    RobotGroupID string            `yaml:"robot_group_id"`
    Users        map[string]string `yaml:"users,omitempty"` // 姓名 → open_id 全局映射
}
```

### 2. YAML 配置示例

```yaml
# 全局用户映射表：姓名 → open_id
users:
  张三: ou_abc123
  李四: ou_def456

robot_group_id: "oc_xxx"
bots:
  - name: dev-executor
    role: executor
    tasks:
      - name: "restart"
        display_name: "开发服重启"
        script: /opt/scripts/restart.sh
        allowed_users: [张三, 李四]
```

### 3. 解析函数

新增 `ResolveUsers` 函数（`internal/config/user_resolver.go`）：

```go
func ResolveUsers(cfg *ServiceConfig) error {
    for i := range cfg.Bots {
        for taskName, task := range cfg.Bots[i].Tasks {
            resolved := make([]string, 0, len(task.AllowedUsers))
            for _, name := range task.AllowedUsers {
                openID, ok := cfg.Users[name]
                if !ok {
                    return fmt.Errorf("bot %q task %q: user %q not found in global users mapping",
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

### 4. 调用时序

`ResolveUsers` 必须在 `main.go` 中的以下位置调用：

```
LoadConfig → ValidateConfig → ResolveUsers → 创建 Registry → 创建 Handlers
```

在 `ValidateConfig` 和创建 Handlers 之间调用，因为：
- `ValidateConfig` 校验配置结构完整性（此时 `allowed_users` 仍然是姓名）
- `ResolveUsers` 将姓名解析为 open_id
- `NewDispatcherHandler` 读取 `allowed_users`，此时已经是 open_id

`ValidateConfig` 中现有的校验（如检查 `allowed_users` 配置了但缺少 `display_name`）只检查 `len > 0`，不受姓名→open_id 转换影响，无需调整校验逻辑。

### 5. 不变更的部分

- `dispatcher.go`、`executor.go` 业务逻辑不变
- `DispatcherHandler`、`ExecutorHandler` 结构不变
- 权限校验逻辑不变（比较的仍然是 open_id）

### 6. 测试

- `user_resolver_test.go`：解析成功、姓名不存在报错、空映射表、空 allowed_users、users value 为空时报错
- 更新 `configs/bots.test.yaml`：添加 `users` 映射，将 `test_user_001`、`test_user_002`、`admin_user` 改为姓名引用
- 确保现有 dispatcher 测试通过

### 7. 配置示例更新

更新 `configs/bots.yaml.example`，添加 `users` 映射表示例和注释。

## 测试配置迁移示例

迁移前（`bots.test.yaml`）：

```yaml
bots:
  - name: "test-dev-executor"
    tasks:
      - name: "mist-dev-restart"
        allowed_users:
          - "test_user_001"
```

迁移后：

```yaml
users:
  测试用户1: test_user_001
  测试用户2: test_user_002
  管理员: admin_user

bots:
  - name: "test-dev-executor"
    tasks:
      - name: "mist-dev-restart"
        allowed_users:
          - "测试用户1"
```

## 影响范围

| 文件 | 变更类型 |
|------|---------|
| `internal/config/config.go` | 新增 `Users` 字段 |
| `internal/config/user_resolver.go` | 新增文件，解析函数 |
| `internal/config/user_resolver_test.go` | 新增文件，测试 |
| `configs/bots.yaml.example` | 添加 users 示例 |
| `configs/bots.test.yaml` | 添加 users 映射，迁移 allowed_users |
| `cmd/bot-service/main.go` | ValidateConfig 后调用 ResolveUsers |
