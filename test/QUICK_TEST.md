# 快速测试指南

本文档介绍如何快速测试飞书机器人服务。

## 前提条件

1. 已编译服务：`go build -o bot-service cmd/bot-service/main.go`
2. 已创建测试配置：`configs/bots.test.yaml`
3. 测试脚本已就位：`test/scripts/*.sh`

## 测试步骤

### 1. 验证测试脚本

```bash
# 测试所有脚本
./test/scripts/test.sh hello world
./test/scripts/deploy.sh production /var/www/app
./test/scripts/health_check.sh
./test/scripts/backup.sh /tmp/backups testdb
./test/scripts/cleanup_logs.sh /var/log 7
./test/scripts/fail.sh
```

### 2. 测试配置加载

```bash
# 验证配置文件格式是否正确
./bot-service start --config configs/bots.test.yaml --help
```

### 3. 测试服务启动

```bash
# 启动所有机器人（应该会阻塞，等待 Ctrl+C）
./bot-service start --config configs/bots.test.yaml --all

# 或在后台运行
./bot-service start --config configs/bots.test.yaml --all &
PID=$!
echo "服务 PID: $PID"

# 稍后停止
kill $PID
```

### 4. 验证启动输出

启动时应该看到类似输出：

```
✅ 配置加载成功
  - 注册机器人: test-dispatcher (dispatcher)
  - 注册机器人: test-executor-1 (executor)
  - 注册机器人: test-executor-2 (executor)
✅ 将启动 3 个机器人
  - test-dispatcher (dispatcher)
  - test-executor-1 (executor)
  - test-executor-2 (executor)
✅ 服务初始化完成
⏳ 等待飞书事件...
提示: 按 Ctrl+C 退出
```

### 5. 单元测试

```bash
# 运行所有单元测试
go test ./... -v

# 运行特定包的测试
go test ./internal/handler -v
go test ./internal/router -v
```

## 集成测试（需要飞书环境）

### 设置测试环境

1. 在飞书开放平台创建测试应用
2. 获取测试用的 app_id 和 app_secret
3. 创建测试用的机器人群
4. 更新 `configs/bots.test.yaml` 中的凭证信息

### 测试消息流程

**场景 1: 用户发起任务**
```
用户在群里发送: "@test-dispatcher 执行 test.sh"
→ Dispatcher 验证用户白名单
→ Dispatcher 验证任务白名单
→ Dispatcher 发送任务给 Executor
→ Executor 执行脚本
→ Executor 返回结果
→ Dispatcher 转发结果给用户
```

**场景 2: 带参数的任务**
```
用户发送: "@test-dispatcher 执行 deploy.sh staging /var/www/app-staging"
→ 参数会被传递给脚本
→ 脚本接收 $1=staging, $2=/var/www/app-staging
```

**场景 3: 错误处理**
```
用户发送: "@test-dispatcher 执行 fail.sh"
→ 脚本返回非零退出码
→ Executor 捕获错误
→ 返回错误信息给用户
```

**场景 4: 白名单拦截**
```
用户发送: "@test-dispatcher 执行 /etc/passwd"
→ 脚本不在白名单中
→ Dispatcher 拒绝执行
```

## 调试技巧

### 查看详细日志

```bash
# 设置环境变量启用调试日志
export DEBUG=true
./bot-service start --config configs/bots.test.yaml --all
```

### 测试单个组件

```bash
# 测试配置加载
go run -c 'package main; import "fmt"; import "github.com/yourname/lark-bot-service/internal/config"; func main() { cfg, _ := config.LoadConfig("configs/bots.test.yaml"); fmt.Printf("%+v\n", cfg) }'

# 测试路由器
go test ./internal/router -v -run TestMessageRouter_Route

# 测试 Handler
go test ./internal/handler -v -run TestDispatcherHandler_HandleUserMessage
```

### 常见问题

**问题 1: 配置文件加载失败**
```bash
# 检查文件路径
ls -la configs/bots.test.yaml

# 验证 YAML 语法
python3 -c "import yaml; yaml.safe_load(open('configs/bots.test.yaml'))"
```

**问题 2: 脚本没有执行权限**
```bash
# 添加执行权限
chmod +x test/scripts/*.sh

# 验证权限
ls -l test/scripts/
```

**问题 3: 服务启动失败**
```bash
# 检查端口占用
lsof -i :8080

# 检查日志
tail -f /var/log/bot-service.log
```

## 性能测试

### 并发测试

```bash
# 启动服务
./bot-service start --config configs/bots.test.yaml --all &
PID=$!

# 模拟并发请求（需要实际的飞书事件）
for i in {1..100}; do
  echo "并发请求 $i"
  # 这里需要实际的飞书 SDK 发送消息
done

# 等待完成
wait $PID
```

### 脚本执行时间

```bash
# 测试脚本执行时间
time ./test/scripts/deploy.sh production /var/www/app

# 应该在几秒内完成
```

## 清理测试环境

```bash
# 停止后台服务
pkill -f bot-service

# 清理测试备份
rm -rf /tmp/backups

# 清理测试日志
rm -f /tmp/bot-service.log
```

## 下一步

测试通过后，可以：

1. ✅ 部署到测试环境
2. ✅ 集成实际的飞书事件监听
3. ✅ 添加消息发送功能
4. ✅ 配置生产环境的凭证
5. ✅ 设置监控和告警

---

**提示：** 当前实现中的事件监听是 TODO 占位，需要集成实际的飞书 SDK 来接收和处理飞书事件。
