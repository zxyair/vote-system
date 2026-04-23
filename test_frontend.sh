#!/bin/bash

# 测试前端功能和实时刷新

echo "=== 投票系统前端测试 ==="
echo "1. 测试首页访问..."
curl -s http://localhost:8080 > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ HTTP服务器正常响应"
else
    echo "❌ HTTP服务器无响应"
    exit 1
fi

echo ""
echo "2. 测试API端点..."

# 测试创建投票
echo "   - 测试创建投票..."
CREATE_RESPONSE=$(curl -s -X POST http://localhost:8080/polls/createPoll \
    -H "Content-Type: application/json" \
    -H "X-User-Id: test_user_1" \
    -d '{
        "question": "测试问题：你最喜欢的编程语言是什么？",
        "options": ["Go", "Rust", "Python", "JavaScript"],
        "expires_at": "'$(date -u +'%Y-%m-%dT%H:%M:%SZ' -d '+60 minutes')'",
        "is_public": true
    }')

POLL_ID=$(echo "$CREATE_RESPONSE" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)

if [ ! -z "$POLL_ID" ]; then
    echo "   ✅ 成功创建投票，ID: $POLL_ID"
else
    echo "   ❌ 创建投票失败，响应: $CREATE_RESPONSE"
    exit 1
fi

# 测试获取投票详情
echo "   - 测试获取投票详情..."
curl -s http://localhost:8080/polls/$POLL_ID \
    -H "X-User-Id: test_user_1" > poll_detail.json
if [ $? -eq 0 ]; then
    echo "   ✅ 获取投票详情成功"
    QUESTION=$(grep -o '"question":"[^"]*"' poll_detail.json | cut -d'"' -f4)
    OPT_COUNT=$(grep -c '"option":"' poll_detail.json)
    echo "   - 投票问题: $QUESTION"
    echo "   - 选项数量: $OPT_COUNT"
else
    echo "   ❌ 获取投票详情失败"
fi

# 测试投票
echo "   - 测试投票..."
VOTE_RESPONSE=$(curl -s -X POST http://localhost:8080/votes/$POLL_ID/vote \
    -H "Content-Type: application/json" \
    -H "X-User-Id: test_user_2" \
    -d '{"option": "Go"}')

if echo "$VOTE_RESPONSE" | grep -q "error"; then
    echo "   ❌ 投票失败: $VOTE_RESPONSE"
else
    echo "   ✅ 投票成功"
fi

# 测试撤销投票
echo "   - 测试撤销投票..."
UNDO_RESPONSE=$(curl -s -X DELETE http://localhost:8080/votes/$POLL_ID/vote \
    -H "Content-Type: application/json" \
    -H "X-User-Id: test_user_2")

if echo "$UNDO_RESPONSE" | grep -q "error"; then
    echo "   ❌ 撤销投票失败: $UNDO_RESPONSE"
else
    echo "   ✅ 撤销投票成功"
fi

echo ""
echo "3. 测试SSE实时更新..."
echo "   - 打开一个新的终端窗口运行以下命令测试SSE："
echo "     curl -N -H 'Accept: text/event-stream' 'http://localhost:8080/events/polls/$POLL_ID?user_id=test_user_1'"
echo "   - 然后在另一个终端窗口进行投票操作，观察是否有实时更新"

echo ""
echo "4. 测试多个页面数据一致性..."
echo "   - 获取公开投票列表..."
curl -s http://localhost:8080/polls/public?include_closed=false \
    -H "X-User-Id: test_user_1" > public_polls.json

echo "   - 获取用户创建的投票..."
curl -s http://localhost:8080/polls/my_created/stats?include_closed=true \
    -H "X-User-Id: test_user_1" > my_created_polls.json

echo "   - 获取用户投票记录..."
curl -s http://localhost:8080/users/me/votes \
    -H "X-User-Id: test_user_1" > my_votes.json

echo "   ✅ 数据已获取，请检查各页面显示的数据是否一致"

# 验证数据一致性
echo ""
echo "5. 验证数据一致性..."
if [ -f public_polls.json ] && [ -f my_created_polls.json ]; then
    PUBLIC_COUNT=$(grep -c '"id":"' public_polls.json)
    MY_CREATED_COUNT=$(grep -c '"id":"' my_created_polls.json)
    echo "   - 公开投票数量: $PUBLIC_COUNT"
    echo "   - 我创建的投票数量: $MY_CREATED_COUNT"

    if [ $PUBLIC_COUNT -gt 0 ] && [ $MY_CREATED_COUNT -gt 0 ]; then
        # 检查是否有重复的投票ID
        PUBLIC_IDS=$(grep -o '"id":"[^"]*"' public_polls.json | cut -d'"' -f4)
        MY_IDS=$(grep -o '"id":"[^"]*'" my_created_polls.json | cut -d'"' -f4)

        for pid in $MY_IDS; do
            if echo "$PUBLIC_IDS" | grep -q "$pid"; then
                echo "   ⚠️  警告：投票ID $pid 同时出现在公开列表和我的创建列表中"
            fi
        done
    fi
fi

# 清理
rm -f poll_detail.json public_polls.json my_created_polls.json my_votes.json

echo ""
echo "=== 测试完成 ==="
echo "📝 注意事项："
echo "1. 打开浏览器访问 http://localhost:8080 进行界面测试"
echo "2. 使用不同用户ID测试多用户场景"
echo "3. 检查SSE连接状态显示"
echo "4. 验证各页面数据一致性"