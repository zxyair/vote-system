$ErrorActionPreference = "Stop"

param(
  [string]$MonitoringNamespace = "monitoring",
  [string]$Release = "kube-prometheus-stack"
)

Write-Host "Installing kube-prometheus-stack and applying voting observability manifests..."
& "$PSScriptRoot\observability_up.ps1" -Namespace $MonitoringNamespace -Release $Release

Write-Host ""
Write-Host "Next (run in separate terminals):"
Write-Host "- Grafana:     powershell scripts\portforward_grafana.ps1 -Namespace $MonitoringNamespace -Release $Release"
Write-Host "- Prometheus:  powershell scripts\portforward_prometheus.ps1 -Namespace $MonitoringNamespace -Release $Release"

