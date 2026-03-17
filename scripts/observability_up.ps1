param(
  [string]$Namespace = "monitoring",
  [string]$Release = "kube-prometheus-stack"
)

$ErrorActionPreference = "Stop"

function Require-Cmd([string]$name) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    throw "Required command not found in PATH: $name"
  }
}

Require-Cmd helm
Require-Cmd kubectl

kubectl get ns $Namespace *> $null 2>&1
if ($LASTEXITCODE -ne 0) {
  kubectl create ns $Namespace | Out-Host
}

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts | Out-Host
helm repo update | Out-Host

helm upgrade --install $Release prometheus-community/kube-prometheus-stack `
  -n $Namespace `
  -f deployments/k8s/observability/kube-prometheus-stack-values.yaml | Out-Host

Write-Host ""
Write-Host "Grafana default credentials:"
Write-Host "- user: admin"
Write-Host "- pass: admin (see deployments/k8s/observability/kube-prometheus-stack-values.yaml)"
Write-Host ""
Write-Host "To access Grafana locally (kind):"
Write-Host "kubectl -n $Namespace port-forward svc/$Release-grafana 3000:80"

