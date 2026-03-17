$ErrorActionPreference = "Stop"

if (-not (Get-Command k6 -ErrorAction SilentlyContinue)) {
  throw "k6 not found in PATH. Install from https://k6.io/docs/get-started/installation/ and retry."
}

$base = $env:BASE_URL
if (-not $base) { $base = "http://localhost:8080" }

Write-Host "Running correctness tests against $base"
k6 run -e BASE_URL=$base scripts/test_correctness.js

