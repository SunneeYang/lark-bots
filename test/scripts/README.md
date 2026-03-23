# 测试脚本目录

此目录包含用于测试飞书机器人服务的 shell 脚本。

## 脚本列表

### 1. deploy.sh
部署应用脚本，模拟完整的部署流程：
- 拉取代码
- 安装依赖
- 运行测试
- 构建应用
- 重启服务

**用法：**
```bash
./deploy.sh [环境] [项目目录]

# 示例
./deploy.sh production /var/www/app
./deploy.sh staging /var/www/app-staging
```

### 2. health_check.sh
系统健康检查脚本，检查：
- 磁盘空间使用率
- 内存使用率
- CPU 负载
- 关键服务状态（nginx, mysql, redis）

**用法：**
```bash
./health_check.sh
```

### 3. backup.sh
数据库备份脚本：
- 创建备份目录
- 执行数据库备份
- 压缩备份文件

**用法：**
```bash
./backup.sh [备份目录] [数据库名]

# 示例
./backup.sh /var/backups myapp
./backup.sh /mnt/backups production_db
```

### 4. cleanup_logs.sh
日志清理脚本：
- 查找过期日志文件
- 计算占用空间
- 删除过期文件
- 报告释放空间

**用法：**
```bash
./cleanup_logs.sh [日志目录] [保留天数]

# 示例
./cleanup_logs.sh /var/log 7
./cleanup_logs.sh /var/www/app/logs 30
```

### 5. test.sh
简单测试脚本，返回基本信息：
- 参数数量和列表
- 执行时间
- 主机名和当前用户

**用法：**
```bash
./test.sh [参数1] [参数2] ...

# 示例
./test.sh
./test.sh arg1 arg2 arg3
```

### 6. fail.sh
模拟错误脚本，用于测试错误处理：
- 返回非零退出码
- 输出错误信息
- 提供故障排查建议

**用法：**
```bash
./fail.sh
```

## 测试配置

使用 `configs/bots.test.yaml` 配置文件来测试这些脚本：

```bash
# 启动测试服务
./bot-service start --config configs/bots.test.yaml --all

# 或只启动特定机器人
./bot-service start --config configs/bots.test.yaml --bots test-dispatcher,test-executor-1
```

## 权限设置

确保所有脚本具有执行权限：

```bash
chmod +x test/scripts/*.sh
```

## 测试场景

### 场景 1: 正常执行
```bash
# 在机器人群发送：
@test-dispatcher 执行 test.sh
```

### 场景 2: 带参数执行
```bash
@test-dispatcher 执行 deploy.sh staging /var/www/app-staging
```

### 场景 3: 错误处理
```bash
@test-dispatcher 执行 fail.sh
# 应该返回错误信息和退出码
```

### 场景 4: 白名单验证
```bash
@test-dispatcher 执行 /etc/passwd
# 应该被白名单拦截
```

## 注意事项

⚠️ **这些是测试脚本，仅用于开发和测试环境！**

在生产环境中使用时：
1. 根据实际需求修改脚本逻辑
2. 添加适当的错误处理
3. 实现日志记录
4. 配置监控和告警
5. 使用绝对路径
6. 设置适当的文件权限
7. 实现回滚机制
