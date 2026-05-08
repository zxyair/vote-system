// 使用内置的 fetch API

// 测试用户1
const userId1 = 'test_user_1';

// 测试用户2
const userId2 = 'test_user_2';

// API基础URL
const baseUrl = 'http://localhost:8080';

// 测试创建投票
async function createPoll() {
    console.log('创建投票...');
    const response = await fetch(`${baseUrl}/polls/createPoll`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-User-Id': userId1,
            'Idempotency-Key': 'create_poll_1'
        },
        body: JSON.stringify({
            question: '你最喜欢的编程语言是什么？',
            options: ['Go', 'JavaScript', 'Python', 'Java'],
            expires_at: new Date(Date.now() + 10 * 60 * 1000).toISOString(),
            is_public: true
        })
    });

    const data = await response.json();
    console.log('创建投票结果:', data);
    return data.id;
}

// 测试投票
async function vote(pollId, userId, option) {
    console.log(`${userId} 投票给 ${option}...`);
    const response = await fetch(`${baseUrl}/votes/${encodeURIComponent(pollId)}/vote`, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'X-User-Id': userId,
            'Idempotency-Key': `${userId}_vote_1`
        },
        body: JSON.stringify({
            option: option
        })
    });

    const data = await response.json();
    console.log(`${userId} 投票结果:`, data);
    return data;
}

// 测试获取投票详情
async function getPoll(pollId) {
    console.log('获取投票详情...');
    const response = await fetch(`${baseUrl}/polls/${encodeURIComponent(pollId)}`, {
        headers: {
            'X-User-Id': userId1
        }
    });

    const data = await response.json();
    console.log('投票详情:', data);
    return data;
}

// 主测试函数
async function main() {
    try {
        const pollId = await createPoll();

        // 等待一下让服务器处理创建请求
        await new Promise(resolve => setTimeout(resolve, 1000));

        // 用户1投票给Go
        await vote(pollId, userId1, 'Go');

        // 等待一下
        await new Promise(resolve => setTimeout(resolve, 2000));

        // 用户2投票给JavaScript
        await vote(pollId, userId2, 'JavaScript');

        // 等待一下
        await new Promise(resolve => setTimeout(resolve, 2000));

        // 用户1撤销投票
        const undoResponse = await fetch(`${baseUrl}/votes/${encodeURIComponent(pollId)}/vote`, {
            method: 'DELETE',
            headers: {
                'X-User-Id': userId1,
                'Idempotency-Key': `${userId1}_undo_1`
            }
        });

        const undoData = await undoResponse.json();
        console.log('撤销投票结果:', undoData);

        // 获取最终结果
        await new Promise(resolve => setTimeout(resolve, 2000));
        await getPoll(pollId);

    } catch (error) {
        console.error('测试失败:', error);
    }
}

main();