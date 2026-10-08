#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME="${KIND_CLUSTER_NAME:-dvo-test}"
TEST_IMAGE="${DVO_TEST_IMAGE:-quay.io/redhatqe/deployment-validation-operator-tests:latest}"
HTTPD_IMAGE="${HTTPD_TEST_IMAGE:-registry.access.redhat.com/ubi8/httpd-24:latest}"
CONTAINER_ENGINE="${CONTAINER_ENGINE:-$(command -v docker || command -v podman)}"
DVO_METRICS_URL="http://localhost:8383"

usage() {
    cat <<EOF
Usage: $(basename "$0") [--image <image>] [-- <pytest args>]

Options:
  --image IMG  Test container image (default: ${TEST_IMAGE})
  --           Pass remaining arguments to pytest

Environment variables:
  DVO_TEST_IMAGE      Test container image
  KIND_CLUSTER_NAME   KinD cluster name (default: dvo-test)
  CONTAINER_ENGINE    Container engine (default: docker or podman)
EOF
    exit 1
}

PYTEST_ARGS=()

while [[ $# -gt 0 ]]; do
    case "$1" in
        --image)
            TEST_IMAGE="$2"
            shift 2
            ;;
        --)
            shift
            PYTEST_ARGS=("$@")
            break
            ;;
        -h|--help)
            usage
            ;;
        *)
            PYTEST_ARGS+=("$1")
            shift
            ;;
    esac
done

if [[ ${#PYTEST_ARGS[@]} -eq 0 ]]; then
    PYTEST_ARGS=(-v -m "not ui")
fi

KUBECONFIG_TMP="$(mktemp)"
kind get kubeconfig --name "${CLUSTER_NAME}" > "${KUBECONFIG_TMP}"
chmod 644 "${KUBECONFIG_TMP}"

echo "==> Running tests from image: ${TEST_IMAGE}"
echo "    Metrics: ${DVO_METRICS_URL}"

${CONTAINER_ENGINE} run --rm \
    --network host \
    -v "${KUBECONFIG_TMP}:/tmp/kubeconfig:Z" \
    -e KUBECONFIG=/tmp/kubeconfig \
    -e DVO_METRICS_URL="${DVO_METRICS_URL}" \
    -e HTTPD_TEST_IMAGE="${HTTPD_IMAGE}" \
    "${TEST_IMAGE}" \
    pytest.sh "${PYTEST_ARGS[@]}"

rm -f "${KUBECONFIG_TMP}"
