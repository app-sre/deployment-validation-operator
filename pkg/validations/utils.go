package validations

import (
	"fmt"
	"strings"

	"github.com/app-sre/deployment-validation-operator/config"
	"golang.stackrox.io/kube-linter/pkg/builtinchecks"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	klConfig "golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"

	"github.com/prometheus/client_golang/prometheus"
)

func DeleteMetrics(labels prometheus.Labels) {
	engine.DeleteMetrics(labels)
}

func newGaugeVecMetric(
	name, help string, labelNames []string, constLabels prometheus.Labels,
) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name:        fmt.Sprintf("%s_%s", strings.ReplaceAll(config.OperatorName, "-", "_"), name),
			Help:        help,
			ConstLabels: constLabels,
		},
		labelNames,
	)
}

// GetKubeLinterRegistry sets up a Kube Linter registry with builtin default validations
func GetKubeLinterRegistry() (checkregistry.CheckRegistry, error) {
	registry := checkregistry.New()
	if err := builtinchecks.LoadInto(registry); err != nil {
		log.Error(err, "failed to load kube-linter built-in validations")
		return nil, err
	}

	return registry, nil
}

// GetAllNamesFromRegistry returns a slice with the names of all valid Kube Linter checks.
// Since any check can be configured ad hoc, we return all valid checks.
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
