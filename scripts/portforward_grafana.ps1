$ErrorActionPreference = "Stop"

param(
  [string]$Namespace = "monitoring",
  [string]$Release = "kube-prometheus-stack",
  [int]$LocalPort = 3000
)

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
  throw "kubectl not found in PATH."
}

Write-Host "Port-forwarding Grafana to http://localhost:$LocalPort ..."
Write-Host "Press Ctrl+C to stop."
kubectl -n $Namespace port-forward svc/$Release-grafana ${LocalPort}:80

