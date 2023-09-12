package validations

import (
	"fmt"
	"strings"

	"golang.stackrox.io/kube-linter/pkg/builtinchecks"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	klConfig "golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"

	"github.com/app-sre/deployment-validation-operator/config"
	"github.com/prometheus/client_golang/prometheus"
)

func DeleteMetrics(labels prometheus.Labels) {
	engine.DeleteMetrics(labels)
}

// GetKubeLinterRegistry returns a CheckRegistry containing kube-linter built-in validations.
// It initializes a new CheckRegistry, loads the built-in validations into the registry,
// and returns the resulting registry if successful.
//
// Returns:
//   - A CheckRegistry containing kube-linter built-in validations if successful.
//   - An error if the built-in validations fail to load into the registry.
func GetKubeLinterRegistry() (checkregistry.CheckRegistry, error) {
	registry := checkregistry.New()
	if err := builtinchecks.LoadInto(registry); err != nil {
		log.Error(err, "failed to load kube-linter built-in validations")
		return nil, err
	}

	return registry, nil
}

// GetAllNamesFromRegistry retrieves the names of all enabled checks from the provided CheckRegistry.
// It fetches the names of checks that are enabled based on a specified configuration, excluding incompatible ones.
//
// Parameters:
//   - reg: A CheckRegistry containing predefined checks and their specifications.
//
// Returns:
//   - A slice of strings containing the names of all enabled checks if successful.
//   - An error if there's an issue while fetching the enabled check names or validating the configuration.
func GetAllNamesFromRegistry(reg checkregistry.CheckRegistry) ([]string, error) {
	// Get all checks except for incompatible ones
	cfg := klConfig.Config{
		Checks: klConfig.ChecksConfig{
			AddAllBuiltIn: true,
		},
	}
	disableIncompatibleChecks(&cfg)

	checks, err := configresolver.GetEnabledChecksAndValidate(&cfg, reg)
	if err != nil {
		log.Error(err, "error getting enabled validations")
		return nil, err
	}

	return checks, nil
}

func newGaugeVecMetric(check klConfig.Check) *prometheus.GaugeVec {
	metricName := strings.ReplaceAll(fmt.Sprintf("%s_%s", config.OperatorName, check.Name), "-", "_")

	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: metricName,
			Help: fmt.Sprintf(
				"Description: %s ; Remediation: %s", check.Description, check.Remediation,
			),
			ConstLabels: prometheus.Labels{
				"check_description": check.Description,
				"check_remediation": check.Remediation,
			},
		}, []string{"namespace_uid", "namespace", "uid", "name", "kind"})
}

// UpdateConfig provides an access to setup new configuration for the generic reconciler
func UpdateConfig(cfg klConfig.Config) {
	engine.config = cfg
}

// InitRegistry forces Validation Engine to initialize a new registry
func InitRegistry() error {
	return engine.InitRegistry()
}

// ResetMetrics resets all the metrics registered in the Validation Engine
func ResetMetrics() {
	for _, metric := range engine.metrics {
		metric.Reset()
	}
}

// GetDefaultChecks provides a default set of checks usable in case there is no custom ConfigMap
func GetDefaultChecks() klConfig.ChecksConfig {
	return klConfig.ChecksConfig{
		DoNotAutoAddDefaults: true,
		Include: []string{
			"host-ipc",
			"host-network",
			"host-pid",
			"non-isolated-pod",
			"pdb-max-unavailable",
			"pdb-min-available",
			"privilege-escalation-container",
			"privileged-container",
			"run-as-non-root",
			"unsafe-sysctls",
			"unset-cpu-requirements",
			"unset-memory-requirements",
		},
	}
}
