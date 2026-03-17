#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-monitoring}"
RELEASE="${RELEASE:-kube-prometheus-stack}"

command -v helm >/dev/null 2>&1 || { echo "missing helm" >&2; exit 1; }
command -v kubectl >/dev/null 2>&1 || { echo "missing kubectl" >&2; exit 1; }

kubectl get ns "${NAMESPACE}" >/dev/null 2>&1 || kubectl create ns "${NAMESPACE}"

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo update

helm upgrade --install "${RELEASE}" prometheus-community/kube-prometheus-stack \
  -n "${NAMESPACE}" \
  -f deployments/k8s/observability/kube-prometheus-stack-values.yaml

echo ""
echo "Grafana default credentials: admin / admin"
echo "Port-forward (local): kubectl -n ${NAMESPACE} port-forward svc/${RELEASE}-grafana 3000:80"

