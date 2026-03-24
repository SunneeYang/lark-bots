# 飞书机器人服务

基于飞书 Go SDK 的多机器人协作服务，支持配置文件管理、启动参数控制、群聊消息通信和任务日志记录。

## 功能特性

- ✅ **配置文件管理**: 通过 YAML 配置文件管理多个机器人
- ✅ **灵活启动控制**: 支持启动参数控制启动机器人列表
- ✅ **角色分离设计**: Dispatcher 和 Executor 角色分离
- ✅ **分层安全机制**: 白名单 + 来源校验的多层安全防护
- ✅ **完整任务日志**: 记录所有任务的执行状态和结果
- ✅ **消息路由**: 基于角色的自动消息路由
- ✅ **轮询机制**: Executor 通过轮询获取群消息，解决机器人间事件不互通问题
- ✅ **错误处理**: 统一的错误类型和处理机制

## 架构设计

### 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户群 (User Group)                       │
│  用户: 私聊 @dispatcher 执行 run_test                            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Dispatcher (分发机器人)                         │
│  - 验证用户白名单                                                   │
│  - 验证任务白名单                                                   │
│  - 发送任务到机器人群（纯文本）                                     │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ 发送纯文本消息（如 "run_test"）
┌─────────────────────────────────────────────────────────────────┐
│                    机器人群 (Robot Group)                         │
│  Dispatcher: 发送纯文本任务名                                      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓ 每秒轮询获取新消息
┌─────────────────────────────────────────────────────────────────┐
│                    Executor (执行机器人)                           │
│  - 轮询获取群消息                                                  │
│  - 验证 Dispatcher 来源（app_id）                                  │
│  - 验证任务名白名单                                                │
│  - 执行脚本并汇报结果到机器人群                                     │
└─────────────────────────────────────────────────────────────────┘
```

### 核心机制说明

#### 为什么使用轮询？

飞书有一个限制：**机器人发送的消息，其他机器人不会收到 `im.message.receive_v1` 事件**。

因此采用了**轮询机制**：
- Executor 每秒调用飞书 `im.v1.messages` API 获取群消息
- 通过检查消息的 `sender.sender_type` 和 `sender.id` 判断是否来自 Dispatcher
- 这种方式可以获取到**所有消息包括机器人发送的消息**

#### 消息流转

```
用户 → Dispatcher（私聊）→ 机器人群（纯文本任务名）
                                   ↓
                         Executor 轮询获取消息
                                   ↓
                         匹配 dispatcher + 任务名
                                   ↓
                         执行脚本 → 汇报结果到机器人群
```

### 角色说明

#### Dispatcher (分发机器人)

**职责:**
- 加入用户群，接收用户指令（私聊或 @mention）
- 验证用户和任务白名单
- 在机器人群发送纯文本任务名

**配置:**
```yaml
- name: "dispatcher"
  app_id: "cli_xxx"
  app_secret: "xxx"
  role: "dispatcher"
  allowed_users:
    - "ou_xxx"  # 允许的用户列表
```

**安全控制:**
- 用户白名单 (`allowed_users`)
- 任务白名单（自动从所有 executor 的 task_scripts 合并）

#### Executor (执行机器人)

**职责:**
- 轮询获取机器人群消息
- 验证 Dispatcher 来源（按 app_id）
- 验证任务名白名单
- 执行脚本并汇报结果

**配置:**
```yaml
- name: "dev-executor"
  app_id: "cli_yyy"
  app_secret: "yyy"
  role: "executor"
  allowed_dispatchers:
    - "cli_xxx"  # 只接受这些 dispatcher 的 app_id
  task_scripts:
    deploy: "/path/to/deploy.sh"
    run_test: "/path/to/test.sh"
```

**安全控制:**
- Dispatcher 白名单 (`allowed_dispatchers`) - 按 app_id 校验
- 任务白名单 (`task_scripts`) - 每个 executor 只执行自己的任务

## 快速开始

### 1. 环境准备

**系统要求:**
- Go 1.21+
- 飞书开放平台账号

**安装依赖:**
```bash
go mod download
```

### 2. 配置

**复制配置文件模板:**
```bash
cp configs/bots.yaml.example configs/bots.yaml
```

**编辑配置文件:**
```bash
vim configs/bots.yaml
```

**配置说明:**
1. `robot_group_id`: 机器人群 ID（所有机器人必须加入此群）
2. `allowed_users`: 允许发起任务的用户列表（仅 dispatcher）
3. `task_scripts`: 任务名 → 脚本路径映射（executor）
4. `bots`: 机器人配置列表

**获取飞书应用信息:**
- 访问 [飞书开放平台](https://open.feishu.cn/app)
- 创建应用或选择已有应用
- 在应用凭证页面查看 `App ID` 和 `App Secret`

**配置 Executor 权限:**
- 权限管理 → 申请 `im:message:readonly`（读取历史消息）
- 事件订阅 → 订阅 `im.message.receive_v1`

### 3. 启动服务

**开发模式启动:**
```bash
# 启动所有机器人
go run cmd/bot-service/main.go start --all

# 启动指定机器人
go run cmd/bot-service/main.go start --bots=dispatcher,dev-executor

# 使用自定义配置文件
go run cmd/bot-service/main.go start --config=/path/to/config.yaml
```

**生产模式启动:**
```bash
# 构建可执行文件
go build -o bot-service cmd/bot-service/main.go

# 启动服务
./bot-service start --all
```

**启动参数说明:**
- `--config, -c`: 配置文件路径（默认: `configs/bots.yaml`）
- `--bots, -b`: 要启动的机器人列表（逗号分隔）
- `--all`: 启动所有机器人

### 4. 使用示例

**用户在私聊发送:**
```
run_test
```

**执行流程:**
1. Dispatcher 接收私聊消息
2. Dispatcher 验证用户是否在白名单中
3. Dispatcher 验证任务是否在白名单中
4. Dispatcher 在机器人群发送纯文本: `run_test`
5. Executor 轮询获取群消息
6. Executor 验证 sender.app_id 是 dispatcher
7. Executor 验证任务名在 task_scripts 中
8. Executor 执行脚本
9. Executor 汇报结果到机器人群

## 配置说明

### 完整配置示例

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
    task_scripts:
      deploy: "/opt/scripts/deploy.sh"
      run_test: "/opt/scripts/test.sh"
      check_logs: "/opt/scripts/check_logs.sh"
```

### 安全配置建议

#### 1. 用户白名单
严格限制可以发起任务的用户，避免未授权访问。

```yaml
allowed_users:
  - "ou_xxx"  # 飞书用户 ID
```

#### 2. Executor 安全校验

**Dispatcher 白名单:**
只接受特定 dispatcher 的命令，防止伪造指令。按 `app_id` 校验。

```yaml
allowed_dispatchers:
  - "cli_xxx"  # dispatcher 的 app_id
```

**任务白名单:**
每个 executor 只执行自己配置的任务，实现任务隔离。

```yaml
task_scripts:
  deploy: "/opt/scripts/deploy.sh"
  run_test: "/opt/scripts/test.sh"
```

#### 3. 脚本权限
确保脚本文件权限正确，推荐设置为 `750`。

```bash
chmod 750 /opt/scripts/deploy.sh
```

#### 4. 网络隔离
机器人在专门的机器人群通信，与用户群隔离。

## 项目结构

```
lark-bot-service/
├── cmd/
│   └── bot-service/
│       └── main.go              # 入口，CLI 命令定义
├── internal/
│   ├── config/
│   │   ├── config.go           # 配置结构体定义
│   │   └── validator.go       # 配置验证逻辑
│   ├── bot/
│   │   ├── types.go           # 核心类型定义
│   │   ├── registry.go        # 机器人注册表
│   │   └── client.go         # 飞书 SDK 客户端封装
│   ├── router/
│   │   └── router.go         # 消息路由器
│   ├── handler/
│   │   ├── handler.go        # Handler 接口定义
│   │   ├── dispatcher.go     # DispatcherHandler 实现
│   │   ├── executor.go       # ExecutorHandler 实现
│   │   └── poller.go        # 消息轮询器（核心机制）
│   ├── logger/
│   │   └── task_logger.go   # 任务日志记录器
│   └── common/
│       ├── sender.go         # 消息发送器
│       └── event.go          # 事件解析工具
├── configs/
│   ├── bots.yaml.example     # 配置文件示例
│   └── bots.yaml             # 实际配置（包含密钥）
├── test/
│   └── scripts/              # 测试脚本
├── go.mod
├── go.sum
└── README.md
```

## 开发指南

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 查看详细测试输出
go test -v ./...
```

### 代码规范

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...
```

### 添加新功能

1. **定义配置结构** (在 `internal/config/config.go`)
2. **实现配置验证** (在 `internal/config/validator.go`)
3. **实现核心逻辑**
4. **编写单元测试**
5. **集成到主程序** (在 `cmd/bot-service/main.go`)

## 常见问题

### 1. 为什么 Executor 使用轮询而不是 WebSocket 事件？

飞书限制：**机器人发送的消息，其他机器人不会收到 `im.message.receive_v1` 事件**。

轮询 API `im.v1.messages` 可以获取所有消息包括机器人发送的消息，因此采用轮询机制。

### 2. 轮询频率限制？

飞书 API 限制：50 QPS（每秒 50 次请求）。

当前配置每秒轮询 1 次，远低于限制，完全安全。

### 3. 如何添加新的执行脚本？

1. 将脚本放到指定目录（如 `/opt/scripts/`）
2. 设置正确的文件权限（`chmod 750 script.sh`）
3. 在配置文件的 `task_scripts` 中添加任务名 → 脚本路径映射
4. 重启服务

### 4. 支持多少个 Executor？

支持多个 Executor，每个 Executor 执行不同的任务。轮询请求频率 = executor 数量 × 1 QPS，远低于 50 QPS 限制。

## 技术栈

- **语言**: Go 1.21+
- **飞书 SDK**: [larksuite/oapi-sdk-go/v3](https://github.com/larksuite/oapi-sdk-go) v3.5.3
- **CLI 框架**: [spf13/cobra](https://github.com/spf13/cobra) v1.8.0
- **YAML 解析**: [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) v3.0.1

## 安全性

本服务实现了多层安全机制：

1. **用户白名单**: 只允许授权用户发起任务
2. **任务白名单**: 只允许执行授权的任务
3. **来源验证**: Executor 只接受授权 Dispatcher 的命令（按 app_id）
4. **任务隔离**: 每个 Executor 只执行自己配置的任务
5. **群组隔离**: 机器人在专门的机器人群通信

## License

MIT
