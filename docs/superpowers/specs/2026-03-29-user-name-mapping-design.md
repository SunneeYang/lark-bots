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

## 变更详情

### 1. 配置模型

在 `ServiceConfig` 中新增 `Users` 字段（`internal/config/config.go`）：

```go
type ServiceConfig struct {
    RobotGroupID string              `yaml:"robot_group_id"`
    Users        map[string]string   `yaml:"users,omitempty"` // 姓名 → open_id 全局映射
    Bots         []BotConfig         `yaml:"bots"`
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
      restart:
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

调用时机：在 `ValidateConfig` 之后立即调用，确保业务逻辑拿到的都是 `open_id`。

### 4. 不变更的部分

- `dispatcher.go`、`executor.go` 业务逻辑不变
- `DispatcherHandler`、`ExecutorHandler` 结构不变
- 权限校验逻辑不变（比较的仍然是 open_id）

### 5. 测试

- `user_resolver_test.go`：解析成功、姓名不存在报错、空映射表、空 allowed_users
- 更新 `configs/bots.test.yaml` 加入 `users` 映射
- 确保现有 dispatcher 测试通过

### 6. 配置示例更新

更新 `configs/bots.yaml.example`，添加 `users` 映射表示例和注释。

## 影响范围

| 文件 | 变更类型 |
|------|---------|
| `internal/config/config.go` | 新增 `Users` 字段 |
| `internal/config/user_resolver.go` | 新增文件，解析函数 |
| `internal/config/user_resolver_test.go` | 新增文件，测试 |
| `internal/config/validator.go` | 可能调整校验顺序 |
| `configs/bots.yaml.example` | 添加 users 示例 |
| `configs/bots.test.yaml` | 添加 users 映射 |
| `cmd/bot-service/main.go` | 调用 ResolveUsers |
