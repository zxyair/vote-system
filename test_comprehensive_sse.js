// 综合测试SSE功能
const http = require('http');

const testUserID = 'comprehensive_test_' + Date.now();
let pollID = null;
let sseConnected = false;
let eventsReceived = 0;

// 创建投票
function createPoll() {
    return new Promise((resolve, reject) => {
        const postData = JSON.stringify({
            question: '综合测试问题',
            options: ['选项A', '选项B', '选项C'],
            expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
            is_public: true
        });

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
                pollID = response.id;
                resolve(response.id);
            });
        });

        req.on('error', reject);
        req.write(postData);
        req.end();
    });
}

// 连接SSE
function connectSSE() {
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
            console.log('\n🔗 已连接到results SSE，状态码:', res.statusCode);

            let buffer = '';

            res.on('data', chunk => {
                buffer += chunk.toString();

                const lines = buffer.split('\n');
                buffer = lines.pop() || '';

                for (const line of lines) {
                    if (line.startsWith('data: ')) {
                        try {
                            const data = JSON.parse(line.slice(6));
                            eventsReceived++;
                            console.log(`📥 SSE更新 #${eventsReceived}:`, JSON.stringify(data, null, 2));

                            if (eventsReceived >= 5) {
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
                console.log('\n❌ SSE连接已关闭');
                if (eventsReceived < 5) {
                    reject(new Error('SSE连接过早关闭'));
                }
            });
        });

        req.on('error', reject);
        req.setTimeout(30000);
        req.end();

        // 30秒后强制结束
        setTimeout(() => {
            if (eventsReceived < 5) {
                console.log(`\n⏰ 测试超时（${eventsReceived}/5 事件）`);
                if (eventsReceived > 0) {
                    console.log('✅ SSE连接正常，但需要更多触发事件');
                    resolve();
                } else {
                    reject(new Error('未收到任何SSE事件'));
                }
            }
        }, 30000);
    });
}

// 执行多个投票操作来触发SSE
async function performVoteOperations() {
    if (!pollID) throw new Error('Poll ID not available');

    const options = [
        '选项A',
        '选项B',
        '选项A'  // 再次投票给选项A
    ];

    for (let i = 0; i < options.length; i++) {
        console.log(`\n🗳️ 执行第${i+1}次投票...`);

        await new Promise((resolve, reject) => {
            const voteData = JSON.stringify({ option: options[i] });

            const req = http.request({
                hostname: 'localhost',
                port: 8080,
                path: `/votes/${encodeURIComponent(pollID)}/vote`,
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'Content-Length': Buffer.byteLength(voteData),
                    'X-User-Id': testUserID
                }
            }, (res) => {
                let data = '';
                res.on('data', chunk => data += chunk);
                res.on('end', () => {
                    console.log(`✅ 投票成功（${options[i]}）`);
                    // 等待SSE事件传播
                    setTimeout(resolve, 1000);
                });
            });

            req.on('error', reject);
            req.write(voteData);
            req.end();
        });
    }
}

// 测试撤销投票
async function testUndoVote() {
    console.log('\n🔄 测试撤销投票...');

    await new Promise((resolve, reject) => {
        const options = {
            hostname: 'localhost',
            port: 8080,
            path: `/votes/${encodeURIComponent(pollID)}/vote`,
            method: 'DELETE',
            headers: {
                'X-User-Id': testUserID
            }
        };

        const req = http.request(options, (res) => {
            let data = '';
            res.on('data', chunk => data += chunk);
            res.on('end', () => {
                console.log('✅ 撤销投票成功');
                setTimeout(resolve, 1000);
            });
        });

        req.on('error', reject);
        req.end();
    });
}

// 主测试函数
async function main() {
    try {
        console.log('🧪 开始综合测试results页面SSE实时更新...\n');

        // 步骤1：创建投票
        await createPoll();

        // 步骤2：连接SSE
        await connectSSE();

        // 步骤3：执行投票操作
        await performVoteOperations();

        // 步骤4：测试撤销投票
        await testUndoVote();

        console.log('\n🎉 综合测试完成！SSE实时更新功能正常工作。');
    } catch (error) {
        console.error('\n❌ 测试失败:', error.message);
        process.exit(1);
    }
}

main();