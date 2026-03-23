# 飞书机器人服务

基于飞书 Go SDK 的多机器人协作服务，支持配置文件管理、启动参数控制、群聊消息通信和任务日志记录。

## 功能特性

- ✅ **配置文件管理**: 通过 YAML 配置文件管理多个机器人
- ✅ **灵活启动控制**: 支持启动参数控制启动机器人列表
- ✅ **角色分离设计**: Dispatcher 和 Executor 角色分离
- ✅ **分层安全机制**: 白名单 + 来源校验的多层安全防护
- ✅ **完整任务日志**: 记录所有任务的执行状态和结果
- ✅ **消息路由**: 基于角色的自动消息路由
- ✅ **错误处理**: 统一的错误类型和处理机制

## 架构设计

### 系统架构

```
┌─────────────────────────────────────────────────────────────────┐
│                         用户群 (User Group)                       │
│  用户: @task-dispatcher 执行 deploy.sh                            │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Dispatcher (分发机器人)                         │
│  - 验证用户白名单                                                   │
│  - 验证任务白名单                                                   │
│  - 分发任务到机器人群                                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                    机器人群 (Robot Group)                         │
│  Dispatcher: @shell-executor execute /opt/scripts/deploy.sh      │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Executor (执行机器人)                           │
│  - 验证 Dispatcher 白名单                                          │
│  - 验证脚本白名单                                                   │
│  - 执行脚本并返回结果                                               │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         ↓
┌─────────────────────────────────────────────────────────────────┐
│                    Dispatcher (回复用户)                           │
│  - 收集执行结果                                                     │
│  - 回复用户任务结果                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### 角色说明

#### Dispatcher (分发机器人)

**职责:**
- 加入用户群，接收用户指令
- 验证用户和任务白名单
- 在机器人群分发任务给执行机器人
- 收集执行结果并回复用户

**配置:**
```yaml
- name: "task-dispatcher"
  app_id: "cli_xxx"
  app_secret: "xxx"
  role: "dispatcher"
```

**安全控制:**
- 用户白名单 (`user_whitelist`)
- 任务白名单 (`task_whitelist`)

#### Executor (执行机器人)

**职责:**
- 只在机器人群，接收分发机器人指令
- 执行脚本等任务
- 返回执行结果

**配置:**
```yaml
- name: "shell-executor"
  app_id: "cli_yyy"
  app_secret: "yyy"
  role: "executor"
  allowed_dispatchers:
    - "cli_xxx"  # 只接受这些 dispatcher 的命令
  allowed_scripts:
    - "/opt/scripts/deploy.sh"  # 只允许执行这些脚本
```

**安全控制:**
- Dispatcher 白名单 (`allowed_dispatchers`)
- 脚本白名单 (`allowed_scripts`)

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
1. **robot_group_id**: 机器人群 ID（所有机器人必须加入此群）
2. **task_whitelist**: 允许执行的任务列表
3. **user_whitelist**: 允许发起任务的用户列表
4. **bots**: 机器人配置列表

**获取飞书应用信息:**
- 访问 [飞书开放平台](https://open.feishu.cn/app)
- 创建应用或选择已有应用
- 在应用凭证页面查看 `App ID` 和 `App Secret`

**获取群 ID:**
- 在飞书群设置中查看群 ID

### 3. 启动服务

**开发模式启动:**
```bash
# 启动所有机器人
go run cmd/bot-service/main.go start --all

# 启动指定机器人
go run cmd/bot-service/main.go start --bots=task-dispatcher,shell-executor

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

**用户在群里发送:**
```
@task-dispatcher 执行 deploy.sh
```

**执行流程:**
1. Dispatcher 验证用户是否在白名单中
2. Dispatcher 验证任务是否在白名单中
3. Dispatcher 在机器人群发送: `@shell-executor execute /opt/scripts/deploy.sh`
4. Executor 验证来源和脚本白名单
5. Executor 执行脚本并返回结果
6. Dispatcher 收集结果并回复用户

## 配置说明

### 完整配置示例

```yaml
# 机器人群 ID
robot_group_id: "oc_xxxxxxxxxxxxxxxxx"

# 任务白名单
task_whitelist:
  - "deploy"
  - "restart"
  - "check_logs"

# 用户白名单
user_whitelist:
  - "ou_xxxxxxxxxxxxxxxxx"
  - "ou_yyyyyyyyyyyyyyyyyy"

# 机器人配置
bots:
  # 分发机器人
  - name: "task-dispatcher"
    app_id: "cli_xxxxxxxxxxxxxxxxx"
    app_secret: "xxxxxxxxxxxxxxxxxxxx"
    role: "dispatcher"

  # 执行机器人
  - name: "shell-executor"
    app_id: "cli_yyyyyyyyyyyyyyyyyy"
    app_secret: "yyyyyyyyyyyyyyyyyy"
    role: "executor"
    allowed_dispatchers:
      - "cli_xxxxxxxxxxxxxxxxx"
    allowed_scripts:
      - "/opt/scripts/deploy.sh"
      - "/opt/scripts/restart.sh"
```

### 安全配置建议

#### 1. 用户白名单
严格限制可以发起任务的用户，避免未授权访问。

```yaml
user_whitelist:
  - "ou_xxx"  # 飞书用户 ID
```

#### 2. 任务白名单
严格限制可以执行的任务名称，防止执行未授权的任务。

```yaml
task_whitelist:
  - "deploy"
  - "restart"
```

#### 3. Executor 安全校验

**Dispatcher 白名单:**
只接受特定 dispatcher 的命令，防止伪造指令。

```yaml
allowed_dispatchers:
  - "cli_xxx"  # dispatcher 的 app_id
```

**脚本白名单:**
只允许执行白名单中的脚本，防止任意命令执行。

```yaml
allowed_scripts:
  - "/opt/scripts/deploy.sh"
```

#### 4. 脚本权限
确保脚本文件权限正确，推荐设置为 `750`。

```bash
chmod 750 /opt/scripts/deploy.sh
```

#### 5. 网络隔离
机器人之间在专门的机器人群通信，与用户群隔离。

## 开发指南

### 项目结构

```
lark-bot-service/
├── cmd/
│   └── bot-service/
│       └── main.go              # 入口，CLI 命令定义
├── internal/
│   ├── config/
│   │   ├── config.go            # 配置结构体定义
│   │   ├── loader.go            # 配置加载逻辑
│   │   └── validator.go         # 配置验证逻辑
│   ├── bot/
│   │   ├── types.go             # 核心类型定义
│   │   ├── registry.go          # 机器人注册表
│   │   └── client.go            # 飞书 SDK 客户端封装
│   ├── router/
│   │   └── router.go            # 消息路由器
│   ├── handler/
│   │   ├── handler.go           # Handler 接口定义
│   │   ├── dispatcher.go        # DispatcherHandler 实现
│   │   └── executor.go          # ExecutorHandler 实现
│   ├── logger/
│   │   └── task_logger.go       # 任务日志记录器
│   └── common/
│       ├── message.go           # 消息发送通用函数
│       └── errors.go            # 错误定义
├── configs/
│   └── bots.yaml.example        # 配置文件示例
├── test/
│   └── mocks/
│       └── lark_mock.go         # 飞书 SDK Mock
├── go.mod
├── go.sum
└── README.md
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行测试并查看覆盖率
go test -cover ./...

# 查看详细测试输出
go test -v ./...

# 生成覆盖率报告
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### 代码规范

```bash
# 格式化代码
go fmt ./...

# 静态检查
go vet ./...

# 使用 golangci-lint
golangci-lint run
```

### 添加新功能

1. **定义配置结构** (在 `internal/config/config.go`)
2. **实现配置验证** (在 `internal/config/validator.go`)
3. **实现核心逻辑**
4. **编写单元测试**
5. **集成到主程序** (在 `cmd/bot-service/main.go`)

## 常见问题

### 1. 如何获取飞书用户 ID？

**方法 1:** 在飞书管理后台查看用户信息

**方法 2:** 通过 API 获取
```go
// 使用飞书 SDK 获取用户信息
```

### 2. 如何添加新的执行脚本？

1. 将脚本放到指定目录（如 `/opt/scripts/`）
2. 设置正确的文件权限（`chmod 750 script.sh`）
3. 在配置文件的 `allowed_scripts` 中添加脚本路径
4. 重启服务

### 3. 如何排查启动失败？

1. 检查配置文件语法是否正确
2. 检查 app_id 和 app_secret 是否正确
3. 查看日志输出的错误信息
4. 运行配置验证命令（如果有）

### 4. 支持多少个机器人？

理论上支持无限个机器人，建议：
- Dispatcher: 1-2 个（避免分发冲突）
- Executor: 根据业务需求添加，实现职责分离

## 技术栈

- **语言**: Go 1.21
- **飞书 SDK**: [larksuite/oapi-sdk-go/v3](https://github.com/larksuite/oapi-sdk-go) v3.0.20
- **CLI 框架**: [spf13/cobra](https://github.com/spf13/cobra) v1.8.0
- **配置管理**: [spf13/viper](https://github.com/spf13/viper) v1.18.0
- **日志**: [uber-go/zap](https://github.com/uber-go/zap) v1.26.0
- **YAML 解析**: [gopkg.in/yaml.v3](https://gopkg.in/yaml.v3) v3.0.1

## 安全性

本服务实现了多层安全机制：

1. **用户白名单**: 只允许授权用户发起任务
2. **任务白名单**: 只允许执行授权的任务
3. **来源验证**: Executor 只接受授权 Dispatcher 的命令
4. **脚本白名单**: 只允许执行授权的脚本
5. **群组隔离**: 机器人在专门的机器人群通信

## 后续扩展

- [ ] 飞书 SDK 实际集成和事件监听
- [ ] 消息发送功能实现
- [ ] 数据库持久化（任务记录）
- [ ] Web UI 管理界面
- [ ] 更多执行器类型（Docker、K8s 等）
- [ ] 任务调度和定时执行
- [ ] 监控和告警

## License

MIT

## 贡献

欢迎提交 Issue 和 Pull Request！

## 联系方式

如有问题，请提交 Issue。
