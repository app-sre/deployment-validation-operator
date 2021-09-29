package validations

import (
	// Used to embed yamls by kube-linter
	_ "embed"
	"fmt"
	"strings"

	// Import checks from DVO
	_ "github.com/app-sre/deployment-validation-operator/pkg/validations/all"

	"golang.stackrox.io/kube-linter/pkg/builtinchecks"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"
	"golang.stackrox.io/kube-linter/pkg/diagnostic"

	// Import and initialize all check templates from kube-linter
	_ "golang.stackrox.io/kube-linter/pkg/templates/all"

	"sigs.k8s.io/controller-runtime/pkg/metrics"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/viper"
)

var engine validationEngine

type validationEngine struct {
	config        config.Config
	registry      checkregistry.CheckRegistry
	enabledChecks []string
	metrics       map[string]*prometheus.GaugeVec
}

func (ve *validationEngine) CheckRegistry() checkregistry.CheckRegistry {
	return ve.registry
}

func (ve *validationEngine) EnabledChecks() []string {
	return ve.enabledChecks
}

func (ve *validationEngine) LoadConfig(path string) error {
	v := viper.New()

	// Load Configuration
	config, err := config.Load(v, path)
	if err != nil {
		log.Error(err, "failed to load config")
		return err
	}
	ve.config = config

	return nil
}

func (ve *validationEngine) InitRegistry() error {
	disableIncompatibleChecks(&ve.config)

	registry := checkregistry.New()
	if err := builtinchecks.LoadInto(registry); err != nil {
		log.Error(err, "failed to load built-in validations")
		return err
	}

	if err := configresolver.LoadCustomChecksInto(&ve.config, registry); err != nil {
		log.Error(err, "failed to load custom checks")
		return err
	}

	enabledChecks, err := configresolver.GetEnabledChecksAndValidate(&ve.config, registry)
	if err != nil {
		log.Error(err, "error finding enabled validations")
		return err
	}

	validationMetrics := map[string]*prometheus.GaugeVec{}
	for _, checkName := range enabledChecks {
		check := registry.Load(checkName)
		if check == nil {
			return fmt.Errorf("unable to create metric for check %s", checkName)
		}
		metric := newGaugeVecMetric(strings.ReplaceAll(check.Spec.Name, "-", "_"),
			// Should this be the Remediation text or the description?
			// For now go with Description
			check.Spec.Description,
			[]string{"namespace", "name", "kind"})
		metrics.Registry.MustRegister(metric)
		validationMetrics[checkName] = metric
	}

	ve.registry = registry
	ve.enabledChecks = enabledChecks
	ve.metrics = validationMetrics

	return nil
}

func (ve *validationEngine) GetMetric(name string) *prometheus.GaugeVec {
	m, ok := ve.metrics[name]
	if !ok {
		return nil
	}
	return m
}

func (ve *validationEngine) DeleteMetrics(labels prometheus.Labels) {
	for _, v := range ve.metrics {
		v.Delete(labels)
	}
}

func (ve *validationEngine) ClearMetrics(reports []diagnostic.WithContext, labels prometheus.Labels) {
	// Create a list of validation names for use to delete the labels from any
	// metric which isn't in the report but for which there is a metric
	reportValidationNames := map[string]struct{}{}
	for _, report := range reports {
		reportValidationNames[report.Check] = struct{}{}
	}

	// Delete the labels for validations that aren't in the list of reports
	for metricValidationName := range ve.metrics {
		if _, ok := reportValidationNames[metricValidationName]; !ok {
			ve.metrics[metricValidationName].Delete(labels)
		}
	}
}

// InitializeValidationEngine will initialize the validation engine from scratch.
// If an existing engine exists, it will not be replaced with the new one unless all
// initialization steps succeed.
func InitializeValidationEngine(path string) error {
	ve := validationEngine{}

	err := ve.LoadConfig(path)
	if err == nil {
		err = ve.InitRegistry()
	}

	// Only replace the exisiting engine if no errors occurred
	if err == nil {
		engine = ve
	}

	return err
}

// disableIncompatibleChecks will forcibly update a kube-linter config
// to disable checks that are incompatible with DVO.
// the same check name may end up in the exclude list multiple times as a result of this; this is OK.
func disableIncompatibleChecks(c *config.Config) {
	c.Checks.Exclude = append(c.Checks.Exclude, getIncompatibleChecks()...)
}

// getIncompatibleChecks returns an array of kube-linter check names that are incompatible with DVO
// these checks involve kube-linter comparing properties from multiple kubernetes objects at once.
// (e.g. "non-existent-service-account" checks that all serviceaccounts referenced by deployment objects
// exist as serviceaccount objects).
// DVO currently only performs a check against a single kubernetes object at a time,
// so these checks that compare multiple objects together will always fail.
func getIncompatibleChecks() []string {
	return []string{
		"dangling-service",
		"non-existent-service-account",
		"non-isolated-pod",
	}
}
