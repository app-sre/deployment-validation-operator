# DVO E2E Tests on KinD

Run the DVO e2e test suite on a local [KinD](https://kind.sigs.k8s.io/) cluster
instead of a full OpenShift cluster.

## Prerequisites

- [KinD](https://kind.sigs.k8s.io/docs/user/quick-start/#installation)
- [kubectl](https://kubernetes.io/docs/tasks/tools/)
- Docker or Podman
- Python 3 with PyYAML (`pip install pyyaml`)

## Usage

```bash
# 1. Create the cluster and deploy DVO
./kind/setup.sh

# 2. Run the tests
./kind/run-tests.sh

# 3. Tear down the cluster
./kind/teardown.sh
```

### What `setup.sh` does

1. Creates a KinD cluster with a NodePort mapping (port 8383) for metrics access
2. Builds the DVO operator image from source
3. Pulls and loads the httpd test image into the cluster
4. Deploys the DVO operator using the upstream manifests with on-the-fly patches
   (removes node affinity/tolerations, sets `imagePullPolicy: IfNotPresent`)
5. Waits for the operator to be ready

### What `run-tests.sh` does

Runs the test suite from the
`quay.io/redhatqe/deployment-validation-operator-tests:latest` container image.
Override with `--image <image>` or the `DVO_TEST_IMAGE` env var.

Extra pytest arguments can be passed after `--`:

```bash
./kind/run-tests.sh -- -k test_configmap_deletion --tb=short
```

## Environment variables

| Variable | Default | Description |
|---|---|---|
| `KIND_CLUSTER_NAME` | `dvo-test` | KinD cluster name |
| `CONTAINER_ENGINE` | auto-detected | `docker` or `podman` |
| `DVO_TEST_IMAGE` | `quay.io/redhatqe/deployment-validation-operator-tests:latest` | Test container image |
| `HTTPD_TEST_IMAGE` | `registry.access.redhat.com/ubi8/httpd-24:latest` | httpd image for test deployments |

## How it works

The DVO operator uses only standard Kubernetes APIs at runtime — no CRDs, no
OpenShift controllers. The adaptations for KinD are minimal:

- **Metrics access**: OpenShift Routes are replaced with a NodePort service.
  The `DVO_METRICS_URL` env var tells the tests to use it directly.
- **httpd image**: The test deployment image is configurable via
  `HTTPD_TEST_IMAGE` instead of pulling from the OpenShift internal registry.
- **Operator manifests**: Node selectors, tolerations, and affinity rules are
  stripped since KinD runs a single node.
