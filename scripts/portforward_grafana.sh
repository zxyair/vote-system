#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-monitoring}"
RELEASE="${RELEASE:-kube-prometheus-stack}"
LOCAL_PORT="${LOCAL_PORT:-3000}"

command -v kubectl >/dev/null 2>&1 || { echo "missing kubectl" >&2; exit 1; }

echo "Port-forwarding Grafana to http://localhost:${LOCAL_PORT} (Ctrl+C to stop)"
kubectl -n "${NAMESPACE}" port-forward "svc/${RELEASE}-grafana" "${LOCAL_PORT}:80"

