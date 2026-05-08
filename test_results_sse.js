// 简单的测试脚本，验证results页面的SSE功能
const http = require('http');

// 测试数据
const testUserID = 'test_user_' + Date.now();
const testPollData = {
    question: '测试实时更新问题',
    options: ['选项1', '选项2', '选项3'],
    expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    is_public: true
};

// 1. 创建投票
function createPoll() {
    return new Promise((resolve, reject) => {
        const postData = JSON.stringify(testPollData);

        const options = {
            hostname: 'localhost',
            port: 8080,
            path: '/polls/createPoll',
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(postData),
                'X-User-Id': testUserID
            }
        };

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                const response = JSON.parse(data);
                console.log('✅ 创建投票成功:', response.id);
                resolve(response.id);
            });
        });

        req.on('error', reject);
        req.write(postData);
        req.end();
    });
}

// 2. 连接到results SSE
function connectResultsSSE() {
    return new Promise((resolve, reject) => {
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: `/events/results?user_id=${encodeURIComponent(testUserID)}`,
            method: 'GET',
            headers: {
                'Accept': 'text/event-stream',
                'Cache-Control': 'no-cache',
                'X-User-Id': testUserID
            }
        };

        const req = http.request(options, (res) => {
            console.log('\n🔗 连接到results SSE...');
            console.log('状态码:', res.statusCode);

            let buffer = '';
            let eventCount = 0;

            res.on('data', chunk => {
                buffer += chunk.toString();

                // 处理SSE消息
                const lines = buffer.split('\n');
                buffer = lines.pop() || ''; // 保留最后一个不完整的行

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        try {
                            const data = JSON.parse(line.slice(6));
                            eventCount++;
                            console.log(`📥 收到实时更新 #${eventCount}:`, data);

                            if (eventCount >= 3) {
                                console.log('\n✅ SSE实时更新测试成功！');
                                resolve();
                            }
                        } catch (e) {
                            console.log('解析SSE数据失败:', line);
                        }
                    }
                }
            });

            res.on('end', () => {
                if (eventCount < 3) {
                    console.log('\n❌ SSE实时更新测试失败：未收到足够的更新事件');
                    reject(new Error('SSE timeout'));
                }
            });

            // 10秒超时
            setTimeout(() => {
                if (eventCount < 3) {
                    console.log('\n❌ SSE实时更新测试失败：超时');
                    reject(new Error('SSE timeout'));
                }
            }, 10000);
        });

        req.on('error', reject);
        req.end();
    });
}

// 3. 投票以触发SSE更新
function vote(pollID) {
    return new Promise((resolve, reject) => {
        const voteData = JSON.stringify({ option: '选项1' });

        const options = {
            hostname: 'localhost',
            port: 8080,
            path: `/votes/${encodeURIComponent(pollID)}/vote`,
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'Content-Length': Buffer.byteLength(voteData),
                'X-User-Id': testUserID
            }
        };

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                console.log('✅ 投票成功');
                resolve();
            });
        });

        req.on('error', reject);
        req.write(voteData);
        req.end();
    });
}

// 主测试函数
async function main() {
    try {
        console.log('🧪 开始测试results页面SSE实时更新...\n');

        // 步骤1：创建投票
        const pollID = await createPoll();

        // 步骤2：连接SSE
        await connectResultsSSE();

        // 步骤3：投票以触发更新
        await vote(pollID);

        console.log('\n🎉 所有测试通过！');
    } catch (error) {
        console.error('\n❌ 测试失败:', error.message);
        process.exit(1);
    }
}

main();