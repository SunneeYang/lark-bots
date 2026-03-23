# 飞书机器人服务设计文档

**日期：** 2026-03-23
**状态：** 设计阶段
**作者：** Claude & User

---

## 1. 概述

### 1.1 目标

构建一个基于飞书 Go SDK 的多机器人协作服务，支持：
- 配置文件管理多个机器人
- 启动时指定要运行的机器人列表
- 机器人通过群聊消息进行协作
- 完整的任务日志记录

### 1.2 使用场景

企业内部工具，多机器人协作完成自动化任务：
- 用户在群内 @ 分发机器人发送指令
- 分发机器人在机器人群 @ 执行机器人
- 执行机器人完成任务后汇报结果
- 分发机器人将结果返回给用户

---

## 2. 系统架构

### 2.1 整体架构

```
┌─────────────────────────────────────┐
│         BotService (服务层)          │
│  - 启动/关闭管理                      │
│  - 健康检查                          │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│      MessageRouter (路由层)          │
│  - 识别消息来源                      │
│  - 分发到对应的 Handler              │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│       Handlers (业务逻辑层)          │
│  - DispatcherHandler                │
│  - ExecutorHandler                  │
└─────────────────────────────────────┘
              ↓
┌─────────────────────────────────────┐
│         Common (通用层)              │
│  - 消息发送、表情回复                │
│  - 任务日志记录                      │
└─────────────────────────────────────┘
```

### 2.2 启动流程

```
1. 加载配置文件 → 验证配置完整性
2. 根据启动参数筛选要启用的机器人
3. 为每个机器人创建独立的飞书 SDK Client
4. 注册 Handler 到消息路由器
5. 启动事件监听（webhook 或长轮询）
6. 各机器人开始接收消息
```

---

## 3. 核心组件

### 3.1 配置管理

```go
type BotConfig struct {
    Name              string
    AppID             string
    AppSecret         string
    Role              string  // "dispatcher" | "executor"

    // Executor 特有配置
    AllowedDispatchers []string  // 只接受这些 dispatcher 的命令
    AllowedScripts     []string  // 可执行的脚本白名单
}

type ServiceConfig struct {
    Bots          []BotConfig
    RobotGroupID  string      // 机器人群 ID
    TaskWhiteList []string    // Dispatcher 可分发的任务
    UserWhiteList []string    // Dispatcher 允许的用户
}
```

### 3.2 机器人注册表 (BotRegistry)

```go
type BotClient struct {
    Config    BotConfig
    SDKClient *lark.Client
    Handler   MessageHandler
}

type BotRegistry struct {
    mu     sync.RWMutex
    bots   map[string]*BotClient          // key: bot_name
    byRole map[string][]*BotClient        // key: role
}
```

### 3.3 消息路由器 (MessageRouter)

```go
type MessageRouter struct {
    registry *BotRegistry
    logger   *TaskLogger
}

func (r *MessageRouter) Route(ctx context.Context, event *lark.Event) error {
    receiverBot := r.registry.GetByAppID(event.Receiver.BotID)
    if receiverBot == nil {
        return ErrUnknownBot
    }

    switch receiverBot.Config.Role {
    case "dispatcher":
        return r.dispatcherHandler.Handle(ctx, event, receiverBot)
    case "executor":
        return r.executorHandler.Handle(ctx, event, receiverBot)
    default:
        return ErrUnknownRole
    }
}
```

### 3.4 任务日志 (TaskLogger)

```go
type TaskRecord struct {
    ID         string
    TaskName   string
    Dispatcher string
    Executor   string
    User       string
    Status     string  // "pending" | "running" | "completed" | "failed"
    StartTime  time.Time
    EndTime    *time.Time
    Result     string
    Error      string
}
```

---

## 4. 消息流程

### 4.1 完整任务流程

```
┌─────────────┐
│  用户在群里  │
│  @分发机器人  │
└──────┬──────┘
       │
       ↓
┌─────────────────────────────────────────┐
│  DispatcherHandler.HandleMessage        │
│  1. 校验用户是否在白名单                   │
│  2. 解析用户指令（任务名 + 参数）           │
│  3. 校验任务是否在白名单                   │
│  4. 在机器人群 @执行机器人                  │
│  5. 创建任务记录（状态：pending）           │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│  机器人群消息                             │
│  @shell-executor-1 执行 deploy.sh prod   │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│  ExecutorHandler.HandleMessage           │
│  1. 校验发送者是否在 allowed_dispatchers  │
│  2. 校验脚本是否在 allowed_scripts        │
│  3. 执行脚本，捕获输出                     │
│  4. 在机器人群 @分发机器人 汇报结果        │
└──────────────┬──────────────────────────┘
               │
               ↓
┌─────────────────────────────────────────┐
│  DispatcherHandler.HandleExecutorReply   │
│  1. 解析执行机器人的回复                   │
│  2. 更新任务记录（状态：completed/failed） │
│  3. 回到原用户群，回复用户任务结果          │
└─────────────────────────────────────────┘
```

### 4.2 消息格式约定

**Dispatcher → Executor（机器人群）：**
```
@shell-executor-1 execute deploy.sh --env=prod
```

**Executor → Dispatcher（机器人群）：**
```
@task-dispatcher task_complete:deploy.sh exit_code:0 output:"部署成功"
```

**Dispatcher → User（用户群）：**
```
@用户 任务 [deploy.sh] 执行成功
输出：部署成功
```

---

## 5. 安全设计

### 5.1 分层授权

1. **Dispatcher 安全**
   - 用户白名单：只允许特定用户发起任务
   - 任务白名单：只允许执行预定义的任务
   - 消息来源校验：确保消息来自用户群

2. **Executor 安全**
   - 来源校验：只接受指定 Dispatcher 的命令
   - 脚本白名单：只允许执行预定义的脚本
   - 参数验证：严格验证传递给脚本的参数

### 5.2 安全校验实现

```go
// Executor Handler 中的校验
func (h *ExecutorHandler) HandleMessage(ctx context.Context, msg *lark.Event) error {
    // 1. 校验消息来源
    if !h.isFromAllowedDispatcher(msg.Sender.BotID) {
        return ErrUnauthorizedDispatcher
    }

    // 2. 校验任务是否在白名单
    if !h.isAllowedTask(msg.Content.TaskName) {
        return ErrTaskNotAllowed
    }

    // 3. 执行任务
    return h.executeTask(msg.Content)
}
```

---

## 6. 错误处理

### 6.1 错误类型

```go
type HandlerError struct {
    Code        string  // "AUTH_FAILED" | "TASK_NOT_ALLOWED" | "EXECUTION_FAILED"
    Message     string
    Cause       error
    ReplyToUser bool    // 是否需要回复给用户
}
```

### 6.2 错误处理策略

- **Dispatcher 错误**：回复用户友好提示，记录详细日志
- **Executor 错误**：向 Dispatcher 汇报错误信息
- **系统错误**：记录日志，回复通用提示

---

## 7. 项目结构

```
lark-bot-service/
├── cmd/
│   └── bot-service/
│       └── main.go              # 入口
├── internal/
│   ├── config/
│   │   ├── config.go
│   │   └── config.yaml
│   ├── bot/
│   │   ├── registry.go
│   │   ├── client.go
│   │   └── types.go
│   ├── router/
│   │   └── router.go
│   ├── handler/
│   │   ├── handler.go
│   │   ├── dispatcher.go
│   │   └── executor.go
│   ├── logger/
│   │   └── task_logger.go
│   └── common/
│       ├── message.go
│       └── errors.go
├── pkg/
│   └── lark/
│       └── client.go
├── configs/
│   └── bots.yaml
├── docs/
│   └── 2026-03-23-bot-service-design.md
├── go.mod
└── README.md
```

---

## 8. 核心依赖

```go
require (
    github.com/larksuite/oapi-sdk-go/v3 v3.0.20
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.0
    go.uber.org/zap v1.26.0
)
```

---

## 9. 启动命令

```bash
# 启动指定机器人
./bot-service start --bots=task-dispatcher,shell-executor-1

# 启动所有机器人
./bot-service start --all

# 查看任务日志
./bot-service logs --task=deploy.sh --status=failed
```

---

## 10. 测试策略

### 10.1 单元测试
- Config 加载与验证
- MessageRouter 路由逻辑
- Handler 业务逻辑（Mock SDK Client）
- TaskLogger CRUD

### 10.2 集成测试
- Mock 飞书事件，测试完整流程
- 错误场景模拟
- 多机器人协作场景

### 10.3 端到端测试（可选）
- 连接真实飞书测试环境

---

## 11. 可扩展性

### 11.1 添加新角色

```go
// 实现 MessageHandler 接口
type NewRoleHandler struct {
    // ...
}

func (h *NewRoleHandler) Handle(ctx context.Context, event *lark.Event, bot *BotClient) error {
    // 处理逻辑
}
```

### 11.2 未来增强

- 接入文本嵌入模型进行语义匹配
- 支持任务优先级队列
- 添加任务重试机制
- Web UI 管理界面

---

## 12. 配置示例

```yaml
bots:
  - name: "task-dispatcher"
    app_id: "cli_xxx"
    app_secret: "xxx"
    role: "dispatcher"
    user_whitelist: ["user_1", "user_2"]
    task_whitelist: ["deploy", "restart", "check_logs"]

  - name: "shell-executor-1"
    app_id: "cli_yyy"
    app_secret: "yyy"
    role: "executor"
    allowed_dispatchers: ["cli_xxx"]
    allowed_scripts: ["/opt/scripts/deploy.sh"]

robot_group_id: "oc_xxx"
```
