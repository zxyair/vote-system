import http from "k6/http";
import { check } from "k6";

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const CREATOR = __ENV.CREATOR || "bench_creator";
const POLL_IDEM = __ENV.POLL_IDEM || "bench-create-1";

function headers(userId, idemKey) {
  const h = { "Content-Type": "application/json", "X-User-Id": userId };
  if (idemKey) h["Idempotency-Key"] = idemKey;
  return h;
}

function createOrGetPoll() {
  const body = JSON.stringify({
    question: "Bench poll: choose a topic",
    options: ["Go 并发", "Kubernetes", "Redis"],
    expires_at: new Date(Date.now() + 6 * 60 * 60 * 1000).toISOString(),
    is_public: false,
  });
  const res = http.post(`${BASE_URL}/polls/createPoll`, body, {
    headers: headers(CREATOR, POLL_IDEM),
  });
  check(res, { "create poll 200": (r) => r.status === 200 });
  return res.json("id");
}

let pollId = "";

export const options = {
  scenarios: {
    vote_rps: {
      executor: "constant-arrival-rate",
      rate: Number(__ENV.RATE || 200),
      timeUnit: "1s",
      duration: __ENV.DURATION || "60s",
      preAllocatedVUs: Number(__ENV.PRE_VUS || 50),
      maxVUs: Number(__ENV.MAX_VUS || 500),
    },
  },
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<300", "p(99)<800"],
  },
};

export function setup() {
  pollId = createOrGetPoll();
  return { pollId };
}

export default function (data) {
  // Make each request a unique user to avoid conflict; best for pure throughput.
  const uid = `u_${__VU}_${__ITER}`;
  const option = "Redis";
  const res = http.post(
    `${BASE_URL}/votes/${data.pollId}/vote`,
    JSON.stringify({ option }),
    { headers: headers(uid, "") }
  );
  check(res, { "vote status ok/conflict": (r) => r.status === 200 || r.status === 409 });
}

