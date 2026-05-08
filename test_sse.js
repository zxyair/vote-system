// 测试 SSE 实时更新功能
const http = require('http');

// 测试服务器
const server = http.createServer((req, res) => {
  // 静态文件服务
  if (req.method === 'GET' && req.url === '/') {
    res.writeHead(200, {
      'Content-Type': 'text/html; charset=utf-8'
    });
    res.end(`
<!DOCTYPE html>
<html>
<head>
    <title>SSE 测试</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; }
        .container { max-width: 800px; margin: 0 auto; }
        .log { background: #f5f5f5; padding: 10px; border-radius: 5px; height: 400px; overflow-y: auto; margin: 20px 0; }
        .log-entry { margin: 5px 0; padding: 5px; border-radius: 3px; }
        .log-open { background: #e8f5e8; }
        .log-message { background: #e8f0ff; }
        .log-error { background: #ffe8e8; }
        button { padding: 10px 20px; margin: 5px; cursor: pointer; }
        .status { font-weight: bold; margin: 10px 0; }
    </style>
</head>
<body>
    <div class="container">
        <h1>SSE 实时更新测试</h1>

        <div>
            <label>用户ID: <input type="text" id="userId" value="test_user_1"></label>
            <label>投票ID: <input type="text" id="pollId" value="7a3bed10-27f9-4ab1-962e-68324bf09a55"></label>
        </div>

        <div>
            <button onclick="testPollSSE()">测试投票页面 SSE</button>
            <button onclick="testResultsSSE()">测试结果页面 SSE</button>
            <button onclick="testVote()">测试投票</button>
        </div>

        <div class="status" id="status">准备就绪</div>

        <div class="log" id="log"></div>
    </div>

    <script>
        function log(message, type = 'message') {
            const logDiv = document.getElementById('log');
            const entry = document.createElement('div');
            entry.className = \`log-entry log-\${type}\`;
            entry.textContent = \`[\${new Date().toLocaleTimeString()}] \${message}\`;
            logDiv.appendChild(entry);
            logDiv.scrollTop = logDiv.scrollHeight;
        }

        function updateStatus(text) {
            document.getElementById('status').textContent = text;
        }

        function testPollSSE() {
            const userId = document.getElementById('userId').value;
            const pollId = document.getElementById('pollId').value;

            updateStatus('连接投票页面 SSE...');
            log('开始连接投票页面 SSE...');

            const es = new EventSource(\`/events/polls/\${pollId}?user_id=\${encodeURIComponent(userId)}\`);

            es.onopen = function() {
                updateStatus('投票页面 SSE 已连接');
                log('SSE 连接已建立', 'open');
            };

            es.onerror = function(err) {
                updateStatus('投票页面 SSE 连接错误');
                log('SSE 连接错误: ' + err, 'error');
                es.close();
            };

            es.addEventListener('poll_invalidate', function(evt) {
                log('收到 poll_invalidate 事件: ' + evt.data, 'message');
                try {
                    const data = JSON.parse(evt.data);
                    log('解析的数据: ' + JSON.stringify(data), 'message');
                } catch (e) {
                    log('JSON 解析错误: ' + e, 'error');
                }
            });

            es.addEventListener('error', function(evt) {
                log('SSE 错误事件', 'error');
            });
        }

        function testResultsSSE() {
            const userId = document.getElementById('userId').value;

            updateStatus('连接结果页面 SSE...');
            log('开始连接结果页面 SSE...');

            const es = new EventSource(\`/events/results?user_id=\${encodeURIComponent(userId)}\`);

            es.onopen = function() {
                updateStatus('结果页面 SSE 已连接');
                log('结果页面 SSE 连接已建立', 'open');
            };

            es.onerror = function(err) {
                updateStatus('结果页面 SSE 连接错误');
                log('结果页面 SSE 连接错误: ' + err, 'error');
                es.close();
            };

            es.addEventListener('invalidate', function(evt) {
                log('收到 invalidate 事件: ' + evt.data, 'message');
                try {
                    const data = JSON.parse(evt.data);
                    log('解析的数据: ' + JSON.stringify(data), 'message');
                } catch (e) {
                    log('JSON 解析错误: ' + e, 'error');
                }
            });
        }

        function testVote() {
            const userId = document.getElementById('userId').value;
            const pollId = document.getElementById('pollId').value;
            const option = 'Python';

            updateStatus('测试投票...');
            log('开始投票测试...');

            fetch('/votes/' + encodeURIComponent(pollId) + '/vote', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-User-Id': userId,
                    'Idempotency-Key': 'test_vote_' + Date.now()
                },
                body: JSON.stringify({
                    option: option
                })
            })
            .then(response => response.json())
            .then(data => {
                log('投票成功: ' + JSON.stringify(data), 'open');
                updateStatus('投票成功');
            })
            .catch(error => {
                log('投票失败: ' + error, 'error');
                updateStatus('投票失败');
            });
        }
    </script>
</body>
</html>
    `);
    } else {
      // 代理请求到本地服务器
      const options = {
        hostname: 'localhost',
        port: 8080,
        path: req.url,
        method: req.method,
        headers: req.headers
      };

      const proxyReq = http.request(options, (proxyRes) => {
        res.writeHead(proxyRes.statusCode, proxyRes.headers);
        proxyRes.pipe(res, { end: true });
      });

      req.pipe(proxyReq, { end: true });
    }
});

server.listen(3000, () => {
  console.log('测试服务器运行在 http://localhost:3000');
  console.log('请在浏览器中打开 http://localhost:3000 进行测试');
});