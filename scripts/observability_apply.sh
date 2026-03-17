#!/usr/bin/env bash
set -euo pipefail

command -v kubectl >/dev/null 2>&1 || { echo "missing kubectl" >&2; exit 1; }

echo "Applying observability kustomization (redis-exporter, ServiceMonitors, PrometheusRule, Grafana dashboard)..."
kubectl apply -k deployments/k8s/observability

