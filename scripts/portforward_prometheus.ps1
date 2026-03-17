$ErrorActionPreference = "Stop"

param(
  [string]$Namespace = "monitoring",
  [string]$Release = "kube-prometheus-stack",
  [int]$LocalPort = 9090
)

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
  throw "kubectl not found in PATH."
}

Write-Host "Port-forwarding Prometheus to http://localhost:$LocalPort ..."
Write-Host "Press Ctrl+C to stop."
kubectl -n $Namespace port-forward svc/$Release-kube-prometheus-prometheus ${LocalPort}:9090

