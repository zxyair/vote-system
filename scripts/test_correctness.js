import http from "k6/http";
import { check, sleep } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

function jsonHeaders(userId, idemKey) {
  const h = {
    "Content-Type": "application/json",
    "X-User-Id": userId,
  };
  if (idemKey) h["Idempotency-Key"] = idemKey;
  return h;
}

function createPoll(userId, idemKey) {
  const body = JSON.stringify({
    question: "你更喜欢哪种技术话题？",
    options: ["Go 并发", "Kubernetes", "Redis"],
    expires_at: new Date(Date.now() + 60 * 60 * 1000).toISOString(),
    is_public: true,
  });
  const res = http.post(`${BASE_URL}/polls/createPoll`, body, {
    headers: jsonHeaders(userId, idemKey),
  });
  check(res, {
    "createPoll status 200": (r) => r.status === 200,
    "createPoll has id": (r) => !!r.json("id"),
  });
  return res.json();
}

function vote(pollId, userId, option, idemKey) {
  const res = http.post(
    `${BASE_URL}/votes/${pollId}/vote`,
    JSON.stringify({ option }),
    { headers: jsonHeaders(userId, idemKey) }
  );
  return res;
}

function undo(pollId, userId, idemKey) {
  return http.del(`${BASE_URL}/votes/${pollId}/vote`, null, {
    headers: jsonHeaders(userId, idemKey),
  });
}

function getPoll(pollId, userId) {
  const res = http.get(`${BASE_URL}/polls/${pollId}`, {
    headers: jsonHeaders(userId),
  });
  check(res, { "getPoll 200": (r) => r.status === 200 });
  return res.json();
}

export const options = {
  vus: 1,
  iterations: 1,
};

export default function () {
  // 1) Create idempotency: same idem key should return same poll id
  const creator = "creator_1";
  const idemCreate = "create-1";
  const p1 = createPoll(creator, idemCreate);
  const p2 = createPoll(creator, idemCreate);
  check(
    { p1, p2 },
    {
      "create idempotent: same poll id": (x) => x.p1.id === x.p2.id,
    }
  );
  const pollId = p1.id;

  // 2) Vote idempotency: same user + same idem key retry should not double count
  const voterA = "voter_a";
  const idemVote = "vote-a-1";
  const r1 = vote(pollId, voterA, "Go 并发", idemVote);
  const r2 = vote(pollId, voterA, "Go 并发", idemVote);
  check(r1, { "vote a first 200": (r) => r.status === 200 });
  check(r2, { "vote a retry 200": (r) => r.status === 200 });

  // 3) Duplicate vote without idempotency key should conflict (409)
  const voterB = "voter_b";
  const rb1 = vote(pollId, voterB, "Redis", "");
  const rb2 = vote(pollId, voterB, "Redis", "");
  check(rb1, { "vote b first 200": (r) => r.status === 200 });
  check(rb2, { "vote b duplicate 409": (r) => r.status === 409 });

  // 4) Undo restores count and allows re-vote
  const voterC = "voter_c";
  const rc1 = vote(pollId, voterC, "Kubernetes", "");
  check(rc1, { "vote c 200": (r) => r.status === 200 });
  const ru1 = undo(pollId, voterC, "undo-c-1");
  const ru2 = undo(pollId, voterC, "undo-c-1"); // idempotent undo retry
  check(ru1, { "undo c 200": (r) => r.status === 200 });
  check(ru2, { "undo c retry 200": (r) => r.status === 200 });
  const rc2 = vote(pollId, voterC, "Kubernetes", "");
  check(rc2, { "vote c again 200": (r) => r.status === 200 });

  // 5) Concurrency no-loss: multiple users vote once; final total should match expected
  const N = 20;
  for (let i = 0; i < N; i++) {
    const uid = `voter_x_${i}`;
    const res = vote(pollId, uid, "Redis", "");
    check(res, { "vote x 200": (r) => r.status === 200 });
  }

  sleep(0.2);
  const final = getPoll(pollId, creator);
  const votes = final.votes || {};
  const total =
    (votes["Go 并发"] || 0) + (votes["Kubernetes"] || 0) + (votes["Redis"] || 0);

  // expected total:
  // voterA counted once, voterB counted once, voterC counted once (after undo+revote),
  // plus N voters_x.
  const expected = 1 + 1 + 1 + N;
  check(
    { total, expected, votes },
    {
      "final total matches expected": (x) => x.total === x.expected,
    }
  );
}

