#!/bin/bash
# 简单的测试脚本 - 返回固定输出

echo "测试脚本执行成功！"
echo "参数数量: $#"
if [ $# -gt 0 ]; then
    echo "参数列表:"
    for arg in "$@"; do
        echo "  - $arg"
    done
fi
echo "执行时间: $(date '+%Y-%m-%d %H:%M:%S')"
echo "主机名: $(hostname)"
echo "当前用户: $(whoami)"

exit 0
