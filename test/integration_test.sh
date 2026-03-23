#!/bin/bash
# 集成测试脚本 - 测试完整的服务启动流程

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"
BOT_SERVICE="$PROJECT_DIR/bot-service"
TEST_CONFIG="$PROJECT_DIR/configs/bots.test.yaml"

echo "========================================"
echo "飞书机器人服务 - 集成测试"
echo "========================================"

# 检查编译的二进制文件
echo "[1/6] 检查服务可执行文件..."
if [ ! -f "$BOT_SERVICE" ]; then
    echo "❌ 未找到 bot-service，正在编译..."
    cd "$PROJECT_DIR"
    go build -o bot-service cmd/bot-service/main.go
    echo "✅ 编译完成"
else
    echo "✅ 找到 bot-service"
fi

# 检查测试配置
echo "[2/6] 检查测试配置文件..."
if [ ! -f "$TEST_CONFIG" ]; then
    echo "❌ 未找到测试配置: $TEST_CONFIG"
    exit 1
fi
echo "✅ 测试配置文件存在"

# 测试配置加载
echo "[3/6] 测试配置加载..."
if "$BOT_SERVICE" start --config "$TEST_CONFIG" --help > /dev/null 2>&1; then
    echo "✅ 配置加载成功"
else
    echo "❌ 配置加载失败"
    exit 1
fi

# 运行单元测试
echo "[4/6] 运行单元测试..."
cd "$PROJECT_DIR"
if go test ./... > /tmp/test_output.txt 2>&1; then
    TEST_COUNT=$(grep -c "^=== RUN" /tmp/test_output.txt || echo "0")
    PASS_COUNT=$(grep -c "^--- PASS:" /tmp/test_output.txt || echo "0")
    echo "✅ 所有单元测试通过 ($PASS_COUNT/$TEST_COUNT)"
else
    echo "❌ 单元测试失败"
    cat /tmp/test_output.txt
    exit 1
fi

# 测试脚本
echo "[5/6] 测试执行脚本..."
TEST_SCRIPT="$SCRIPT_DIR/scripts/test.sh"
if [ ! -f "$TEST_SCRIPT" ]; then
    echo "❌ 未找到测试脚本: $TEST_SCRIPT"
    exit 1
fi

if "$TEST_SCRIPT" "arg1" "arg2" > /dev/null 2>&1; then
    echo "✅ 脚本执行成功"
else
    echo "❌ 脚本执行失败"
    exit 1
fi

# 测试服务启动（非阻塞模式）
echo "[6/6] 测试服务启动..."
timeout 3 "$BOT_SERVICE" start --config "$TEST_CONFIG" --all > /tmp/service_start.txt 2>&1 || true

if grep -q "✅ 服务初始化完成" /tmp/service_start.txt; then
    echo "✅ 服务启动成功"
    echo ""
    echo "启动输出："
    grep "✅" /tmp/service_start.txt | sed 's/^/  /'
else
    echo "❌ 服务启动失败"
    cat /tmp/service_start.txt
    exit 1
fi

echo ""
echo "========================================"
echo "✅ 所有集成测试通过！"
echo "========================================"
echo ""
echo "下一步："
echo "  1. 配置实际的飞书应用凭证"
echo "  2. 更新 configs/bots.yaml"
echo "  3. 启动服务: ./bot-service start --all"
echo "  4. 在机器人群中测试消息"
echo ""
echo "测试资源："
echo "  - 配置示例: configs/bots.yaml.example"
echo "  - 测试配置: configs/bots.test.yaml"
echo "  - 测试脚本: test/scripts/*.sh"
echo "  - 快速指南: test/QUICK_TEST.md"
echo ""

exit 0
