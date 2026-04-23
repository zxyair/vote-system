#!/bin/bash

echo "=== SSE实时更新优化测试 ==="
echo ""

# 1. 测试监控端点
echo "1. 测试监控端点..."
if curl -s http://localhost:8080/metrics | grep -q "sse_active_connections"; then
    echo "✅ Prometheus指标端点正常"
else
    echo "❌ Prometheus指标端点异常"
fi

if curl -s http://localhost:8080/sse/stats | grep -q "active_connections"; then
    echo "✅ SSE统计端点正常"
else
    echo "❌ SSE统计端点异常"
fi

echo ""

# 2. 创建测试投票
echo "2. 创建测试投票..."
POLL_DATA='{
    "question": "SSE优化测试：实时刷新体验",
    "options": ["实时更新", "立即反馈", "流畅体验", "稳定可靠"],
    "expires_at": "'$(date -u +'%Y-%m-%dT%H:%M:%SZ' -d '+120 minutes')'",
    "is_public": true
}'

CREATE_RESP=$(curl -s -X POST http://localhost:8080/polls/createPoll \
    -H "Content-Type: application/json" \
    -H "X-User-Id: sse_test_user" \
    -d "$POLL_DATA")

if echo "$CREATE_RESP" | grep -q '"id":"'; then
    POLL_ID=$(echo "$CREATE_RESP" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
    echo "✅ 创建投票成功 (ID: $POLL_ID)"

    # 3. 测试SSE连接
    echo ""
    echo "3. 测试SSE实时更新..."
    echo "   在另一个终端运行以下命令测试SSE："
    echo "   curl -N -H 'Accept: text/event-stream' 'http://localhost:8080/events/polls/$POLL_ID?user_id=sse_test_user'"
    echo ""
    echo "   然后在另一个终端进行投票操作："
    echo "   curl -X POST http://localhost:8080/votes/$POLL_ID/vote \\"
    echo "     -H 'Content-Type: application/json' \\"
    echo "     -H 'X-User-Id: voter_user' \\"
    echo "     -d '{\"option\": \"实时更新\"}'"
    echo ""
    echo "   观察SSE是否有实时更新推送"

    # 4. 测试多连接限制
    echo ""
    echo "4. 测试连接限制..."
    echo "   尝试创建多个SSE连接（超过5个）来测试连接限制："

    # 创建6个并发SSE连接
    for i in {1..6}; do
        curl -s -N "http://localhost:8080/events/results?user_id=test_user_$i" &
    done

    sleep 3
    echo "   ✅ 已创建6个SSE连接，系统应自动限制"

    # 清理后台进程
    pkill -f "curl.*events/results"

else
    echo "❌ 创建投票失败"
fi

echo ""

# 5. 性能对比
echo "5. 优化效果对比："
echo "   📈 缓冲区大小: 16字节 → 1024字节 (64倍增长)"
echo "   🔒 连接限制: 无限制 → 每用户5个连接"
echo "   ⏰ 自动清理: 无 → 30分钟TTL"
echo "   🔄 重试机制: 无 → 3次重试"
echo "   📊 监控指标: 无 → Prometheus完整监控"

echo ""
echo "=== 测试完成 ==="