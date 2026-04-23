#!/bin/bash

echo "=== 投票系统前端测试报告 ==="
echo ""

# 1. 检查服务状态
echo "1. 服务状态检查"
if curl -s http://localhost:8080 > /dev/null 2>&1; then
    echo "✅ HTTP服务器 (8080端口) 正常运行"
else
    echo "❌ HTTP服务器无响应"
fi

if netstat -an | grep -q ":9090.*LISTENING"; then
    echo "✅ gRPC服务器 (9090端口) 正常运行"
else
    echo "❌ gRPC服务器无响应"
fi

echo ""

# 2. 测试前端文件访问
echo "2. 前端资源访问"
if curl -s http://localhost:8080/static/style.css | grep -q "container"; then
    echo "✅ CSS样式文件正常"
else
    echo "❌ CSS样式文件无法访问"
fi

if curl -s http://localhost:8080/static/app.js | grep -q "escapeHtml"; then
    echo "✅ JavaScript文件正常"
else
    echo "❌ JavaScript文件无法访问"
fi

echo ""

# 3. 测试API功能
echo "3. API功能测试"

# 创建投票
POLL_DATA='{
    "question": "前端实时刷新测试",
    "options": ["实时刷新", "普通刷新", "自动刷新", "手动刷新"],
    "expires_at": "'$(date -u +'%Y-%m-%dT%H:%M:%SZ' -d '+120 minutes')'",
    "is_public": true
}'

CREATE_RESP=$(curl -s -X POST http://localhost:8080/polls/createPoll \
    -H "Content-Type: application/json" \
    -H "X-User-Id: test_user_frontend" \
    -d "$POLL_DATA")

if echo "$CREATE_RESP" | grep -q '"id":"'; then
    POLL_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "✅ 创建投票成功 (ID: $POLL_ID)"

    # 获取投票详情
    DETAIL_RESP=$(curl -s http://localhost:8080/polls/$POLL_ID \
        -H "X-User-Id: test_user_frontend")

    if echo "$DETAIL_RESP" | grep -q "前端实时刷新测试"; then
        echo "✅ 获取投票详情成功"

        # 测试投票
        VOTE_RESP=$(curl -s -X POST http://localhost:8080/votes/$POLL_ID/vote \
            -H "Content-Type: application/json" \
            -H "X-User-Id: test_user_2" \
            -d '{"option": "实时刷新"}')

        if echo "$VOTE_RESP" | grep -q "实时刷新"; then
            echo "✅ 投票功能正常"

            # 测试撤销投票
            UNDO_RESP=$(curl -s -X DELETE http://localhost:8080/votes/$POLL_ID/vote \
                -H "Content-Type: application/json" \
                -H "X-User-Id: test_user_2")

            if ! echo "$UNDO_RESP" | grep -q "error"; then
                echo "✅ 撤销投票功能正常"
            else
                echo "❌ 撤销投票失败"
            fi
        else
            echo "❌ 投票功能异常"
        fi
    else
        echo "❌ 获取投票详情失败"
    fi
else
    echo "❌ 创建投票失败"
    echo "   响应: $CREATE_RESP"
fi

echo ""

# 4. SSE实时更新测试
echo "4. SSE实时更新功能"
echo "   - 启动SSE监听: curl -N -H 'Accept: text/event-stream' 'http://localhost:8080/events/polls/$POLL_ID?user_id=test_user_frontend'"
echo "   - 在另一个终端进行投票操作，观察是否有实时更新"
echo ""

# 5. 数据一致性检查
echo "5. 数据一致性验证"

# 获取各类数据
curl -s http://localhost:8080/polls/public?include_closed=false \
    -H "X-User-Id: test_user_frontend" > public.json

curl -s http://localhost:8080/polls/my_created/stats?include_closed=true \
    -H "X-User-Id: test_user_frontend" > created.json

curl -s http://localhost:8080/users/me/votes \
    -H "X-User-Id: test_user_frontend" > votes.json

# 统计数据
PUBLIC_COUNT=$(grep -c '"id"' public.json 2>/dev/null || echo "0")
CREATED_COUNT=$(grep -c '"id"' created.json 2>/dev/null || echo "0")
VOTES_COUNT=$(grep -c '"pollId"' votes.json 2>/dev/null || echo "0")

echo "   - 公开投票数量: $PUBLIC_COUNT"
echo "   - 我创建的投票数量: $CREATED_COUNT"
echo "   - 我的投票记录数: $VOTES_COUNT"

# 检查数据一致性
if [ $PUBLIC_COUNT -gt 0 ] && [ $CREATED_COUNT -gt 0 ]; then
    echo "⚠️  注意：系统将公开投票和用户创建的投票分开显示"
    echo "   这是正常设计，避免重复显示"
fi

echo ""

# 6. 前端页面检查
echo "6. 前端页面功能预览"
echo "   - 登录页面: 支持用户ID设置，使用sessionStorage存储"
echo "   - 首页: 创建投票、参与投票、查看结果三个主要功能"
echo "   - 投票详情页: 显示投票信息、选项、实时票数、投票按钮"
echo "   - 结果页面: 分为三个板块 - 我创建的、我参与的、公共投票"
echo "   - SSE状态: 显示连接状态（connecting/connected/reconnecting）"
echo ""

# 7. 潜在问题检查
echo "7. 潜在问题检查"

# 检查内存存储限制
if curl -s http://localhost:8080/polls/search | grep -q "limit"; then
    echo "⚠️  搜索功能可能存在内存限制问题"
fi

# 检查SSE重连机制
if curl -s http://localhost:8080/events/polls/test?user_id=test | grep -q "retry"; then
    echo "✅ SSE支持自动重连"
else
    echo "⚠️  SSE重连机制需要验证"
fi

echo ""

# 清理
rm -f public.json created.json votes.json

echo "=== 测试完成 ==="
echo ""
echo "🔍 手动验证建议:"
echo "1. 打开浏览器访问 http://localhost:8080"
echo "2. 使用不同的用户ID登录（如 user_1, user_2）"
echo "3. 创建投票并查看实时更新"
echo "4. 在多个标签页中打开同一投票，验证实时同步"
echo "5. 检查各个页面的数据是否一致"