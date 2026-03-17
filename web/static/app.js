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

function genIdemKey(prefix = "req") {
  const uid = (typeof crypto !== "undefined" && crypto.randomUUID && crypto.randomUUID()) || null;
  if (uid) return `${prefix}-${uid}`;
  return `${prefix}-${Date.now()}-${Math.random().toString(16).slice(2)}`;
}

async function api(path, { method = "GET", body, headers: extraHeaders, idempotencyKey } = {}) {
const headers = { "X-User-Id": getUserId() };
if (body !== undefined) headers["Content-Type"] = "application/json";
if (idempotencyKey) headers["Idempotency-Key"] = idempotencyKey;
if (extraHeaders && typeof extraHeaders === "object") Object.assign(headers, extraHeaders);
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

    let publishInFlight = false;
    let publishIdemKey = "";

    function markDirty() {
      // Any input change should start a new logical "create" attempt,
      // so we must not reuse the previous idempotency key.
      publishIdemKey = "";
      if (publishInFlight) return;
      const btn = $("publish");
      if (btn) {
        btn.disabled = false;
        btn.textContent = "发布";
      }
      const created = $("created");
      if (created) created.innerHTML = "";
    }

    ["q", "mins", "opts", "isPublic"].forEach((id) => {
      const el = $(id);
      if (!el) return;
      el.addEventListener("input", markDirty);
      el.addEventListener("change", markDirty);
    });

    $("publish").addEventListener("click", async () => {
      if (publishInFlight) return;
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
      const btn = $("publish");
      const oldText = btn.textContent;
      try {
        publishInFlight = true;
        btn.disabled = true;
        btn.textContent = "发布中...";

        // Same key for rapid clicks / retries within this create action.
        // Backend binds idem:{user}:{key} -> poll_id for 5 minutes.
        if (!publishIdemKey) publishIdemKey = genIdemKey("create");

        const poll = await api("/polls/createPoll", {
          method: "POST",
          body: { question: q, options: opts, expires_at: exp, is_public: isPublic },
          idempotencyKey: publishIdemKey,
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

        // After success we keep button disabled; any further input change will re-enable it
        // and generate a fresh idempotency key for the new create attempt.
        btn.textContent = "已发布";
      } catch (e) {
        console.error(e);
        showToast(e.message || "创建投票失败");
        // allow retry with the same idempotency key (and restore UI state)
        btn.disabled = false;
        btn.textContent = oldText || "发布";
      } finally {
        publishInFlight = false;
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
        const p = normalizePoll(await api(`/polls/${encodeURIComponent(pollId)}`));
      const votes = p.votes || {};
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
          try {
            await api(`/votes/${encodeURIComponent(pollId)}/vote`, {
              method: "POST",
              body: { option: opt },
            });
            await load();
            showToast("投票成功", "success");
          } catch (e) {
            console.error(e);
            showToast(e.message || "投票失败");
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

    try {
      const es = new EventSource(`/events/polls/${encodeURIComponent(pollId)}?user_id=${encodeURIComponent(uid || "")}`);
      window.__pollSSE = es;
      es.onopen = () => {
        pollSseStatus.textContent = "SSE: connected";
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
        pollSseStatus.textContent = "SSE: reconnecting";
        // #region agent log
        fetch("http://127.0.0.1:7402/ingest/748aa12d-1387-4465-9d4b-c3e83bffd60c", {
          method: "POST",
          headers: { "Content-Type": "application/json", "X-Debug-Session-Id": "cd7378" },
          body: JSON.stringify({
            sessionId: "cd7378",
            runId: "sse-verify",
            hypothesisId: "H_SSE",
            location: "web/static/app.js:poll_sse",
            message: "poll_sse_error",
            data: { pollId, uid },
            timestamp: Date.now(),
          }),
        }).catch(() => {});
        // #endregion
      };
      es.addEventListener("poll_invalidate", async (evt) => {
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
          await load();
        } catch (e) {
          console.error("poll SSE refresh failed", e);
        }
      });
    } catch (e) {
      console.error(e);
      pollSseStatus.textContent = "SSE: unavailable";
    }

    $("refreshPoll").addEventListener("click", () => load().catch((e) => showToast(e.message || "刷新投票详情失败")));
    $("undo").addEventListener("click", async () => {
      try {
        await api(`/votes/${encodeURIComponent(pollId)}/vote`, { method: "DELETE" });
        await load();
      } catch (e) {
        console.error(e);
        showToast(e.message || "撤销投票失败");
      }
    });

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
            <span id="sseStatus" class="badge">SSE: connecting</span>
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
      const mv = payload?.myVotes;
      const ps = payload?.publicStats;
      const mc = payload?.myCreated;

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
      const mv = await api("/users/me/votes");
      const ps = await api("/polls/public/stats?include_closed=true");
      const mc = await api(`/polls/my_created/stats?include_closed=true`);
      currentPayload = { myVotes: mv, publicStats: ps, myCreated: mc };
      renderResults(currentPayload);
    }

      // SSE: incremental invalidation updates
    if (window.__resultsSSE) {
      window.__resultsSSE.close();
      window.__resultsSSE = null;
    }
    const statusEl = $("sseStatus");
    try {
      const es = new EventSource(`/events/results?user_id=${encodeURIComponent(uid || "")}`);
      window.__resultsSSE = es;
      es.onopen = () => {
        statusEl.textContent = "SSE: connected";
      };
      es.onerror = () => {
        statusEl.textContent = "SSE: reconnecting";
      };
        es.addEventListener("invalidate", async (evt) => {
          let inv = null;
          try {
            inv = JSON.parse(evt.data);
          } catch {
            inv = null;
          }
          try {
            // If server doesn't specify, refresh all.
            const needMyVotes = inv?.myVotes ?? true;
            const needPublicStats = inv?.publicStats ?? true;
            const needMyCreated = inv?.myCreated ?? true;

            const nextPayload = { ...currentPayload };
            const tasks = [];
            if (needMyVotes) tasks.push(api("/users/me/votes").then((v) => (nextPayload.myVotes = v)));
            if (needPublicStats) tasks.push(api("/polls/public/stats?include_closed=true").then((v) => (nextPayload.publicStats = v)));
            if (needMyCreated) tasks.push(api("/polls/my_created/stats?include_closed=true").then((v) => (nextPayload.myCreated = v)));
            await Promise.all(tasks);
            currentPayload = nextPayload;
            renderResults(currentPayload);
          } catch (e) {
            console.error("invalidate refresh failed", e);
          }
        });
    } catch (e) {
      console.error(e);
      statusEl.textContent = "SSE: unavailable";
    }

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