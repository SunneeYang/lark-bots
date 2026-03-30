#!/bin/bash
# 根据用户姓名查找 bot-service 的 OpenID
#
# 用法:
#   ./scripts/get_user_openid.sh "张三"
#   ./scripts/get_user_openid.sh "张三" "李四" "王五"

set -e

if [ $# -eq 0 ]; then
    echo "用法: $0 <姓名1> [姓名2] [姓名3] ..."
    echo ""
    echo "示例:"
    echo "  $0 \"张三\""
    echo "  $0 \"张三\" \"李四\" \"王五\""
    exit 1
fi

BOT_APP_ID="cli_a939d37873b8dcc0"
BOT_APP_SECRET="GfygeKKCanXMGrsn0VHTTegRbtxo0J8m"
CONFIG_BACKUP="/tmp/lark-config-backup-$(date +%s).json"

# 保存并切换配置
cp ~/.lark-cli/config.json "$CONFIG_BACKUP" 2>/dev/null || true

cleanup() {
    cp "$CONFIG_BACKUP" ~/.lark-cli/config.json 2>/dev/null || true
    rm -f "$CONFIG_BACKUP"
}
trap cleanup EXIT

for TARGET_NAME in "$@"; do
    echo "======================================"
    echo "🔍 查询: $TARGET_NAME"
    echo "======================================"

    # 步骤 1: 用 user 身份搜索，获取 user_id
    echo "① user 身份搜索..."

    # 恢复 user 配置
    cp "$CONFIG_BACKUP" ~/.lark-cli/config.json 2>/dev/null || true

    SEARCH=$(lark-cli contact +search-user --query "$TARGET_NAME" --as user --page-size 10 --format json 2>&1)

    USER_COUNT=$(echo "$SEARCH" | jq -r '.data.users | length // 0')

    if [ "$USER_COUNT" = "0" ] || [ "$USER_COUNT" = "null" ]; then
        echo "❌ 未找到 \"$TARGET_NAME\""
        echo ""
        continue
    fi

    # 精确匹配
    EXACT=$(echo "$SEARCH" | jq -r --arg name "$TARGET_NAME" '.data.users[] | select(.name == $name)')
    EXACT_COUNT=$(echo "$EXACT" | jq -s 'length')

    if [ "$EXACT_COUNT" = "1" ]; then
        SELECTED="$EXACT"
    elif [ "$EXACT_COUNT" -gt 1 ]; then
        echo "找到多个精确匹配:"
        echo "$EXACT" | jq -r '.name + " (" + .department_ids[0] + ")"' | nl -w2 -s'. '
        echo ""
        read -p "请选择序号: " CHOICE
        SELECTED=$(echo "$EXACT" | jq -s ".[$((CHOICE-1))]")
    else
        echo "未精确匹配到 \"$TARGET_NAME\"，模糊匹配结果:"
        echo "$SEARCH" | jq -r '.data.users[] | .name + " (" + .department_ids[0] + ")"' | nl -w2 -s'. '
        echo ""
        read -p "请选择序号: " CHOICE
        SELECTED=$(echo "$SEARCH" | jq -r --argjson i "$((CHOICE-1))" '.data.users[$i]')
    fi

    USER_ID=$(echo "$SELECTED" | jq -r '.user_id')
    MATCH_NAME=$(echo "$SELECTED" | jq -r '.name')

    echo "   姓名: $MATCH_NAME"
    echo "   user_id: $USER_ID"

    # 步骤 2: 切换到 bot 配置，获取 bot-service 的 OpenID
    echo "② bot 身份查询 OpenID..."

    cat > ~/.lark-cli/config.json << EOF
{
  "apps": [
    {
      "appId": "$BOT_APP_ID",
      "appSecret": "$BOT_APP_SECRET",
      "brand": "feishu",
      "lang": "zh"
    }
  ]
}
EOF

    BOT_INFO=$(lark-cli api GET "/open-apis/contact/v3/users/$USER_ID" --as bot --params '{"user_id_type":"user_id"}' 2>&1)

    if echo "$BOT_INFO" | jq -e '.code == 0' > /dev/null 2>&1; then
        BOT_OPENID=$(echo "$BOT_INFO" | jq -r '.data.user.open_id')
        EMAIL=$(echo "$BOT_INFO" | jq -r '.data.user.email // "未绑定"')

        echo ""
        echo "✅ 成功"
        echo "   姓名:       $MATCH_NAME"
        echo "   Bot OpenID: $BOT_OPENID"
        echo "   邮箱:       $EMAIL"
        echo ""
        echo "   配置: \"$MATCH_NAME\": \"$BOT_OPENID\""
    else
        ERROR_MSG=$(echo "$BOT_INFO" | jq -r '.msg // .error.message // "未知错误"' 2>/dev/null)
        echo ""
        echo "❌ bot 无法查询该用户"
        echo "   可能原因: 该用户不在 bot-service 的可见范围内"
        echo "   解决方案: 将该用户加入与机器人相同的群组后重试"
    fi

    echo ""
done

echo "======================================"
echo "✅ 查询完成"
echo "======================================"
