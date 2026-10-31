#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-dvo-test}"

echo "==> Deleting KinD cluster '${CLUSTER_NAME}'..."
kind delete cluster --name "${CLUSTER_NAME}"
echo "==> Done."
