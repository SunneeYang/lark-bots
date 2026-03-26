# 任务参数化和发布者信息传递设计文档

**日期**: 2026-03-26
**作者**: Claude
**状态**: 设计阶段

## 1. 需求概述

### 1.1 当前问题

当前 Dispatcher 和 Executor 之间的消息机制存在以下限制：

1. **缺少发布者信息**：Executor 执行任务时无法知道是谁发起的任务
2. **参数硬编码**：脚本参数必须硬编码在脚本中，无法灵活配置
3. **执行日志不清晰**：无法在日志中区分不同用户发起的同名任务

### 1.2 目标

1. **参数配置化**：在配置文件中为每个任务预定义参数列表
2. **发布者信息传递**：Dispatcher 获取并传递用户真实姓名给 Executor
3. **用户信息缓存**：在 Dispatcher 中缓存 OpenID → 真实姓名映射，减少 API 调用
4. **统一 JSON 格式**：使用 JSON 格式在 Dispatcher 和 Executor 之间传递任务信息
5. **安全性优先**：先检查用户白名单，再调用 API 获取用户信息

### 1.3 使用场景示例

**用户输入**：
```
土豆开发服重启
```

**Dispatcher 处理**：
1. 检查用户白名单
2. 匹配任务：`miwu-dev-restart`
3. 获取用户真实姓名：`张三`
4. 从配置读取参数：`["dev", "zh"]`
5. 构建 JSON：`{"task": "miwu-dev-restart", "requester": "张三"}`
6. 发送到机器人群

**Executor 执行**：
1. 解析 JSON 获取任务名和发布者
2. 从配置读取参数：`["dev", "zh"]`
3. 执行脚本：`./build.sh dev zh`
4. 汇报结果：`[张三] 任务 [miwu-dev-restart] 执行成功`

## 2. 架构设计

### 2.1 消息流转

```
用户输入 "土豆开发服重启"
    ↓
Dispatcher 匹配任务
    ↓
检查用户白名单（必须首先执行）
    ↓ 通过
获取发布者信息（缓存或 API）
    ↓
构建精简 JSON: {"task":"miwu-dev-restart","requester":"张三"}
    ↓
发送到机器人群
    ↓
Executor 轮询获取消息
    ↓
解析 JSON，从配置中读取参数列表 ["dev", "zh"]
    ↓
执行脚本: ./build.sh dev zh
    ↓
汇报结果: "[张三] 任务 [miwu-dev-restart] 执行成功"
```

### 2.2 JSON 消息格式

**Dispatcher → Executor** 消息格式：

```json
{
  "task": "miwu-dev-restart",
  "requester": "张三"
}
```

**字段说明**：
- `task`: 任务名称（字符串，必填）
- `requester`: 发布者真实姓名（字符串，必填）

**设计决策**：
- 只传递任务名和发布者，不传递参数列表（参数在 Executor 配置中维护）
- 保持消息精简，便于解析和调试
- 发布者信息用于日志追踪和权限审计

## 3. 数据结构设计

### 3.1 TaskCommand 结构

```go
// TaskCommand 任务命令结构（Dispatcher → Executor 通信格式）
type TaskCommand struct {
    TaskName  string // 任务名称
    Requester string // 发布者姓名
}

// JSON 序列化为 JSON 格式
func (c *TaskCommand) JSON() string {
    data := map[string]interface{}{
        "task":      c.TaskName,
        "requester": c.Requester,
    }
    bytes, _ := json.Marshal(data)
    return string(bytes)
}

// ParseTaskCommandJSON 从 JSON 字符串解析任务命令
func ParseTaskCommandJSON(input string) (*TaskCommand, error) {
    var data map[string]interface{}
    if err := json.Unmarshal([]byte(input), &data); err != nil {
        return nil, fmt.Errorf("JSON 解析失败: %w", err)
    }

    taskName, ok := data["task"].(string)
    if !ok || taskName == "" {
        return nil, fmt.Errorf("缺少或无效的 task 字段")
    }

    requester, ok := data["requester"].(string)
    if !ok {
        return nil, fmt.Errorf("缺少或无效的 requester 字段")
    }

    return &TaskCommand{
        TaskName:  taskName,
        Requester: requester,
    }, nil
}
```

### 3.2 配置结构

```go
// TaskDetail 任务详细配置
type TaskDetail struct {
    Script string   // 脚本路径
    Params []string // 参数列表
}

// ExecutorConfig executor 配置
type ExecutorConfig struct {
    Name               string                 // 机器人名字
    AppID              string                 // 飞书应用 ID
    AppSecret          string                 // 飞书应用密钥
    Role               string                 // 角色
    AllowedDispatchers []string               // 允许的 dispatcher app_id 列表
    Tasks              map[string]TaskDetail  // 任务名 → 任务详细配置
}
```

### 3.3 Dispatcher 内部结构

```go
// DispatcherHandler 分发机器人处理器
type DispatcherHandler struct {
    *BaseHandler

    userWhiteList   map[string]bool
    layeredMatcher  *matcher.LayeredMatcher
    groupProjectMap map[string]string
    userInfoCache   map[string]string  // OpenID → 真实姓名（新增）
}
// larkClient 使用 botClient.LarkClient，不需要新增字段
```

## 4. Dispatcher 实现

### 4.1 处理流程

```
收到用户消息
    ↓
1. 检查用户白名单（必须首先执行）
    ↓ 通过
2. 解析消息内容
    ↓
3. 匹配任务
    ↓ 匹配成功
4. 获取用户真实姓名（调用 API 或缓存）
    ↓
5. 构建 JSON 消息
    ↓
6. 发送到机器人群
    ↓
7. 回复用户
```

### 4.2 用户信息获取

```go
// getUserInfo 获取用户真实姓名（带缓存）
func (h *DispatcherHandler) getUserInfo(ctx context.Context, botClient *bot.BotClient, openID string) (string, error) {
    // 检查缓存
    if name, exists := h.userInfoCache[openID]; exists {
        return name, nil
    }

    // 调用飞书 API
    userInfo, err := botClient.LarkClient.Authen.UserInfo.Get(ctx, larkauthen.NewGetUserInfoReqBuilder().
        OpenID(openID).
        Build())
    if err != nil {
        return "", fmt.Errorf("获取用户信息失败: %w", err)
    }

    realName := userInfo.Data.Name
    // 存入缓存
    h.userInfoCache[openID] = realName

    return realName, nil
}
```

### 4.3 任务分发逻辑

```go
// handleLayeredMode 使用分层匹配器处理消息
func (h *DispatcherHandler) handleLayeredMode(ctx context.Context, event interface{}, message, _ string, botClient *bot.BotClient, senderID string) error {
    // 自动补充群组的项目关键词
    enhancedMessage := h.enhanceMessageWithGroupProject(ctx, event, message)

    result := h.layeredMatcher.Match(ctx, matcher.LayeredMatchInput{UserInput: enhancedMessage})

    // 否定检测
    if result.HasNegation {
        replyMsg := "检测到否定意图（如「不要」「别」），请重新描述你要执行的操作"
        if err := h.replyToUser(event, replyMsg, botClient); err != nil {
            return fmt.Errorf("回复用户失败: %w", err)
        }
        return fmt.Errorf("检测到否定意图")
    }

    // 多意图检测
    if len(result.Matches) > 1 {
        replyMsg := "检测到多个匹配的任务，请一次只说一个服务器和一个操作"
        if err := h.replyToUser(event, replyMsg, botClient); err != nil {
            return fmt.Errorf("回复用户失败: %w", err)
        }
        return fmt.Errorf("多意图请求: %d 个匹配", len(result.Matches))
    }

    // 无匹配
    if len(result.Matches) == 0 {
        replyMsg := "未找到匹配的任务，请检查输入是否包含服务器名和操作（如「土豆开发服重启」）"
        if err := h.replyToUser(event, replyMsg, botClient); err != nil {
            return fmt.Errorf("回复用户失败: %w", err)
        }
        return fmt.Errorf("未找到匹配任务")
    }

    // 单个匹配，执行
    match := result.Matches[0]

    // 获取发布者信息（此时用户已在白名单中，可以安全调用 API）
    requester, err := h.getUserInfo(ctx, botClient, senderID)
    if err != nil {
        replyMsg := "❌ 获取用户信息失败，请稍后重试"
        h.replyToUser(event, replyMsg, botClient)
        return fmt.Errorf("获取发布者信息失败: %w", err)
    }

    // 构建精简的 TaskCommand
    cmd := &TaskCommand{
        TaskName:  match.TaskName,
        Requester: requester,
    }
    msgToSend := cmd.JSON()

    fmt.Printf("📤 [%s] 分发任务: %s\n", botClient.Name, msgToSend)

    if err := h.SendToGroup(msgToSend, botClient); err != nil {
        return fmt.Errorf("分发任务失败: %w", err)
    }

    // 回复用户
    taskDisplay := match.DisplayName
    if taskDisplay == "" {
        taskDisplay = match.TaskName
    }
    replyMsg := fmt.Sprintf("✅ 任务 [%s] 开始执行", taskDisplay)
    if err := h.replyToUser(event, replyMsg, botClient); err != nil {
        return fmt.Errorf("回复用户失败: %w", err)
    }

    return nil
}
```

## 5. Executor 实现

### 5.1 Handle 方法

```go
// Handle 处理消息（统一 JSON 格式）
func (h *ExecutorHandler) Handle(_ context.Context, event interface{}, botClient *bot.BotClient) error {
    // executor 使用轮询机制获取任务，不处理 WebSocket 事件
    // 过滤：只处理来自机器人的消息，忽略用户消息
    if !common.IsFromApp(event) {
        return nil // 静默忽略用户消息
    }

    // 解析消息内容
    rawContent, _ := common.ExtractMessageContent(event)
    message, _ := common.ParseMessageContent(rawContent)

    // 通过 app_id 校验是否来自允许的 dispatcher
    senderAppID, err := common.ExtractSenderAppID(event)
    if err != nil {
        return fmt.Errorf("解析 sender app_id 失败: %w", err)
    }

    // 校验是否来自允许的 dispatcher（按 app_id）
    if !h.allowedDispatchers[senderAppID] {
        return fmt.Errorf("未授权的 dispatcher: %s", senderAppID)
    }

    // 解析 JSON 格式的任务命令
    taskCmd, err := ParseTaskCommandJSON(message)
    if err != nil {
        return fmt.Errorf("解析任务命令失败: %w", err)
    }

    // 执行任务
    return h.executeTask(taskCmd, botClient)
}
```

### 5.2 executeTask 方法

```go
// executeTask 执行任务
func (h *ExecutorHandler) executeTask(taskCmd *TaskCommand, botClient *bot.BotClient) error {
    taskName := taskCmd.TaskName
    requester := taskCmd.Requester

    // 从配置中查找任务详细配置
    taskDetail, ok := h.tasks[taskName]
    if !ok {
        return nil // 任务不在自己的任务列表中，静默忽略
    }

    // 执行脚本（传递参数）
    output, err := h.executeScript(taskDetail.Script, taskDetail.Params)

    // 向机器人群汇报结果（包含发布者信息）
    resultMsg := fmt.Sprintf("[%s] [%s] 任务 [%s] 执行%s",
        requester, botClient.Name, taskName,
        map[bool]string{true: "成功", false: "失败"}[err == nil])
    if err != nil {
        resultMsg += fmt.Sprintf(": %v", err)
    } else {
        // 截断输出
        outputLines := strings.Split(strings.TrimSpace(output), "\n")
        if len(outputLines) > 5 {
            output = strings.Join(outputLines[:5], "\n") + "\n...(输出已截断)"
        }
        resultMsg += fmt.Sprintf("\n输出:\n%s", strings.TrimSpace(output))
    }

    fmt.Printf("📤 [%s] 汇报结果: %s\n", botClient.Name, taskName)
    if err := h.SendToGroup(resultMsg, botClient); err != nil {
        fmt.Printf("❌ [%s] 汇报结果失败: %v\n", botClient.Name, err)
    }

    return nil
}
```

### 5.3 executeScript 方法

```go
// executeScript 执行脚本（位置参数传递）
func (h *ExecutorHandler) executeScript(scriptPath string, args []string) (string, error) {
    cmd := exec.Command(scriptPath, args...)
    output, err := cmd.CombinedOutput()
    if err != nil {
        return string(output), fmt.Errorf("脚本执行失败: %w", err)
    }
    return string(output), nil
}
```

## 6. 配置文件格式

### 6.1 完整配置示例

```yaml
# 机器人群 ID
robot_group_id: "oc_xxxxxxxxxxxxxxxxx"

# 机器人配置
bots:
  # 分发机器人
  - name: "dispatcher"
    app_id: "cli_xxxxxxxxxxxxxxxxx"
    app_secret: "xxxxxxxxxxxxxxxxxxxx"
    role: "dispatcher"
    allowed_users:
      - "ou_xxxxxxxxxxxxxxxxx"  # 允许的用户列表

  # 执行机器人
  - name: "dev-executor"
    app_id: "cli_yyyyyyyyyyyyyyyyyy"
    app_secret: "yyyyyyyyyyyyyyyyyy"
    role: "executor"
    allowed_dispatchers:
      - "cli_xxxxxxxxxxxxxxxxx"  # dispatcher 的 app_id
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
```

### 6.2 配置字段说明

**Executor 配置字段**：
- `tasks`: 任务映射表（对象）
  - `<task_name>`: 任务名称（键）
    - `script`: 脚本路径（字符串，必填）
    - `params`: 参数列表（数组，必填，可以为空）

## 7. 错误处理

### 7.1 Dispatcher 错误处理

**用户不在白名单**：
```go
if !h.userWhiteList[senderID] {
    return fmt.Errorf("用户不在白名单中: %s", senderID)
}
```

**获取用户信息失败**：
```go
if err != nil {
    replyMsg := "❌ 获取用户信息失败，请稍后重试"
    h.replyToUser(event, replyMsg, botClient)
    return fmt.Errorf("获取发布者信息失败: %w", err)
}
```

**任务匹配失败**：
- 否定意图：提示用户重新描述
- 多意图：提示用户一次只说一个任务
- 无匹配：提示用户检查输入格式

### 7.2 Executor 错误处理

**JSON 解析失败**：
```go
taskCmd, err := ParseTaskCommandJSON(message)
if err != nil {
    return fmt.Errorf("解析任务命令失败: %w", err)
}
```

**任务不在配置中**：
```go
if !ok {
    return nil // 静默忽略（可能是发给其他 executor 的）
}
```

**脚本执行失败**：
```go
if err != nil {
    resultMsg += fmt.Sprintf(": %v", err)
}
// 仍然汇报到群组，包含错误信息
```

## 8. 测试策略

### 8.1 单元测试

**TaskCommand 序列化/反序列化测试**：
```go
func TestTaskCommand_JSONSerialization(t *testing.T) {
    cmd := &TaskCommand{
        TaskName:  "miwu-dev-restart",
        Requester: "张三",
    }
    json := cmd.JSON()

    parsed, err := ParseTaskCommandJSON(json)
    assert.NoError(t, err)
    assert.Equal(t, cmd.TaskName, parsed.TaskName)
    assert.Equal(t, cmd.Requester, parsed.Requester)
}
```

**用户信息缓存测试**：
```go
func TestDispatcherHandler_GetUserInfo(t *testing.T) {
    handler := NewDispatcherHandler(nil)

    // Mock 飞书 API
    // 第一次调用：验证 API 被调用
    name1, err := handler.getUserInfo(ctx, botClient, "ou_test1")
    assert.NoError(t, err)
    assert.Equal(t, "张三", name1)

    // 第二次调用：验证使用缓存，API 未被调用
    name2, err := handler.getUserInfo(ctx, botClient, "ou_test1")
    assert.NoError(t, err)
    assert.Equal(t, "张三", name2)
}
```

**Executor 执行测试**：
```go
func TestExecutorHandler_ExecuteTask(t *testing.T) {
    handler := NewExecutorHandler()
    handler.tasks = map[string]TaskDetail{
        "test-task": {
            Script: "/tmp/test.sh",
            Params: []string{"arg1", "arg2"},
        },
    }

    taskCmd := &TaskCommand{
        TaskName:  "test-task",
        Requester: "张三",
    }

    // Mock 脚本执行
    // 验证参数传递正确
}
```

### 8.2 集成测试

**完整消息流测试**：
```bash
# 1. 启动 dispatcher 和 executor
./start.sh dispatcher,dev-executor

# 2. 在测试群发送消息
# 用户：土豆开发服重启

# 3. 验证：
# - Dispatcher 日志显示获取用户信息
# - 机器人群收到 JSON 消息
# - Executor 日志显示执行脚本及参数
# - 执行结果包含发布者信息
```

## 9. 实施步骤

### 阶段 1：数据结构扩展
1. 扩展 `TaskCommand` 结构，添加 `Requester` 字段
2. 实现 `JSON()` 方法序列化为 JSON 格式
3. 实现 `ParseTaskCommandJSON()` 方法解析 JSON
4. 添加 `TaskDetail` 配置结构
5. 更新 `ExecutorConfig` 添加 `Tasks` 字段

### 阶段 2：Dispatcher 实现
1. 添加 `userInfoCache` 字段到 `DispatcherHandler`
2. 实现 `getUserInfo()` 方法（支持缓存和 API 调用）
3. 修改 `Handle()` 方法传递 `senderID` 参数
4. 修改 `handleLayeredMode()` 方法调用 `getUserInfo()` 并构建 JSON
5. 修改 `handleLegacyMode()` 方法（如果需要支持旧格式）
6. 更新单元测试

### 阶段 3：Executor 实现
1. 移除旧格式支持（`taskScripts` 和 `taskNameToScript`）
2. 添加 `tasks` 字段（`map[string]TaskDetail`）
3. 实现 `SetTasks()` 方法
4. 实现 `executeTask()` 方法
5. 修改 `Handle()` 方法使用 JSON 解析
6. 更新 `executeScript()` 方法（保持不变）
7. 更新单元测试

### 阶段 4：配置和验证
1. 更新 `config.go` 中的配置结构
2. 更新 `validator.go` 添加 `tasks` 字段验证
3. 更新配置加载逻辑（`main.go` 中的配置解析）
4. 更新 `configs/bots.yaml.example` 配置模板

### 阶段 5：测试和文档
1. 运行所有单元测试
2. 运行集成测试
3. 更新 README.md 文档
4. 更新配置示例和说明

## 10. 风险和注意事项

### 10.1 性能考虑

**用户信息缓存**：
- 缓存存储在内存中，服务重启后清空
- 预期缓存命中率：90%+（大部分用户会重复发起任务）
- API 调用频率：仅在首次遇到新用户时调用

**内存占用**：
- 每个用户约 100 bytes（OpenID + 真实姓名）
- 1000 个用户约 100 KB，可忽略不计

### 10.2 安全性考虑

**白名单优先**：
- 必须先检查用户白名单，再调用 API
- 避免为未授权用户调用 API 资源

**API 限流**：
- 飞书 API 有 QPS 限制，当前设计下影响很小
- 仅在新用户首次发起任务时调用

### 10.3 可维护性

**配置可读性**：
- JSON 格式消息易于调试
- 配置文件结构清晰，参数列表明确

**向后兼容性**：
- 不支持旧格式，需要一次性迁移所有配置
- 建议提供迁移指南

## 11. 后续优化方向

1. **持久化缓存**：将用户信息缓存存储到 Redis 或数据库
2. **缓存过期**：为用户信息缓存添加 TTL（如 24 小时）
3. **批量获取**：支持批量获取用户信息，减少 API 调用
4. **参数模板**：支持参数占位符，如 `{"env": "{{env}}"}` 从用户输入中提取
5. **权限细化**：不同用户对同一任务可以传递不同参数
