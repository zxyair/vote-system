$ErrorActionPreference = "Stop"

if (-not (Get-Command k6 -ErrorAction SilentlyContinue)) {
  throw "k6 not found in PATH. Install from https://k6.io/docs/get-started/installation/ and retry."
}

$base = $env:BASE_URL
if (-not $base) { $base = "http://localhost:8080" }

$rate = $env:RATE
if (-not $rate) { $rate = "200" }

$duration = $env:DURATION
if (-not $duration) { $duration = "60s" }

Write-Host "Running benchmark against $base (RATE=$rate, DURATION=$duration)"
k6 run -e BASE_URL=$base -e RATE=$rate -e DURATION=$duration scripts/bench_vote.js

