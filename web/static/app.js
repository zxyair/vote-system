// TODO
// 1、投票页面显示持续时间，发起人
// 2、实时显示票数——轮询/推送/websocket/单向server->client/回调
// 3、投票详情页是否需要，直接展示在页面即可

function escapeHtml(s) {
  return String(s).replace(/[&<>"']/g, (c) => ({
    "&": "&amp;",
    "<": "&lt;",
    ">": "&gt;",
    '"': "&quot;",
    "'": "&#039;",
  }[c]));
}

function pick(obj, camel, snake) {
  if (!obj) return undefined;
  if (obj[camel] !== undefined) return obj[camel];
  if (obj[snake] !== undefined) return obj[snake];
  return undefined;
}

function normalizePoll(p) {
  if (!p) return p;
  return {
    ...p,
    id: pick(p, "id", "id"),
    question: pick(p, "question", "question"),
    options: pick(p, "options", "options") || [],
    votes: pick(p, "votes", "votes") || {},
    createdBy: pick(p, "createdBy", "created_by"),
    updatedBy: pick(p, "updatedBy", "updated_by"),
    createdAt: pick(p, "createdAt", "created_at"),
    expiresAt: pick(p, "expiresAt", "expires_at"),
    isClosed: pick(p, "isClosed", "is_closed"),
    isPublic: pick(p, "isPublic", "is_public"),
  };
}
const $ = (id) => document.getElementById(id);
const app = () => $("app");

function getUserId() {
// Use per-tab storage to avoid cross-tab identity overwrite.
return sessionStorage.getItem("user_id") || "";
}
function setUserId(v) {
sessionStorage.setItem("user_id", v);
}

function setUserCookie(v) {
  // allow httpserver to authenticate SSE via cookie
  const encoded = encodeURIComponent(v || "");
  document.cookie = `user_id=${encoded}; Path=/; Max-Age=2592000; SameSite=Lax`;
}

function nav(to) {
window.location.hash = to;
}

function parseRoute() {
const h = window.location.hash || "";
const raw = h.startsWith("#") ? h.slice(1) : h;
const path = raw || (getUserId() ? "/home" : "/login");
const [p, qs] = path.split("?");
const parts = p.split("/").filter(Boolean);
const query = new URLSearchParams(qs || "");
return { path: p, parts, query };
}

// #region agent log
fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
  method: "POST",
  headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
  body: JSON.stringify({
    sessionId: "cd7378",
    runId: "pre-fix2",
    hypothesisId: "H5",
    location: "web/static/app.js:45",
    message: "app_js_loaded",
    data: { buildTag: "2026-03-17-pre-fix2", hasParseProtoTimestamp: typeof parseProtoTimestamp === "function" },
    timestamp: Date.now(),
  }),
}).catch(() => {});
// #endregion

async function api(path, { method = "GET", body } = {}) {
const headers = { "X-User-Id": getUserId() };
if (body !== undefined) headers["Content-Type"] = "application/json";
const resp = await fetch(path, {
  method,
  headers,
  body: body === undefined ? undefined : JSON.stringify(body),
});
const text = await resp.text();
let data;
try {
  data = text ? JSON.parse(text) : null;
} catch {
  data = text;
}
if (!resp.ok) {
  // #region agent log
  fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Debug-Session-Id": "cd7378",
    },
    body: JSON.stringify({
      sessionId: "cd7378",
      runId: "pre-fix2",
      hypothesisId: "H2",
      location: "web/static/app.js:60",
      message: "api_error",
      data: {
        path,
        method,
        status: resp.status,
        statusText: resp.statusText,
        responseText: text,
      },
      timestamp: Date.now(),
    }),
  }).catch(() => {});
  // #endregion

  const msg = data?.error || resp.statusText;
  throw new Error(`${resp.status} ${msg}`);
}
return data;
}

function showToast(message, type = "error") {
  let box = document.getElementById("toast-root");
  if (!box) {
    box = document.createElement("div");
    box.id = "toast-root";
    box.className = "toast-root";
    document.body.appendChild(box);
  }

  const el = document.createElement("div");
  el.className = `toast toast-${type}`;
  el.textContent = message;
  box.appendChild(el);
  setTimeout(() => {
    el.classList.add("leaving");
    setTimeout(() => el.remove(), 300);
  }, 2500);
}

function parseProtoTimestamp(ts) {
  if (!ts) return null;
  if (typeof ts === "string") {
    const d = new Date(ts);
    return Number.isNaN(d.getTime()) ? null : d;
  }
  if (typeof ts === "object" && ts.seconds !== undefined) {
    const ms = Number(ts.seconds) * 1000 + Math.floor(Number(ts.nanos || 0) / 1e6);
    const d = new Date(ms);
    return Number.isNaN(d.getTime()) ? null : d;
  }
  return null;
}

function pad2(n) {
  return String(n).padStart(2, "0");
}

function formatExact(d) {
  if (!d) return "-";
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`;
}

function formatDuration(ms) {
  const abs = Math.abs(ms);
  const s = Math.floor(abs / 1000);
  const m = Math.floor(s / 60);
  const h = Math.floor(m / 60);
  const d = Math.floor(h / 24);
  const mm = m % 60;
  const hh = h % 24;
  if (d > 0) return `${d}天${hh}小时`;
  if (h > 0) return `${h}小时${mm}分钟`;
  return `${m}分钟`;
}

function formatRelativeToNow(end) {
  if (!end) return "";
  const delta = end.getTime() - Date.now();
  if (delta >= 0) return `剩余 ${formatDuration(delta)}`;
  return `已结束 ${formatDuration(delta)}`;
}

function requireLogin(nextHash) {
if (getUserId()) return false;
nav(`/login?next=${encodeURIComponent(nextHash)}`);
return true;
}

function layout(title, bodyHtml) {
const uid = getUserId();
const homeBtn = uid ? `<button class="btnLink" data-nav="/home">首页</button>` : "";
const resultsBtn = uid ? `<button class="btnLink" data-nav="/results">查看结果</button>` : "";
const userBadge = uid ? `<span class="badge">当前用户：<code>${escapeHtml(uid)}</code></span>` : "";
return `
  <section class="card">
    <div class="topbar">
      <div>
        <h2>${escapeHtml(title)}</h2>
        ${userBadge}
      </div>
      <div class="topActions">
        ${homeBtn}
        ${resultsBtn}
        <button class="btnLink" data-nav="/login">切换用户</button>
      </div>
    </div>
  </section>
  ${bodyHtml}
`;
}

function bindNavButtons(root) {
root.querySelectorAll("[data-nav]").forEach((b) => {
  b.addEventListener("click", () => nav(b.getAttribute("data-nav")));
});
}

async function render() {
const r = parseRoute();

  // Close results SSE when leaving results page.
  if (r.parts[0] !== "results" && window.__resultsSSE) {
    window.__resultsSSE.close();
    window.__resultsSSE = null;
  }

if (r.parts[0] === "poll" && r.parts[1]) {
  const next = `#/poll/${encodeURIComponent(r.parts[1])}`;
  if (requireLogin(next)) return;
}
if (["home", "create", "join", "results"].includes(r.parts[0] || "")) {
  if (requireLogin(`#/${r.parts[0]}`)) return;
}

try {
  if (r.parts[0] === "login" || r.parts.length === 0) {
    const next = r.query.get("next") || "#/home";
    app().innerHTML = layout(
      "登录",
      `
        <section class="card">
          <div class="row">
            <label>
              用户ID（请求头 <code>X-User-Id</code>）
              <input id="userId" placeholder="例如：user_123" value="${escapeHtml(getUserId())}" />
            </label>
            <button id="saveUser">进入</button>
          </div>
          <p class="muted">说明：后端接口要求必须带 <code>X-User-Id</code>。</p>
        </section>
      `,
    );
    bindNavButtons(app());
    $("saveUser").addEventListener("click", () => {
      const uid = $("userId").value.trim();
      setUserId(uid);
      setUserCookie(uid);
      console.log("saved user id", uid);
      window.location.href = next;
    });
    return;
  }

  if (r.parts[0] === "home") {
    app().innerHTML = layout(
      "首页",
      `
        <section class="card">
          <div class="row">
            <button data-nav="/create">创建投票</button>
            <button data-nav="/join">参与投票</button>
            <button data-nav="/results">查看投票结果</button>
          </div>
        </section>
      `,
    );
    bindNavButtons(app());
    return;
  }

  if (r.parts[0] === "create") {
    app().innerHTML = layout(
      "创建投票",
      `
        <section class="card">
          <div class="grid2">
            <label>
              投票主题
              <input id="q" placeholder="例如：你最喜欢的技术话题？" />
            </label>
            <label>
              持续时间（分钟）
              <input id="mins" type="number" min="1" value="60" />
            </label>
          </div>
          <label style="margin-top:12px;">
            选项（每行一个，至少 3 个）
            <textarea id="opts" rows="5" placeholder="Go\nKubernetes\nRedis"></textarea>
          </label>
          <div class="row" style="margin-top:12px;">
            <label class="check">
              <input id="isPublic" type="checkbox" checked />
              公开投票（会出现在公开列表里）
            </label>
            <button id="publish">发布</button>
          </div>
          <div id="created" class="muted" style="margin-top:12px;"></div>
        </section>
      `,
    );
    bindNavButtons(app());

    $("publish").addEventListener("click", async () => {
      const q = $("q").value.trim();
      const mins = Number($("mins").value || "0");
      const rawLines = $("opts")
        .value.split("\n")
        .map((s) => s.trim())
        .filter(Boolean);
      const seen = new Set();
      const opts = [];
      let hasDup = false;
      for (const o of rawLines) {
        if (seen.has(o)) {
          hasDup = true;
          break;
        }
        seen.add(o);
        opts.push(o);
      }
      if (!q) {
        showToast("请输入投票主题");
        return;
      }
      if (opts.length < 3) {
        showToast("至少需要 3 个不重复的选项");
        return;
      }
      if (hasDup) {
        showToast("选项不能重复（逐行对比，区分大小写）");
        return;
      }
      const isPublic = $("isPublic").checked;

      const exp = new Date(Date.now() + Math.max(1, mins) * 60 * 1000).toISOString();
      try {
        const poll = await api("/polls/createPoll", {
          method: "POST",
          body: { question: q, options: opts, expires_at: exp, is_public: isPublic },
        });
        const link = `${location.origin}/#/poll/${poll.id}`;
        console.log("created poll", poll);
        $("created").innerHTML = `
          已发布：<code>${escapeHtml(poll.id)}</code>
          <div style="margin-top:6px;">
            链接：<a href="${escapeHtml(link)}">${escapeHtml(link)}</a>
            <button class="btnSmall" id="copyLink">复制链接</button>
          </div>
        `;
        $("copyLink").addEventListener("click", async () => {
          await navigator.clipboard.writeText(link);
        });
      } catch (e) {
        console.error(e);
        showToast(e.message || "创建投票失败");
      }
    });
    return;
  }

  if (r.parts[0] === "join") {
    app().innerHTML = layout(
      "参与投票",
      `
        <section class="card">
          <div class="row">
            <label>
              输入投票ID查询
              <input id="pollId" placeholder="poll_id" />
            </label>
            <button id="go">进入投票</button>
            <button id="refreshPublic">刷新公开投票列表</button>
          </div>
          <div id="publicList" class="polls"></div>
        </section>
      `,
    );
    bindNavButtons(app());

    $("go").addEventListener("click", () => {
      const id = $("pollId").value.trim();
      if (!id) return;
      nav(`/poll/${encodeURIComponent(id)}`);
    });

    async function loadPublic() {
      try {
        const resp = await api("/polls/public?include_closed=false");
        const polls = resp?.polls || [];
        const root = $("publicList");
        root.innerHTML = polls.length
          ? ""
          : `<p class="muted">暂无公开投票（或你还没创建公开投票）。</p>`;
        for (const p of polls) {
          const el = document.createElement("div");
          el.className = "poll";
          el.innerHTML = `
            <div class="pollHead">
              <div>
                <div class="question">${escapeHtml(p.question)}</div>
                <div class="meta">
                  <span>id: <code>${escapeHtml(p.id)}</code></span>
                  <span>状态: <b>${p.isClosed ? "已关闭" : "进行中"}</b></span>
                </div>
              </div>
              <div class="actions">
                <button data-open="${escapeHtml(p.id)}">进入</button>
              </div>
            </div>
          `;
          el.querySelector("[data-open]").addEventListener("click", () =>
            nav(`/poll/${encodeURIComponent(p.id)}`),
          );
          root.appendChild(el);
        }
      } catch (e) {
        console.error(e);
        showToast(e.message || "刷新公开投票列表失败");
      }
    }

    $("refreshPublic").addEventListener("click", loadPublic);
    loadPublic();
    return;
  }

  if (r.parts[0] === "poll" && r.parts[1]) {
    const pollId = decodeURIComponent(r.parts[1]);
    app().innerHTML = layout(
      "投票详情",
      `
        <section class="card">
          <div class="row">
            <button id="refreshPoll">刷新</button>
            <button id="undo">撤销投票</button>
            <span id="pollSseStatus" class="badge">SSE: connecting</span>
            <span class="muted">分享链接：<code id="share"></code></span>
            <button class="btnSmall" id="copyShare">复制</button>
          </div>
          <div id="pollView" class="polls"></div>
        </section>
      `,
    );
    bindNavButtons(app());

    const link = `${location.origin}/#/poll/${encodeURIComponent(pollId)}`;
    $("share").textContent = link;
    $("copyShare").addEventListener("click", async () => navigator.clipboard.writeText(link));

    // Close results SSE when leaving poll page.
    if (window.__pollSSE) {
      window.__pollSSE.close();
      window.__pollSSE = null;
    }

    const pollSseStatus = $("pollSseStatus");
    const uid = getUserId();

    let loadInFlight = null;
    let loadQueued = false;

    async function load() {
      if (loadInFlight) {
        loadQueued = true;
        return loadInFlight;
      }
      // #region agent log
      fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
        body: JSON.stringify({
          sessionId: "cd7378",
          runId: "sse-verify",
          hypothesisId: "H_LOAD",
          location: "web/static/app.js:load",
          message: "poll_load_start",
          data: { pollId, loadQueued, hasInFlight: Boolean(loadInFlight) },
          timestamp: Date.now(),
        }),
      }).catch(() => {});
      // #endregion

      loadInFlight = (async () => {
        console.log(`[DEBUG] 开始加载投票数据 - PollID: ${pollId}`);
        const p = normalizePoll(await api(`/polls/${encodeURIComponent(pollId)}`));
        console.log(`[DEBUG] 获取到投票数据 - PollID: ${pollId}`, p);
      const votes = p.votes || {};
        console.log(`[DEBUG] 投票票数数据 - PollID: ${pollId}`, votes);
        const totalVotes = Object.values(votes).reduce((sum, count) => sum + (count || 0), 0);
        console.log(`[DEBUG] 总票数统计 - PollID: ${pollId}, Total: ${totalVotes}`);
      const root = $("pollView");
      const createdAt = parseProtoTimestamp(p.createdAt);
      const expiresAt = parseProtoTimestamp(p.expiresAt);
      const createdBy = p.createdBy || "-";
      const optionsHtml = (p.options || [])
        .map((opt) => {
          const n = votes[opt] ?? 0;
          const disabled = p.isClosed ? "disabled" : "";
          return `
            <div class="option">
              <div class="optName">${escapeHtml(opt)}</div>
              <div class="optMeta">
                <span class="count">${n}</span>
                <button ${disabled} data-vote="${escapeHtml(opt)}">投票</button>
              </div>
            </div>
          `;
        })
        .join("");

      root.innerHTML = `
        <div class="poll">
          <div class="pollHead">
            <div>
              <div class="question">${escapeHtml(p.question)}</div>
              <div class="meta">
                <span>id: <code>${escapeHtml(p.id)}</code></span>
                <span>创建人: <code>${escapeHtml(createdBy)}</code></span>
                <span>公开: <b>${p.isPublic ? "是" : "否"}</b></span>
                <span>状态: <b>${p.isClosed ? "已关闭" : "进行中"}</b></span>
                <span>起止: <b>${escapeHtml(formatExact(createdAt))}</b> ~ <b>${escapeHtml(formatExact(expiresAt))}</b></span>
                <span>${escapeHtml(formatRelativeToNow(expiresAt))}</span>
              </div>
            </div>
          </div>
          <div class="options">${optionsHtml}</div>
        </div>
      `;

      root.querySelectorAll("[data-vote]").forEach((b) => {
        b.addEventListener("click", async () => {
          const opt = b.getAttribute("data-vote");
          const originalText = b.textContent;

          // 禁用按钮防止重复投票
          b.disabled = true;
          b.textContent = "投票中...";

          try {
            console.log("开始投票", pollId, opt);

            // 发起投票请求
            const response = await api(`/votes/${encodeURIComponent(pollId)}/vote`, {
              method: "POST",
              body: { option: opt },
            });

            console.log("投票响应", response);

            // 立即更新页面，不等待SSE
            await load();

            // 显示成功消息
            showToast("投票成功", "success");

            console.log("投票页面已更新");
          } catch (e) {
            console.error("投票失败", e);
            showToast(e.message || "投票失败");

            // 恢复按钮状态
            b.disabled = false;
            b.textContent = originalText;
          } finally {
            // 无论成功失败，都恢复按钮状态（成功后按钮已被禁用）
            if (!b.disabled) {
              b.textContent = originalText;
            }
          }
        });
      });

      // #region agent log
      try {
        const voteSum = Object.values(votes).reduce((a, b) => a + Number(b || 0), 0);
        fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
          method: "POST",
          headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
          body: JSON.stringify({
            sessionId: "cd7378",
            runId: "sse-verify",
            hypothesisId: "H_LOAD",
            location: "web/static/app.js:load",
            message: "poll_load_rendered",
            data: { pollId: p.id, isClosed: Boolean(p.isClosed), voteSum },
            timestamp: Date.now(),
          }),
        }).catch(() => {});
      } catch {
        // ignore
      }
      // #endregion
      })()
        .finally(async () => {
          loadInFlight = null;
          if (loadQueued) {
            loadQueued = false;
            try {
              await load();
            } catch {
              // ignore
            }
          }
        });
      return loadInFlight;
    }

    // SSE连接管理器
    let reconnectAttempts = 0;
    const maxReconnectAttempts = 5;
    let reconnectTimeout = null;

    function connectSSE() {
      try {
        // 关闭旧连接
        if (window.__pollSSE) {
          window.__pollSSE.close();
          window.__pollSSE = null;
        }

        const es = new EventSource(`/events/polls/${encodeURIComponent(pollId)}?user_id=${encodeURIComponent(uid || "")}`);
        window.__pollSSE = es;

        es.onopen = () => {
          console.log("SSE连接已建立", pollId);
          pollSseStatus.textContent = "SSE: connected";
          pollSseStatus.style.color = "green";
          reconnectAttempts = 0; // 重置重试计数
          // #region agent log
          fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
            method: "POST",
            headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
            body: JSON.stringify({
              sessionId: "cd7378",
              runId: "sse-verify",
              hypothesisId: "H_SSE",
              location: "web/static/app.js:poll_sse",
              message: "poll_sse_open",
              data: { pollId, uid, url: `/events/polls/${pollId}?user_id=${uid || ""}` },
              timestamp: Date.now(),
            }),
          }).catch(() => {});
          // #endregion
        };

        es.onerror = () => {
          console.log("SSE连接错误，准备重连", pollId);
          pollSseStatus.textContent = "SSE: reconnecting";
          pollSseStatus.style.color = "orange";

          // 关闭错误连接
          es.close();
          window.__pollSSE = null;

          // 清理之前的重连定时器
          if (reconnectTimeout) {
            clearTimeout(reconnectTimeout);
          }

          // 尝试重连
          if (reconnectAttempts < maxReconnectAttempts) {
            reconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, reconnectAttempts - 1), 30000); // 指数退避，最大30秒
            console.log(`SSE重试第${reconnectAttempts}次，${delay}ms后重连`);

            reconnectTimeout = setTimeout(() => {
              connectSSE();
            }, delay);
          } else {
            pollSseStatus.textContent = "SSE: failed";
            pollSseStatus.style.color = "red";
            console.error("SSE重连失败次数过多，放弃重连");
          }
        };

        es.addEventListener("poll_invalidate", async (evt) => {
          console.log("收到SSE更新事件", pollId, evt.data);
          // #region agent log
          fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
            method: "POST",
            headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
            body: JSON.stringify({
              sessionId: "cd7378",
              runId: "post-sse",
              hypothesisId: "SSE_POLL",
              location: "web/static/app.js:poll_sse",
              message: "poll_invalidate_received",
              data: { pollId, data: evt && evt.data ? String(evt.data).slice(0, 200) : "" },
              timestamp: Date.now(),
            }),
          }).catch(() => {});
          // #endregion

          try {
            console.log(`[DEBUG] 准备通过SSE更新数据 - PollID: ${pollId}`);

            // 添加短暂延迟确保服务器数据已更新
            await new Promise(resolve => setTimeout(resolve, 100));

            // 设置超时
            await Promise.race([
              load(),
              new Promise((_, reject) =>
                setTimeout(() => reject(new Error("加载超时")), 5000)
              )
            ]);

            console.log(`[DEBUG] SSE更新完成 - PollID: ${pollId}`);
          } catch (e) {
            console.error("poll SSE刷新失败", e);
            showToast("实时更新失败，请手动刷新", "error");
          }
        });

        // 添加连接关闭处理
        es.addEventListener("close", () => {
          console.log("SSE连接已关闭", pollId);
          pollSseStatus.textContent = "SSE: disconnected";
          pollSseStatus.style.color = "gray";
        });

      } catch (e) {
        console.error("SSE连接初始化失败", e);
        pollSseStatus.textContent = "SSE: unavailable";
        pollSseStatus.style.color = "red";
      }
    }

    // 建立连接
    connectSSE();

    $("refreshPoll").addEventListener("click", () => load().catch((e) => showToast(e.message || "刷新投票详情失败")));
    const undoBtn = $("undo");
    if (undoBtn) {
      undoBtn.addEventListener("click", async () => {
        const originalText = undoBtn.textContent;

        // 禁用按钮防止重复操作
        undoBtn.disabled = true;
        undoBtn.textContent = "撤销中...";

        try {
          console.log("开始撤销投票", pollId);

          // 发起撤销投票请求
          await api(`/votes/${encodeURIComponent(pollId)}/vote`, { method: "DELETE" });

          // 立即更新页面，不等待SSE
          await load();

          // 显示成功消息
          showToast("撤销成功", "success");

          console.log("撤销投票页面已更新");
        } catch (e) {
          console.error("撤销投票失败", e);
          showToast(e.message || "撤销投票失败");

          // 恢复按钮状态
          undoBtn.disabled = false;
          undoBtn.textContent = originalText;
        } finally {
          // 恢复按钮状态
          undoBtn.disabled = false;
          undoBtn.textContent = originalText;
        }
      });
    }

    load().catch((e) => showToast(e.message || "加载投票详情失败"));
    return;
  }

  if (r.parts[0] === "results") {
    const uid = getUserId();
    app().innerHTML = layout(
      "投票结果",
      `
        <section class="card">
          <div class="row">
            <button id="refreshAll">刷新</button>
            <span id="statusEl" class="badge">SSE: connecting</span>
          </div>
          <h3>我创建的投票</h3>
          <div id="myCreated" class="polls"></div>
          <h3 style="margin-top:16px;">我参与的投票</h3>
          <div id="myVotes" class="polls"></div>
          <h3 style="margin-top:16px;">公共主题投票结果</h3>
          <div id="publicStats" class="polls"></div>
        </section>
      `,
    );
    bindNavButtons(app());

    function pollCard(p, extraHtml = "") {
      p = normalizePoll(p);
      const votes = p.votes || {};
      const createdAt = parseProtoTimestamp(p.createdAt);
      const expiresAt = parseProtoTimestamp(p.expiresAt);
      const createdBy = p.createdBy || "-";
      const opts = (p.options || [])
        .map((opt) => {
          const n = votes[opt] ?? 0;
          return `<div class="option"><div class="optName">${escapeHtml(opt)}</div><div class="optMeta"><span class="count">${n}</span></div></div>`;
        })
        .join("");
      return `
        <div class="poll">
          <div class="pollHead">
            <div>
              <div class="question">${escapeHtml(p.question)}</div>
              <div class="meta">
                <span>id: <code>${escapeHtml(p.id)}</code></span>
                <span>创建人: <code>${escapeHtml(createdBy)}</code></span>
                <span>公开: <b>${p.isPublic ? "是" : "否"}</b></span>
                <span>状态: <b>${p.isClosed ? "已关闭" : "进行中"}</b></span>
                <span>起止: <b>${escapeHtml(formatExact(createdAt))}</b> ~ <b>${escapeHtml(formatExact(expiresAt))}</b></span>
                <span>${escapeHtml(formatRelativeToNow(expiresAt))}</span>
              </div>
            </div>
            ${extraHtml}
          </div>
          <div class="options">${opts}</div>
        </div>
      `;
    }

    let currentPayload = { myVotes: null, publicStats: null, myCreated: null };

    function renderResults(payload) {
      console.log("[DEBUG] 开始渲染结果页面", payload);

      if (!payload) {
        console.error("[DEBUG] renderResults: payload is null");
        return;
      }

      const mv = payload?.myVotes;
      const ps = payload?.publicStats;
      const mc = payload?.myCreated;

      console.log("[DEBUG] 解构后的数据:", { mv, ps, mc });

      // Dedup + priority:
      // 1) 我创建的投票
      // 2) 我参与的投票
      // 3) 公共主题投票
      const taken = new Set();

      const createdPolls = mc?.polls || [];
      const createdFiltered = createdPolls.filter((p) => {
        const id = p?.id;
        if (!id || taken.has(id)) return false;
        taken.add(id);
        return true;
      });

      const myVotesAll = mv?.votes || [];
      const myVotesFiltered = myVotesAll.filter((v) => {
        const pollId = v?.pollId ?? v?.poll_id ?? v?.poll?.id;
        if (!pollId || taken.has(pollId)) return false;
        taken.add(pollId);
        return true;
      });

      const publicPolls = ps?.polls || [];
      const publicFiltered = publicPolls.filter((p) => {
        const id = p?.id;
        if (!id || taken.has(id)) return false;
        taken.add(id);
        return true;
      });

      // 我创建的投票
      if (!createdFiltered.length) {
        $("myCreated").innerHTML = `<p class="muted">你还没有创建任何投票。</p>`;
      } else {
        $("myCreated").innerHTML = createdFiltered
          .map((p) =>
            pollCard(
              p,
              `<div class="actions">
                <button data-open="${escapeHtml(p.id)}">进入</button>
                <button data-close="${escapeHtml(p.id)}">关闭</button>
                <button class="del" data-del="${escapeHtml(p.id)}">删除</button>
              </div>`,
            ),
          )
          .join("");

        $("myCreated")
          .querySelectorAll("[data-open]")
          .forEach((b) => b.addEventListener("click", () => nav(`/poll/${encodeURIComponent(b.getAttribute("data-open"))}`)));

        $("myCreated")
          .querySelectorAll("[data-close]")
          .forEach((b) =>
            b.addEventListener("click", async () => {
              const id = b.getAttribute("data-close");
              try {
                await api(`/polls/close/${encodeURIComponent(id)}`, { method: "GET" });
                showToast("已关闭投票", "success");
              } catch (e) {
                console.error(e);
                showToast(e.message || "关闭失败");
              }
            }),
          );

        $("myCreated")
          .querySelectorAll("[data-del]")
          .forEach((b) =>
            b.addEventListener("click", async () => {
              const id = b.getAttribute("data-del");
              if (!confirm(`确认删除投票 ${id}？`)) return;
              try {
                await api(`/polls/delete/${encodeURIComponent(id)}`, { method: "GET" });
                showToast("已删除投票", "success");
              } catch (e) {
                console.error(e);
                showToast(e.message || "删除失败");
              }
            }),
          );
      }

      // 我参与的投票
      $("myVotes").innerHTML = myVotesFiltered.length
        ? myVotesFiltered
            .map((v) => {
              const p = v.poll;
              const pollId = v.pollId ?? v.poll_id ?? p?.id;
              return pollCard(
                p,
                `<div class="actions"><span class="badge">我选择：<b>${escapeHtml(v.option)}</b></span><button data-open="${escapeHtml(pollId)}">进入</button></div>`,
              );
            })
            .join("")
        : myVotesAll.length
          ? `<p class="muted">你参与的投票已在上方“我创建的投票”中展示。</p>`
          : `<p class="muted">你还没有参与任何投票。</p>`;
      $("myVotes")
        .querySelectorAll("[data-open]")
        .forEach((b) =>
          b.addEventListener("click", () => {
            const id = b.getAttribute("data-open");
            if (!id) {
              showToast("投票ID缺失，无法打开");
              return;
            }
            nav(`/poll/${encodeURIComponent(id)}`);
          }),
        );

      // 公共主题投票结果
      $("publicStats").innerHTML = publicFiltered.length
        ? publicFiltered.map((p) => pollCard(p, `<div class="actions"><button data-open="${escapeHtml(p.id)}">进入</button></div>`)).join("")
        : publicPolls.length
          ? `<p class="muted">公共投票已在上方板块展示。</p>`
          : `<p class="muted">暂无公共投票。</p>`;
      $("publicStats")
        .querySelectorAll("[data-open]")
        .forEach((b) => b.addEventListener("click", () => nav(`/poll/${encodeURIComponent(b.getAttribute("data-open"))}`)));
    }

    async function loadAllOnce() {
      try {
        console.log("[DEBUG] 开始加载所有结果数据");
        const mv = await api("/users/me/votes");
        console.log("[DEBUG] 获取到我的投票数据", mv);

        const ps = await api("/polls/public/stats?include_closed=true");
        console.log("[DEBUG] 获取到公共统计数据", ps);

        const mc = await api(`/polls/my_created/stats?include_closed=true`);
        console.log("[DEBUG] 获取到我的创建数据", mc);

        currentPayload = { myVotes: mv, publicStats: ps, myCreated: mc };
        renderResults(currentPayload);
        console.log("[DEBUG] 结果页面数据加载完成");
      } catch (e) {
        console.error("[DEBUG] 加载结果数据失败", e);
        showToast("加载数据失败，请刷新页面", "error");
      }
    }

    // 结果页面SSE连接管理器
    let resultsReconnectAttempts = 0;
    const maxResultsReconnectAttempts = 5;
    let resultsReconnectTimeout = null;

    function connectResultsSSE() {
      // 关闭旧连接
      if (window.__resultsSSE) {
        window.__resultsSSE.close();
        window.__resultsSSE = null;
      }

      try {
        const es = new EventSource(`/events/results?user_id=${encodeURIComponent(uid || "")}`);
        window.__resultsSSE = es;

        es.onopen = () => {
          console.log("结果页面SSE连接已建立");
          statusEl.textContent = "SSE: connected";
          statusEl.style.color = "green";
          resultsReconnectAttempts = 0; // 重置重试计数
        };

        es.onerror = () => {
          console.log("结果页面SSE连接错误，准备重连");
          statusEl.textContent = "SSE: reconnecting";
          statusEl.style.color = "orange";

          // 关闭错误连接
          es.close();
          window.__resultsSSE = null;

          // 清理之前的重连定时器
          if (resultsReconnectTimeout) {
            clearTimeout(resultsReconnectTimeout);
          }

          // 尝试重连
          if (resultsReconnectAttempts < maxResultsReconnectAttempts) {
            resultsReconnectAttempts++;
            const delay = Math.min(1000 * Math.pow(2, resultsReconnectAttempts - 1), 30000); // 指数退避，最大30秒
            console.log(`结果页面SSE重试第${resultsReconnectAttempts}次，${delay}ms后重连`);

            resultsReconnectTimeout = setTimeout(() => {
              connectResultsSSE();
            }, delay);
          } else {
            statusEl.textContent = "SSE: failed";
            statusEl.style.color = "red";
            console.error("结果页面SSE重连失败次数过多，放弃重连");
          }
        };

        es.addEventListener("invalidate", async (evt) => {
          console.log("收到结果页面SSE更新事件", evt.data);

          let inv = null;
          try {
            inv = JSON.parse(evt.data);
          } catch {
            inv = null;
          }

          try {
            console.log(`[DEBUG] 准备通过SSE更新结果页面`);
            console.log(`[DEBUG] 更新触发数据:`, inv);

            // 添加短暂延迟确保服务器数据已更新
            await new Promise(resolve => setTimeout(resolve, 100));

            // If server doesn't specify, refresh all.
            const needMyVotes = inv?.myVotes ?? true;
            const needPublicStats = inv?.publicStats ?? true;
            const needMyCreated = inv?.myCreated ?? true;

            console.log(`[DEBUG] 需要更新的数据:`, { needMyVotes, needPublicStats, needMyCreated });

            const nextPayload = { ...currentPayload };
            const tasks = [];

            if (needMyVotes) {
              tasks.push(api("/users/me/votes").then((v) => {
                console.log("[DEBUG] 我的投票数据已更新:", v);
                nextPayload.myVotes = v;
              }));
            }
            if (needPublicStats) {
              tasks.push(api("/polls/public/stats?include_closed=true").then((v) => {
                console.log("[DEBUG] 公共统计数据已更新:", v);
                nextPayload.publicStats = v;
              }));
            }
            if (needMyCreated) {
              tasks.push(api("/polls/my_created/stats?include_closed=true").then((v) => {
                console.log("[DEBUG] 我的创建数据已更新:", v);
                nextPayload.myCreated = v;
              }));
            }

            // 设置超时
            await Promise.race([
              Promise.all(tasks),
              new Promise((_, reject) =>
                setTimeout(() => reject(new Error("数据加载超时")), 10000)
              )
            ]);

            currentPayload = nextPayload;
            console.log("[DEBUG] 准备渲染结果页面");
            renderResults(currentPayload);
            console.log("结果页面数据已更新");
          } catch (e) {
            console.error("结果页面SSE刷新失败", e);
            console.error("错误详情:", {
                error: e,
                eventData: evt.data,
                inv: inv,
                timestamp: new Date().toISOString()
            });
            showToast("实时更新失败，请手动刷新", "error");
          }
        });

        // 添加连接关闭处理
        es.addEventListener("close", () => {
          console.log("结果页面SSE连接已关闭");
          statusEl.textContent = "SSE: disconnected";
          statusEl.style.color = "gray";
        });

      } catch (e) {
        console.error("结果页面SSE连接初始化失败", e);
        statusEl.textContent = "SSE: unavailable";
        statusEl.style.color = "red";
      }
    }

    // 建立连接
    connectResultsSSE();

    $("refreshAll").addEventListener("click", () => loadAllOnce().catch((e) => showToast(e.message || "刷新结果失败")));
    loadAllOnce().catch((e) => showToast(e.message || "加载结果失败"));
    return;
  }

  nav("/home");
} catch (e) {
  // #region agent log
  fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      "X-Debug-Session-Id": "cd7378",
    },
    body: JSON.stringify({
      sessionId: "cd7378",
      runId: "pre-fix",
      hypothesisId: "H1",
      location: "web/static/app.js:673",
      message: "render_error",
      data: {
        error: e && e.message ? e.message : String(e),
        hasParseProtoTimestamp: typeof parseProtoTimestamp === "function",
      },
      timestamp: Date.now(),
    }),
  }).catch(() => {});
  // #endregion

  console.error(e);
  app().innerHTML = layout("出错了", `<section class="card"><p class="muted">${escapeHtml(e.message || String(e))}</p></section>`);
  bindNavButtons(app());
}
}

window.addEventListener("hashchange", () => render());
render();

// #region agent log
window.addEventListener("error", (event) => {
  try {
    fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Debug-Session-Id": "cd7378",
      },
      body: JSON.stringify({
        sessionId: "cd7378",
        runId: "pre-fix2",
        hypothesisId: "H3",
        location: "web/static/app.js:685",
        message: "window_error",
        data: {
          message: event.message,
          filename: event.filename,
          lineno: event.lineno,
          colno: event.colno,
        },
        timestamp: Date.now(),
      }),
    }).catch(() => {});
  } catch {
    // ignore logging failure
  }
});

window.addEventListener("unhandledrejection", (event) => {
  try {
    fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        "X-Debug-Session-Id": "cd7378",
      },
      body: JSON.stringify({
        sessionId: "cd7378",
        runId: "pre-fix2",
        hypothesisId: "H4",
        location: "web/static/app.js:703",
        message: "unhandled_rejection",
        data: {
          reason: event.reason && event.reason.message ? event.reason.message : String(event.reason || ""),
        },
        timestamp: Date.now(),
      }),
    }).catch(() => {});
  } catch {
    // ignore logging failure
  }
});
// #endregion