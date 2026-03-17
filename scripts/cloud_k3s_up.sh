#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NS="${CLUSTER_NS:-voting}"
HTTP_NODEPORT="${HTTP_NODEPORT:-30080}"

require() {
  command -v "$1" >/dev/null 2>&1 || { echo "missing command: $1" >&2; exit 1; }
}

echo "==> Installing k3s (single-node)"
if ! command -v k3s >/dev/null 2>&1; then
  curl -sfL https://get.k3s.io | sh -
fi

export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
require kubectl

echo "==> Installing metrics-server (for HPA)"
kubectl apply -f deployments/k8s/addons/metrics-server.yaml

echo "==> Building app images (requires docker)"
require docker
docker build -f Dockerfile.grpcserver -t voting/grpcserver:local .
docker build -f Dockerfile.httpserver -t voting/httpserver:local .

echo "==> Importing images into k3s containerd"
docker save voting/grpcserver:local | k3s ctr images import -
docker save voting/httpserver:local | k3s ctr images import -

echo "==> Deploying manifests (k3s overlay, httpserver NodePort=${HTTP_NODEPORT})"
kubectl apply -k deployments/k8s/overlays/k3s

echo "==> Waiting for rollouts"
kubectl -n "${CLUSTER_NS}" rollout status deploy/grpcserver --timeout=180s
kubectl -n "${CLUSTER_NS}" rollout status deploy/httpserver --timeout=180s
kubectl -n "${CLUSTER_NS}" rollout status statefulset/redis --timeout=180s

echo ""
echo "Done."
echo "- httpserver:  http://<your_server_public_ip>:${HTTP_NODEPORT}/"
echo "- healthz:     http://<your_server_public_ip>:${HTTP_NODEPORT}/healthz"
echo ""
echo "Make sure your cloud security group allows inbound TCP ${HTTP_NODEPORT}."

