package validations

import (
	// Used to embed yamls by kube-linter
	_ "embed"
	"fmt"
	"os"
	"strings"

	// Import checks from DVO
	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	_ "github.com/app-sre/deployment-validation-operator/pkg/validations/all"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"
	"golang.stackrox.io/kube-linter/pkg/diagnostic"
	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/run"

	// Import and initialize all check templates from kube-linter
	_ "golang.stackrox.io/kube-linter/pkg/templates/all"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/viper"
)

var engine validationEngine

type validationEngine struct {
	config           config.Config
	registry         checkregistry.CheckRegistry
	enabledChecks    []string
	registeredChecks map[string]config.Check
	metrics          map[string]*prometheus.GaugeVec
}

func NewEngine(configPath string) (*validationEngine, error) {
	ve := &validationEngine{}

	err := ve.LoadConfig(configPath)
	if err != nil {
		return nil, err
	}

	err = ve.InitRegistry()
	if err != nil {
		return nil, err
	}

	return ve, nil
}

func (ve *validationEngine) CheckRegistry() checkregistry.CheckRegistry {
	return ve.registry
}

func (ve *validationEngine) EnabledChecks() []string {
	return ve.enabledChecks
}

func (ve *validationEngine) RunForObjects(objects []client.Object, namespaceUID string) (ValidationOutcome, error) {
	lintCtx := &lintContextImpl{}
	for _, obj := range objects {
		// Only run checks against an object with no owners.  This should be
		// the object that controls the configuration
		if !utils.IsOwner(obj) {
			continue
		}
		// If controller has no replicas clear do not run any validations
		if isControllersWithNoReplicas(obj) {
			continue
		}
		lintCtx.addObjects(lintcontext.Object{K8sObject: obj})
	}
	lintCtxs := []lintcontext.LintContext{lintCtx}
	if len(lintCtxs) == 0 {
		return ObjectValidationIgnored, nil
	}
	result, err := run.Run(lintCtxs, ve.CheckRegistry(), ve.EnabledChecks())
	if err != nil {
		log.Error(err, "error running validations")
		return "", fmt.Errorf("error running validations: %v", err)
	}

	// Clear labels from past run to ensure only results from this run
	// are reflected in the metrics
	for _, o := range objects {
		req := NewRequestFromObject(o)
		req.NamespaceUID = namespaceUID
		ve.ClearMetrics(result.Reports, req.ToPromLabels())
	}
	return processResult(result, namespaceUID)
}

// Get info on config file if it exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func (ve *validationEngine) LoadConfig(path string) error {
	if !fileExists(path) {
		log.Info(fmt.Sprintf("config file %s does not exist. Use default configuration", path))
		// TODO - This hardcode will be removed when a ConfigMap is set by default in regular installation
		ve.config.Checks.DoNotAutoAddDefaults = true
		ve.config.Checks.Include = []string{
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
		}

		return nil
	}

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

type PrometheusRegistry interface {
	Register(prometheus.Collector) error
}

func (ve *validationEngine) InitRegistry() error {
	disableIncompatibleChecks(&ve.config)

	registry, err := GetKubeLinterRegistry()
	if err != nil {
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

	// TODO - Use same approach than in prometheus package
	validationMetrics := map[string]*prometheus.GaugeVec{}
	registeredChecks := map[string]config.Check{}
	for _, checkName := range enabledChecks {
		check := registry.Load(checkName)
		if check == nil {
			return fmt.Errorf("unable to create metric for check %s", checkName)
		}
		registeredChecks[check.Spec.Name] = check.Spec
		metric := newGaugeVecMetric(
			strings.ReplaceAll(check.Spec.Name, "-", "_"),
			fmt.Sprintf("Description: %s ; Remediation: %s",
				check.Spec.Description, check.Spec.Remediation),
			[]string{"namespace_uid", "namespace", "uid", "name", "kind"},
			prometheus.Labels{
				"check_description": check.Spec.Description,
				"check_remediation": check.Spec.Remediation,
			},
		)

		validationMetrics[checkName] = metric
	}

	ve.registry = registry
	ve.enabledChecks = enabledChecks
	ve.metrics = validationMetrics
	ve.registeredChecks = registeredChecks

	// keeping the validation engine glogal scoped for compatibility
	// it would be a good idea to remove the global variable use in the long run
	engine = *ve

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
	for _, vector := range ve.metrics {
		vector.Delete(labels)
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

func (ve *validationEngine) GetCheckByName(name string) (config.Check, error) {
	check, ok := ve.registeredChecks[name]
	if !ok {
		return config.Check{}, fmt.Errorf("check '%s' is not registered", name)
	}
	return check, nil
}

// UpdateConfig TODO - doc
func (ve *validationEngine) UpdateConfig(newconfig config.Config) {
	ve.config = newconfig
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
		//"non-isolated-pod",
	}
}
