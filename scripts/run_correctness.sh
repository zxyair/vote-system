#!/usr/bin/env bash
set -euo pipefail

command -v k6 >/dev/null 2>&1 || { echo "missing k6 (https://k6.io/docs/get-started/installation/)" >&2; exit 1; }

BASE_URL="${BASE_URL:-http://localhost:8080}"
echo "Running correctness tests against ${BASE_URL}"
k6 run -e BASE_URL="${BASE_URL}" scripts/test_correctness.js

