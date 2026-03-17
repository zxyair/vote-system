param(
  [string]$ClusterName = "voting",
  [string]$Namespace = "voting",
  [switch]$RecreateCluster
)

$ErrorActionPreference = "Stop"

function Require-Cmd([string]$name) {
  if (-not (Get-Command $name -ErrorAction SilentlyContinue)) {
    throw "Required command not found in PATH: $name"
  }
}

Require-Cmd kind
Require-Cmd kubectl
Require-Cmd docker

if ($RecreateCluster) {
  kind delete cluster --name $ClusterName | Out-Host
}

$clusters = kind get clusters 2>$null
if ($clusters -notcontains $ClusterName) {
  Write-Host "Creating kind cluster '$ClusterName'..."
  kind create cluster --name $ClusterName | Out-Host
} else {
  Write-Host "kind cluster '$ClusterName' already exists."
}

Write-Host "Installing metrics-server (for HPA)..."
kubectl apply -f deployments/k8s/addons/metrics-server.yaml | Out-Host

Write-Host "Building app images..."
docker build -f Dockerfile.grpcserver -t voting/grpcserver:local . | Out-Host
docker build -f Dockerfile.httpserver -t voting/httpserver:local . | Out-Host

Write-Host "Loading images into kind..."
kind load docker-image --name $ClusterName voting/grpcserver:local | Out-Host
kind load docker-image --name $ClusterName voting/httpserver:local | Out-Host

Write-Host "Deploying manifests..."
kubectl apply -k deployments/k8s/base | Out-Host

Write-Host "Waiting for rollouts..."
kubectl -n $Namespace rollout status deploy/grpcserver --timeout=180s | Out-Host
kubectl -n $Namespace rollout status deploy/httpserver --timeout=180s | Out-Host
kubectl -n $Namespace rollout status statefulset/redis --timeout=180s | Out-Host

Write-Host ""
Write-Host "Port-forwarding httpserver to http://localhost:8080 ..."
Write-Host "Press Ctrl+C to stop port-forward."
kubectl -n $Namespace port-forward svc/httpserver 8080:80

