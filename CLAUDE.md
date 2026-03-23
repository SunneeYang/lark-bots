# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 构建与测试命令

```bash
# 构建
go build -o bot-service cmd/bot-service/main.go

# 运行测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 运行单个测试文件
go test -v ./internal/handler/dispatcher_test.go ./internal/handler/dispatcher.go

# 代码格式化与检查
go fmt ./...
go vet ./...

# 集成测试
./test/integration_test.sh
```

## 开发命令

```bash
# 启动所有机器人
go run cmd/bot-service/main.go start --all

# 启动指定机器人
go run cmd/bot-service/main.go start --bots=dispatcher,dev-executor

# 获取机器人 OpenID
go run cmd/bot-service/main.go fetch-openid dispatcher dev-executor
```

## 架构设计

### 角色分离模式

机器人服务采用 **Dispatcher + Executor** 双角色模式：

- **Dispatcher（分发机器人）**：驻留在用户群，接收用户 @ 指令，校验用户/任务白名单，向机器人群分发任务
- **Executor（执行机器人）**：仅驻留在机器人群，校验分发者来源和脚本白名单，执行脚本，通过 @mention 汇报结果

### 消息流转

```
用户 (@Dispatcher) → Dispatcher (校验白名单) → 机器人群 → Executor (校验来源、执行脚本) → Dispatcher (接收结果)
```

### WebSocket 事件处理

使用飞书 SDK WebSocket（`larksuite/oapi-sdk-go/v3/ws`）实现双向长连接：
- 每个机器人使用自己的 `AppID`/`AppSecret` 创建独立 `ws.Client`
- 事件分发器处理 `im.message.receive_v1`，同时支持 P1（私聊/群聊）和 P2（@mention）消息格式
- 在 `main.go:startBotWSClient` 中以 goroutine 方式启动

### 路由器模式

`MessageRouter` 按以下顺序路由事件：
1. 优先精确匹配机器人名称（支持每个 executor 独立的 handler）
2. 回退到角色匹配（dispatcher/executor）

### 核心类型

- `BotClient`：每个机器人的状态，包含飞书 SDK 客户端、角色、OpenID、允许的分发者列表
- `BotRegistry`：线程安全的注册表，使用 `sync.RWMutex`，按名称/AppID/角色索引
- `TaskRecord`：内存任务日志，使用 `sync.RWMutex` 保护

### 配置模型

- YAML 配置，每个机器人 `BotConfig`（app_id、app_secret、role、open_id）
- Dispatcher 特有：`allowed_users`、`allowed_tasks`
- Executor 特有：`allowed_dispatchers`、`task_scripts`（任务名 → 脚本路径映射）
- 全局：`robot_group_id`
- `config/validator.go` 验证：至少一个 dispatcher、名称唯一、角色合法、引用的 dispatcher 存在

## 配置文件

- `configs/bots.yaml.example`：带注释的模板，包含安全配置说明
- `configs/bots.yaml`：实际配置（包含密钥，禁止提交）
- `configs/bots.test.yaml`：测试配置，使用模拟凭证
