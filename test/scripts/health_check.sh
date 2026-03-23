#!/bin/bash
# 健康检查脚本

set -e

echo "==================================="
echo "系统健康检查"
echo "==================================="

# 检查磁盘空间
echo "[1/4] 检查磁盘空间..."
DISK_USAGE=$(df -h / | tail -1 | awk '{print $5}' | sed 's/%//')
if [ "$DISK_USAGE" -lt 80 ]; then
    echo "✓ 磁盘使用率: ${DISK_USAGE}% (正常)"
else
    echo "⚠ 磁盘使用率: ${DISK_USAGE}% (警告)"
fi

# 检查内存使用
echo "[2/4] 检查内存使用..."
MEM_USAGE=$(free | grep Mem | awk '{printf("%.0f", $3/$2 * 100)}')
echo "✓ 内存使用率: ${MEM_USAGE}%"

# 检查 CPU 负载
echo "[3/4] 检查 CPU 负载..."
LOAD=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,//')
echo "✓ CPU 负载: ${LOAD}"

# 检查关键服务
echo "[4/4] 检查关键服务..."
SERVICES=("nginx" "mysql" "redis")
for service in "${SERVICES[@]}"; do
    if systemctl is-active --quiet "$service" 2>/dev/null || pgrep -x "$service" > /dev/null 2>&1; then
        echo "✓ $service: 运行中"
    else
        echo "⚠ $service: 未运行 (可能未安装)"
    fi
done

echo "==================================="
echo "健康检查完成"
echo "检查时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "==================================="

exit 0
