#!/usr/bin/env bash
set -euo pipefail

NAMESPACE="${NAMESPACE:-monitoring}"
RELEASE="${RELEASE:-kube-prometheus-stack}"

echo "Installing kube-prometheus-stack and applying voting observability manifests..."
./scripts/observability_up.sh

echo ""
echo "Tip: on a cloud host you can still port-forward over SSH, e.g.:"
echo "ssh -L 3000:localhost:3000 root@<server> -- bash -lc 'kubectl -n ${NAMESPACE} port-forward svc/${RELEASE}-grafana 3000:80'"
echo "ssh -L 9090:localhost:9090 root@<server> -- bash -lc 'kubectl -n ${NAMESPACE} port-forward svc/${RELEASE}-kube-prometheus-prometheus 9090:9090'"

