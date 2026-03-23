# 飞书机器人服务

多机器人协作服务，支持配置文件管理和任务分发。

## 快速开始

```bash
# 复制配置文件
cp configs/bots.yaml.example configs/bots.yaml

# 编辑配置
vim configs/bots.yaml

# 启动服务
./bot-service start --bots=task-dispatcher,shell-executor-1
```

## 配置说明

详见 `configs/bots.yaml.example`

## 开发

```bash
# 运行测试
go test ./...

# 构建
go build -o bot-service cmd/bot-service/main.go
```
