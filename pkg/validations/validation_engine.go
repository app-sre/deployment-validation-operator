package validations

import (
	// Used to embed yamls by kube-linter

	_ "embed" // nolint:golint
	"fmt"
	"os"
	"reflect"
	"regexp"

	// Import checks from DVO

	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	_ "github.com/app-sre/deployment-validation-operator/pkg/validations/all" // nolint:golint
	"sigs.k8s.io/controller-runtime/pkg/client"

	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"
	"golang.stackrox.io/kube-linter/pkg/diagnostic"
	"golang.stackrox.io/kube-linter/pkg/lintcontext"
	"golang.stackrox.io/kube-linter/pkg/run"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	// Import and initialize all check templates from kube-linter
	_ "golang.stackrox.io/kube-linter/pkg/templates/all" // nolint:golint

	"github.com/prometheus/client_golang/prometheus"

	"github.com/spf13/viper"
)

var log = logf.Log.WithName("validations")

type ValidationOutcome string

var (
	ObjectNeedsImprovement  ValidationOutcome = "object needs improvement"
	ObjectValid             ValidationOutcome = "object valid"
	ObjectsValid            ValidationOutcome = "objects are valid"
	ObjectValidationIgnored ValidationOutcome = "object validation ignored"
)

type Interface interface {
	InitRegistry() error
	DeleteMetrics(labels prometheus.Labels)
	ResetMetrics()
	UpdateConfig(cfg config.Config)
	RunValidationsForObjects(objects []client.Object, namespaceUID string) (ValidationOutcome, error)
}

type validationEngine struct {
	config           config.Config
	registry         checkregistry.CheckRegistry
	enabledChecks    []string
	registeredChecks map[string]config.Check
	metrics          map[string]*prometheus.GaugeVec
}

// NewValidationEngine creates a new ValidationEngine instance with the provided configuration path, a watcher, and metrics. // nolint: lll
// It initializes a ValidationEngine with the provided watcher for configmap changes and a set of preloaded metrics. // nolint: lll
// The engine's configuration is loaded from the specified configuration path, and its check registry is initialized. // nolint: lll
// InitRegistry sets this instance in the package scope in engine variable.
//
// Parameters:
//   - configPath: The path to the configuration file for the ValidationEngine.
//   - cmw: A configmap.Watcher for monitoring changes to configmaps.
//   - metrics: A map of preloaded Prometheus GaugeVec metrics.
//
// Returns:
//   - An error if there's an issue loading the configuration or initializing the check registry.
func NewValidationEngine(configPath string, metrics map[string]*prometheus.GaugeVec) (Interface, error) {
	cfg, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	ve := &validationEngine{
		metrics: metrics,
		config:  cfg,
	}

	err = ve.InitRegistry()
	if err != nil {
		return nil, err
	}

	return ve, nil
}

// Get info on config file if it exists
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func loadConfig(path string) (config.Config, error) {
	if !fileExists(path) {
		log.Info(fmt.Sprintf("config file %s does not exist. Use default configuration", path))
		return config.Config{
			Checks: GetDefaultChecks(),
		}, nil
	}

	v := viper.New()
	// Load Configuration
	return config.Load(v, path)
}

// RunValidationsForObjects runs validation for the group of related objects
func (ve *validationEngine) RunValidationsForObjects(objects []client.Object,
	namespaceUID string) (ValidationOutcome, error) {
	lintCtx := &lintContextImpl{}
	for _, obj := range objects {
		// Only run checks against an object with no owners.  This should be
		// the object that controls the configuration
		if !utils.IsOwner(obj) {
			continue
		}
		// If controller has no replicas clear do not run any validations
		if ve.isControllersWithNoReplicas(obj) {
			continue
		}
		lintCtx.addObjects(lintcontext.Object{K8sObject: obj})
	}
	lintCtxs := []lintcontext.LintContext{lintCtx}
	if len(lintCtxs) == 0 {
		return ObjectValidationIgnored, nil
	}
	result, err := run.Run(lintCtxs, ve.registry, ve.enabledChecks)
	if err != nil {
		log.Error(err, "error running validations")
		return "", fmt.Errorf("error running validations: %v", err)
	}

	// Clear labels from past run to ensure only results from this run
	// are reflected in the metrics
	for _, o := range objects {
		req := NewRequestFromObject(o)
		req.NamespaceUID = namespaceUID
		ve.clearMetrics(result.Reports, req.ToPromLabels())
	}
	return ve.processResult(result, namespaceUID)
}

// isControllersWithNoReplicas checks if the provided object has no replicas
func (ve *validationEngine) isControllersWithNoReplicas(obj client.Object) bool {
	objValue := reflect.Indirect(reflect.ValueOf(obj))
	spec := objValue.FieldByName("Spec")
	if spec.IsValid() {
		replicas := spec.FieldByName("Replicas")
		if replicas.IsValid() {
			numReplicas, ok := replicas.Interface().(*int32)

			// clear labels if we fail to get a value for numReplicas, or if value is <= 0
			if !ok || numReplicas == nil || *numReplicas <= 0 {
				req := NewRequestFromObject(obj)
				ve.DeleteMetrics(req.ToPromLabels())
				return true
			}
		}
	}
	return false
}

func (ve *validationEngine) processResult(result run.Result, namespaceUID string) (ValidationOutcome, error) {
	outcome := ObjectValid
	for _, report := range result.Reports {
		check, err := ve.getCheckByName(report.Check)
		if err != nil {
			log.Error(err, fmt.Sprintf("Failed to get check '%s' by name", report.Check))
			return "", fmt.Errorf("error running validations: %v", err)
		}
		obj := report.Object.K8sObject
		logger := log.WithValues(
			"request.namespace", obj.GetNamespace(),
			"request.name", obj.GetName(),
			"kind", obj.GetObjectKind().GroupVersionKind().Kind,
			"validation", report.Check,
			"check_description", check.Description,
			"check_remediation", report.Remediation,
			"check_failure_reason", report.Diagnostic.Message,
		)
		metric := ve.getMetric(report.Check)
		if metric == nil {
			log.Error(nil, "no metric found for validation", report.Check)
		} else {
			req := NewRequestFromObject(obj)
			req.NamespaceUID = namespaceUID
			metric.With(req.ToPromLabels()).Set(1)
			logger.Info(report.Remediation)
			outcome = ObjectNeedsImprovement
		}
	}
	return outcome, nil
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

	enabledChecks, err := ve.getValidChecks(registry)
	if err != nil {
		log.Error(err, "error finding enabled validations")
		return err
	}

	registeredChecks := map[string]config.Check{}
	for _, checkName := range enabledChecks {
		check := registry.Load(checkName)
		if check == nil {
			return fmt.Errorf("unable to create metric for check %s", checkName)
		}
		registeredChecks[check.Spec.Name] = check.Spec
	}

	ve.registry = registry
	ve.enabledChecks = enabledChecks
	ve.registeredChecks = registeredChecks

	return nil
}

func (ve *validationEngine) getMetric(name string) *prometheus.GaugeVec {
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

func (ve *validationEngine) clearMetrics(reports []diagnostic.WithContext, labels prometheus.Labels) {
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

func (ve *validationEngine) ResetMetrics() {
	for _, metric := range ve.metrics {
		metric.Reset()
	}
}

func (ve *validationEngine) getCheckByName(name string) (config.Check, error) {
	check, ok := ve.registeredChecks[name]
	if !ok {
		return config.Check{}, fmt.Errorf("check '%s' is not registered", name)
	}
	return check, nil
}

// getValidChecks function fetches and validates the list of enabled checks from the ValidationEngine's
// configuration. It uses the provided check registry to validate the enabled checks against available checks.
// If any checks are found to be invalid (not present in the check registry), they are removed from the configuration.
// The function then recursively calls itself to fetch a new list of valid checks without the invalid ones.
func (ve *validationEngine) getValidChecks(registry checkregistry.CheckRegistry) ([]string, error) {
	enabledChecks, err := configresolver.GetEnabledChecksAndValidate(&ve.config, registry)
	if err != nil {
		// error format from configresolver:
		// "enabled checks validation error: [check \"check name\" not found, ...]"}
		re := regexp.MustCompile(`check \"([^,]*)\" not found`)
		if matches := re.FindAllStringSubmatch(err.Error(), -1); matches != nil {
			for i := range matches {
				log.Info("entered ConfigMap check was not validated and is ignored",
					"validation name", matches[i][1],
				)
				ve.removeCheckFromConfig(matches[i][1])
			}
			return ve.getValidChecks(registry)
		}
		return []string{}, err
	}

	return enabledChecks, nil
}

func (ve *validationEngine) UpdateConfig(cfg config.Config) {
	ve.config = cfg
}

// removeCheckFromConfig function searches for the given check name in both the "Include" and "Exclude" lists
// of checks in the ValidationEngine's configuration. If the check is found in either list, it is removed by updating
// the respective list.
func (ve *validationEngine) removeCheckFromConfig(check string) {
	include := ve.config.Checks.Include
	for i := 0; i < len(include); i++ {
		if include[i] == check {
			ve.config.Checks.Include = append(include[:i], include[i+1:]...)
			return
		}
	}

	exclude := ve.config.Checks.Exclude
	for i := 0; i < len(exclude); i++ {
		if exclude[i] == check {
			ve.config.Checks.Exclude = append(exclude[:i], exclude[i+1:]...)
			return
		}
	}
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
