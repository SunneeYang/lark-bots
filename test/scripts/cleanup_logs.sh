#!/bin/bash
# 日志清理脚本

set -e

LOG_DIR="${1:-/var/log}"
RETENTION_DAYS="${2:-7}"

echo "==================================="
echo "日志清理"
echo "==================================="

echo "日志目录: $LOG_DIR"
echo "保留天数: $RETENTION_DAYS"

TOTAL_SIZE_BEFORE=0
TOTAL_SIZE_AFTER=0

# 清理 .log 文件
echo "[1/3] 查找过期日志文件..."
OLD_LOGS=$(find "$LOG_DIR" -name "*.log" -type f -mtime +$RETENTION_DAYS 2>/dev/null || true)
FILE_COUNT=$(echo "$OLD_LOGS" | grep -c "^" || echo "0")

if [ "$FILE_COUNT" -eq 0 ]; then
    echo "✓ 没有找到过期日志"
else
    echo "找到 $FILE_COUNT 个过期日志文件"

    echo "[2/3] 计算占用空间..."
    while IFS= read -r log_file; do
        if [ -f "$log_file" ]; then
            size=$(du -b "$log_file" | cut -f1)
            TOTAL_SIZE_BEFORE=$((TOTAL_SIZE_BEFORE + size))
        fi
    done <<< "$OLD_LOGS"
    echo "原始大小: $(numfmt --to=iec $TOTAL_SIZE_BEFORE 2>/dev/null || echo ${TOTAL_SIZE_BEFORE} bytes)"

    echo "[3/3] 删除过期日志..."
    while IFS= read -r log_file; do
        if [ -f "$log_file" ]; then
            rm -f "$log_file"
            echo "✓ 已删除: $(basename $log_file)"
        fi
    done <<< "$OLD_LOGS"

    echo "==================================="
    echo "清理完成！"
    echo "删除文件数: $FILE_COUNT"
    echo "释放空间: $(numfmt --to=iec $TOTAL_SIZE_BEFORE 2>/dev/null || echo ${TOTAL_SIZE_BEFORE} bytes)"
    echo "==================================="
fi

echo "清理时间: $(date '+%Y-%m-%d %H:%M:%S')"

exit 0
