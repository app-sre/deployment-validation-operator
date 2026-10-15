#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
OPERATOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
CLUSTER_NAME="${KIND_CLUSTER_NAME:-dvo-test}"
DVO_IMAGE="localhost/dvo-operator:latest"
DVO_NAMESPACE="openshift-operators"
HTTPD_IMAGE="${HTTPD_TEST_IMAGE:-registry.access.redhat.com/ubi8/httpd-24:latest}"
CONTAINER_ENGINE="${CONTAINER_ENGINE:-$(command -v docker || command -v podman)}"

echo "==> Using container engine: ${CONTAINER_ENGINE}"

# 1. Delete existing cluster if present
if kind get clusters 2>/dev/null | grep -q "^${CLUSTER_NAME}$"; then
    echo "==> Deleting existing KinD cluster '${CLUSTER_NAME}'..."
    kind delete cluster --name "${CLUSTER_NAME}"
fi

# 2. Create KinD cluster
echo "==> Creating KinD cluster '${CLUSTER_NAME}'..."
kind create cluster \
    --name "${CLUSTER_NAME}" \
    --config "${SCRIPT_DIR}/config.yaml" \
    --wait 60s

# 3. Build the DVO operator image
echo "==> Building DVO operator image..."
cd "${OPERATOR_DIR}"
${CONTAINER_ENGINE} build -f build/Dockerfile -t "${DVO_IMAGE}" .

# 4. Load operator image into KinD
# Using save + image-archive instead of "kind load docker-image" because the
# latter can't find images in Podman's store (https://github.com/kubernetes-sigs/kind/issues/2038)
echo "==> Loading DVO operator image into KinD..."
${CONTAINER_ENGINE} save "${DVO_IMAGE}" | kind load image-archive /dev/stdin --name "${CLUSTER_NAME}"

# 5. Pull and load httpd image into KinD
echo "==> Preparing httpd image..."
${CONTAINER_ENGINE} pull "${HTTPD_IMAGE}"
${CONTAINER_ENGINE} save "${HTTPD_IMAGE}" | kind load image-archive /dev/stdin --name "${CLUSTER_NAME}"

# 6. Create namespace
echo "==> Creating ${DVO_NAMESPACE} namespace..."
kubectl create namespace "${DVO_NAMESPACE}" --dry-run=client -o yaml | kubectl apply -f -

# 7. Deploy DVO manifests
DEPLOY_DIR="${OPERATOR_DIR}/deploy/openshift"
echo "==> Deploying DVO resources..."

kubectl apply -f "${DEPLOY_DIR}/service-account.yaml" -n "${DVO_NAMESPACE}"
kubectl apply -f "${DEPLOY_DIR}/cluster-role.yaml"

# The upstream ClusterRoleBinding and RoleBinding reference the ServiceAccount in
# namespace "deployment-validation-operator". In KinD we deploy to "openshift-operators"
# to match what OLM does in production. Patch the SA namespace reference on-the-fly.
sed "s/namespace: deployment-validation-operator/namespace: ${DVO_NAMESPACE}/g" \
    "${DEPLOY_DIR}/cluster-role-binding.yaml" | kubectl apply -f -

kubectl apply -f "${DEPLOY_DIR}/role.yaml" -n "${DVO_NAMESPACE}"
sed "s/namespace: deployment-validation-operator/namespace: ${DVO_NAMESPACE}/g" \
    "${DEPLOY_DIR}/role-binding.yaml" | kubectl apply -f -

kubectl apply -f "${DEPLOY_DIR}/configmap.yaml" -n "${DVO_NAMESPACE}"

# Deploy service as NodePort
echo "==> Deploying DVO Service (NodePort)..."
kubectl apply -f "${DEPLOY_DIR}/service.yaml" -n "${DVO_NAMESPACE}"
kubectl patch svc deployment-validation-operator-metrics \
    -n "${DVO_NAMESPACE}" \
    --type='json' \
    -p='[
        {"op": "replace", "path": "/spec/type", "value": "NodePort"},
        {"op": "add", "path": "/spec/ports/0/nodePort", "value": 30383}
    ]'

# Deploy operator with KinD-compatible spec (single apply, no rolling update issues)
echo "==> Deploying DVO operator (patched for KinD)..."
sed -e "s|quay.io/deployment-validation-operator/dv-operator:latest|${DVO_IMAGE}|" \
    -e "s|imagePullPolicy: Always|imagePullPolicy: IfNotPresent|" \
    -e 's|value: "2m"|value: "10s"|' \
    "${DEPLOY_DIR}/operator.yaml" | \
    python3 -c "
import sys, yaml
doc = yaml.safe_load(sys.stdin)
spec = doc['spec']['template']['spec']
spec.pop('nodeSelector', None)
spec.pop('tolerations', None)
spec.pop('affinity', None)
yaml.dump(doc, sys.stdout, default_flow_style=False)
" | kubectl apply -f - -n "${DVO_NAMESPACE}"

# 8. Wait for DVO pod readiness
echo "==> Waiting for DVO operator to be ready..."
kubectl rollout status deployment/deployment-validation-operator \
    -n "${DVO_NAMESPACE}" \
    --timeout=120s

echo ""
echo "==> KinD cluster '${CLUSTER_NAME}' is ready!"
echo "    Metrics endpoint: http://localhost:8383/metrics"
echo ""
echo "    Run tests with: ./kind/run-tests.sh"
