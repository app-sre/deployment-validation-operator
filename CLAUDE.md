# Deployment Validation Operator (DVO)

A Kubernetes operator that continuously validates deployments and other resources against curated best practices using kube-linter, with Prometheus metrics for monitoring compliance.

## Project Overview

**Type**: Kubernetes Operator  
**Language**: Go  
**Framework**: Operator SDK with controller-runtime  
**Primary Purpose**: Continuous deployment validation and compliance monitoring  
**Target Users**: Platform engineers, SREs, DevOps teams, cluster administrators

## Technical Stack

- **Language**: Go 1.24+ with modern Go modules
- **Kubernetes Integration**: controller-runtime v0.19.7
- **Validation Engine**: kube-linter v0.6.8 (StackRox)
- **Metrics**: Prometheus client for metrics exposure
- **Configuration**: Viper for configuration management
- **Logging**: Zap for structured logging
- **Testing**: Ginkgo for BDD-style testing

## Architecture & Core Concepts

### Operator Pattern
- **Controller-Runtime**: Uses Kubernetes controller-runtime for efficient resource watching
- **No CRDs**: Currently operates without custom resource definitions
- **Read-Only**: Monitors and validates resources without modification
- **Metrics-Driven**: Reports validation results via Prometheus metrics

### Validation Engine
- **kube-linter Integration**: Continuous version of StackRox's static analysis tool
- **Best Practices Focus**: Emphasizes fault-tolerance and security
- **Configurable Checks**: Supports custom validation rules via ConfigMap
- **Resource Exclusion**: Annotation-based and namespace-based filtering

### Metrics and Monitoring
- **Prometheus Metrics**: Gauge metrics reporting validation failures
- **Standard Labels**: `name`, `namespace`, and `kind` for all metrics
- **Value Semantics**: `1` indicates failed validation, `0` indicates success
- **Grafana Integration**: Pre-built dashboard templates available

## Default Validation Checks

The operator enables these security and reliability checks by default:
- **Security**: `privileged-container`, `privilege-escalation-container`, `run-as-non-root`
- **Network**: `host-ipc`, `host-network`, `host-pid`, `non-isolated-pod`
- **Reliability**: `pdb-max-unavailable`, `pdb-min-available`
- **Resources**: `unset-cpu-requirements`, `unset-memory-requirements`
- **System**: `unsafe-sysctls`

## Development Workflow

### Local Development
```bash
# Build the operator binary
make go-build

# Run locally with development settings
POD_NAMESPACE="deployment-validation-operator" \
WATCH_NAMESPACE="" \
NAMESPACE_IGNORE_PATTERN='^(openshift.*|kube-.*)$' \
build/_output/bin/deployment-validation-operator \
--kubeconfig=$HOME/.kube/config --zap-devel

# Check exposed metrics
curl localhost:8383/metrics
```

### Testing
```bash
# Run unit tests
make test

# Run end-to-end tests (requires ginkgo and KUBECONFIG)
make e2e-test
```

## AI Development Guidelines

### Code Architecture Patterns

#### Controller Implementation
```go
// Follow controller-runtime patterns
type MyReconciler struct {
    client.Client
    Log    logr.Logger
    Scheme *runtime.Scheme
}

func (r *MyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // Standard reconcile loop implementation
}
```

#### Metrics Integration
```go
// Use Prometheus client patterns
var (
    validationFailures = prometheus.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "dvo_validation_failures_total",
            Help: "Number of validation failures",
        },
        []string{"name", "namespace", "kind", "check"},
    )
)
```

### Configuration Management

#### ConfigMap Structure
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: deployment-validation-operator-config
data:
  config.yaml: |
    checks:
      doNotAutoAddDefaults: false
      addAllBuiltIn: false
      include: []
      exclude: []
```

#### Resource Exclusion Patterns
```yaml
# Exclude specific check
metadata:
  annotations:
    ignore-check.kube-linter.io/run-as-non-root: "Legacy system requirement"

# Exclude all checks  
metadata:
  annotations:
    kube-linter.io/ignore-all: "OLM managed deployment"
```

### Common Development Tasks

#### Adding New Validation Checks
1. **Kube-linter Integration**: Leverage existing kube-linter checks
2. **Custom Checks**: Implement using kube-linter's check framework
3. **Configuration**: Update ConfigMap schema for new options
4. **Metrics**: Add appropriate Prometheus metrics
5. **Documentation**: Update check documentation

#### Extending Resource Support
1. **Controller Watching**: Add new resource types to controller watches
2. **RBAC**: Update cluster roles for new resource permissions
3. **Validation Logic**: Extend validation pipeline for new resources
4. **Testing**: Add comprehensive test coverage

#### Performance Optimization
- **Watch Filtering**: Use field selectors and label selectors
- **Batch Processing**: Process multiple resources efficiently
- **Memory Management**: Monitor memory usage with large clusters
- **Rate Limiting**: Implement appropriate rate limiting for API calls

### Deployment Considerations

#### Security and RBAC
- **Cluster-wide Permissions**: Requires broad read access for comprehensive validation
- **Service Account**: Uses dedicated service account with minimal required permissions
- **Network Policies**: Consider network policies for metrics scraping
- **Node Affinity**: Default configuration targets infrastructure nodes

#### Scalability
- **Single Instance**: Currently designed for single-pod deployment
- **Resource Requirements**: Configure appropriate CPU/memory limits
- **Namespace Filtering**: Use `NAMESPACE_IGNORE_PATTERN` to reduce load
- **Check Configuration**: Disable unnecessary checks to improve performance

#### Monitoring and Observability
- **Prometheus Integration**: Standard `/metrics` endpoint on port 8383
- **Grafana Dashboards**: Deploy provided dashboard templates
- **Alerting**: Create alerts based on validation failure metrics
- **Logging**: Use structured logging with appropriate log levels

## Common Development Patterns

### Error Handling
```go
// Standard error handling patterns
if err != nil {
    r.Log.Error(err, "Failed to validate resource", 
        "resource", resource.Name, 
        "namespace", resource.Namespace)
    return ctrl.Result{RequeueAfter: time.Minute * 5}, err
}
```

### Configuration Loading
```go
// Use viper for configuration management
viper.SetConfigName("config")
viper.SetConfigType("yaml")
viper.AddConfigPath("/etc/dvo/")
```

### Metrics Updates
```go
// Update metrics with proper labels
validationFailures.WithLabelValues(
    resource.Name,
    resource.Namespace, 
    resource.Kind,
    checkName,
).Set(1)
```

## Useful Commands

```bash
# Development
make go-build                 # Build operator binary
make test                     # Run unit tests
make e2e-test                # Run end-to-end tests

# Deployment
make deploy                   # Deploy to cluster
make undeploy                # Remove from cluster
make bundle                   # Generate OLM bundle

# Code Quality
make lint                     # Run linting
make fmt                      # Format code
make vet                      # Run go vet

# Release Management
operator-sdk generate bundle  # Generate bundle manifests
```

## Integration Points

### kube-linter Integration
- **Check Configuration**: Uses kube-linter's configuration format
- **Custom Checks**: Can add organization-specific validation rules
- **Annotation Support**: Leverages kube-linter's ignore annotations
- **Rule Updates**: Benefits from upstream kube-linter rule improvements

### Prometheus Integration
- **Metrics Endpoint**: Exposes `/metrics` on port 8383
- **ServiceMonitor**: Compatible with Prometheus Operator
- **Alert Rules**: Can define PrometheusRule resources
- **Grafana**: Pre-built dashboard templates available

### OpenShift Integration
- **Platform Compatibility**: Tested on OpenShift clusters
- **Security Context**: Respects OpenShift security policies
- **Network Policies**: Supports OpenShift SDN networking
- **Resource Quotas**: Operates within OpenShift resource constraints

## Project Structure

```
├── api/                    # API definitions (currently empty)
├── bundle/                 # OLM bundle manifests
├── config/                 # Kustomize configuration
├── deploy/                 # Deployment manifests
│   ├── openshift/         # OpenShift-specific manifests
│   └── observability/     # Grafana dashboard templates
├── docs/                   # Documentation
├── hack/                   # Development scripts
├── internal/              # Internal packages
├── pkg/                   # Public packages
│   ├── controller/        # Controller implementations
│   ├── metrics/          # Prometheus metrics
│   └── validation/       # Validation logic
└── version/               # Version information
```

## Contributing Guidelines

- **Go Standards**: Follow standard Go conventions and idioms
- **Controller Patterns**: Use controller-runtime best practices
- **Testing**: Maintain comprehensive test coverage
- **Documentation**: Update docs for new features and changes
- **Metrics**: Ensure all validation results are properly instrumented
- **Configuration**: Support configurable behavior via ConfigMap
- **Backwards Compatibility**: Maintain API stability for existing deployments

This operator serves as a critical component for maintaining deployment quality and security compliance in Kubernetes environments.