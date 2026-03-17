#!/usr/bin/env bash
set -euo pipefail

command -v k6 >/dev/null 2>&1 || { echo "missing k6 (https://k6.io/docs/get-started/installation/)" >&2; exit 1; }

BASE_URL="${BASE_URL:-http://localhost:8080}"
RATE="${RATE:-200}"
DURATION="${DURATION:-60s}"

echo "Running benchmark against ${BASE_URL} (RATE=${RATE}, DURATION=${DURATION})"
k6 run -e BASE_URL="${BASE_URL}" -e RATE="${RATE}" -e DURATION="${DURATION}" scripts/bench_vote.js

