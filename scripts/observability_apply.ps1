$ErrorActionPreference = "Stop"

if (-not (Get-Command kubectl -ErrorAction SilentlyContinue)) {
  throw "kubectl not found in PATH."
}

Write-Host "Applying observability kustomization (includes redis-exporter, ServiceMonitors, PrometheusRule, Grafana dashboard)..."
kubectl apply -k deployments/k8s/observability | Out-Host

