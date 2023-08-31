package validations

import (
	"fmt"
	"strings"
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/testutils"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/prometheus/client_golang/prometheus"
	prom_tu "github.com/prometheus/client_golang/prometheus/testutil"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/stretchr/testify/assert"
	"golang.stackrox.io/kube-linter/pkg/builtinchecks"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	customCheckName        = "test-minimum-replicas"
	customCheckDescription = "some description"
	customCheckRemediation = "some remediation"
	customCheckTemplate    = "minimum-replicas"
)

func newEngine(c config.Config) (validationEngine, error) {
	ve := validationEngine{
		config: c,
	}
	loadErr := ve.InitRegistry()
	if loadErr != nil {
		return validationEngine{}, loadErr
	}
	// checks now are preloaded, adding them after Registry init
	ve.metrics = make(map[string]*prometheus.GaugeVec)
	for _, checkName := range ve.EnabledChecks() {
		check := ve.registeredChecks[checkName]
		ve.metrics[checkName] = newGaugeVecMetric(check)
	}
	return ve, nil
}

func newCustomCheck() config.Check {
	return config.Check{
		Name:        customCheckName,
		Description: customCheckDescription,
		Remediation: customCheckRemediation,
		Template:    customCheckTemplate,
		Scope: &config.ObjectKindsDesc{
			ObjectKinds: []string{"DeploymentLike"},
		},
		Params: map[string]interface{}{"minReplicas": 3},
	}
}

func newEngineConfigWithCustomCheck(customCheck []config.Check) config.Config {

	// Create custom config with custom check array
	return config.Config{
		CustomChecks: customCheck,
		Checks: config.ChecksConfig{
			AddAllBuiltIn:        false,
			DoNotAutoAddDefaults: true,
		},
	}
}

func newEngineConfigWithAllChecks() config.Config {
	return config.Config{
		CustomChecks: []config.Check{},
		Checks: config.ChecksConfig{
			AddAllBuiltIn:        true,
			DoNotAutoAddDefaults: false,
		},
	}
}

func createTestDeployment(replicas int32) (*appsv1.Deployment, error) {
	d, err := testutils.CreateDeploymentFromTemplate(
		testutils.NewTemplateArgs())
	if err != nil {
		return nil, err
	}
	d.Spec.Replicas = &replicas
	return &d, nil
}

func initializeEngine(customCheck ...config.Check) error {
	// Check if custom check has been set
	if len(customCheck) > 0 {
		// Initialize engine with custom check
		e, err := newEngine(newEngineConfigWithCustomCheck(customCheck))
		if err != nil {
			return err
		}
		engine = e
	} else {
		// Initialize engine for all checks
		e, err := newEngine(newEngineConfigWithAllChecks())
		if err != nil {
			return err
		}
		engine = e
	}
	return nil
}

func TestRunValidationsIssueCorrection(t *testing.T) {

	customCheck := newCustomCheck()

	// Initialize engine
	err := initializeEngine(customCheck)
	if err != nil {
		t.Errorf("Error initializing engine %v", err)
	}

	replicaCnt := int32(1)
	deployment, err := createTestDeployment(replicaCnt)
	request := NewRequestFromObject(deployment)
	if err != nil {
		t.Errorf("Error creating deployment from template %v", err)
	}

	_, err = RunValidations(request, deployment)
	if err != nil {
		t.Errorf("Error running validations: %v", err)
	}

	labels := request.ToPromLabels()
	metric, err := engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	expectedConstLabelSubString := fmt.Sprintf(""+
		"constLabels: {check_description=\"%s\",check_remediation=\"%s\"}",
		customCheck.Description,
		customCheck.Remediation,
	)
	if !strings.Contains(metric.Desc().String(), expectedConstLabelSubString) {
		t.Errorf("Metric is missing expected constant labels! Expected:\n%s\nGot:\n%s",
			expectedConstLabelSubString,
			metric.Desc().String())
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 1 {
		t.Errorf("Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 1)
	}

	// Problem resolved
	replicaCnt = int32(3)
	deployment.Spec.Replicas = &replicaCnt
	_, err = RunValidations(request, deployment)
	if err != nil {
		t.Errorf("Error running validations: %v", err)
	}
	// Metric with label combination should be successfully cleared because problem was resolved.
	// The 'GetMetricWith()' function will create a new metric with provided labels if it
	// does not exist. The default value of a metric is 0. Therefore, a value of 0 implies we
	// successfully cleared the metric label combination.
	metric, err = engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 0)
	}
}

func TestRunValidationsForObjectsIssueCorrection(t *testing.T) {
	customCheck := newCustomCheck()
	// Initialize engine
	err := initializeEngine(customCheck)
	assert.NoError(t, err, "Error initializing engine")

	replicaCnt := int32(1)
	deployment, err := createTestDeployment(replicaCnt)
	assert.NoError(t, err, "Error creating deployment from template")
	request := NewRequestFromObject(deployment)
	request.NamespaceUID = "1234-6789-1011-testUID"

	// run validations with "broken" (replica=1) deployment object
	_, err = RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
	assert.NoError(t, err, "Error running validations")

	labels := request.ToPromLabels()
	metric, err := engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	assert.NoError(t, err, "Error getting prometheus metric")

	expectedConstLabelSubString := fmt.Sprintf(""+
		"constLabels: {check_description=\"%s\",check_remediation=\"%s\"}",
		customCheck.Description,
		customCheck.Remediation,
	)
	assert.Contains(t, metric.Desc().String(), expectedConstLabelSubString,
		"Metric is missing expected constant labels! Expected:\n%s\nGot:\n%s",
		expectedConstLabelSubString,
		metric.Desc().String())
	metricValue := int(prom_tu.ToFloat64(metric))
	assert.Equal(t, 1, metricValue, "Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 1)

	// Problem resolved
	replicaCnt = int32(3)
	deployment.Spec.Replicas = &replicaCnt
	_, err = RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
	assert.NoError(t, err, "Error running validations")
	// Metric with label combination should be successfully cleared because problem was resolved.
	// The 'GetMetricWith()' function will create a new metric with provided labels if it
	// does not exist. The default value of a metric is 0. Therefore, a value of 0 implies we
	// successfully cleared the metric label combination.
	metric, err = engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	assert.NoError(t, err, "Error getting prometheus metric")

	metricValue = int(prom_tu.ToFloat64(metric))
	assert.Equal(t, 0, metricValue, "Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 0)
}

func TestIncompatibleChecksAreDisabled(t *testing.T) {

	// Initialize engine
	err := initializeEngine()
	if err != nil {
		t.Errorf("Error initializing engine: %v", err)
	}

	badChecks := getIncompatibleChecks()
	allKubeLinterChecks, err := getAllBuiltInKubeLinterChecks()
	if err != nil {
		t.Fatalf("Got unexpected error while getting all checks built-into kube-linter: %v", err)
	}
	expectedNumChecks := (len(allKubeLinterChecks) - len(badChecks))

	enabledChecks := engine.EnabledChecks()
	if len(enabledChecks) != expectedNumChecks {
		t.Errorf("Expected exactly %v checks to be enabled, but got '%v' checks from list '%v'",
			expectedNumChecks, len(enabledChecks), enabledChecks)
	}

	for _, badCheck := range badChecks {
		if stringInSlice(badCheck, enabledChecks) {
			t.Errorf("Expected incompatible kube-linter check '%v' to not be enabled, "+
				"but it was in the enabled list '%v'",
				badCheck, enabledChecks)
		}
	}
}

// Test to check if a resource has 0 replicas it clears metrics
func TestValidateZeroReplicas(t *testing.T) {

	customCheck := newCustomCheck()

	// Initialize Engine
	err := initializeEngine(customCheck)
	if err != nil {
		t.Errorf("Error initializing engine %v", err)
	}

	// Setup test deployment file with 0 replicas
	replicaCnt := int32(0)
	deployment, err := createTestDeployment(replicaCnt)
	request := NewRequestFromObject(deployment)
	if err != nil {
		t.Errorf("Error creating deployment from template %v", err)
	}

	// Run validations against test environment
	_, err = RunValidations(request, deployment)
	if err != nil {
		t.Errorf("Error running validations: %v", err)
	}
	// Acquire labels generated from validations
	labels := request.ToPromLabels()

	// The 'GetMetricWith()' function will create a new metric with provided labels if it
	// does not exist. The default value of a metric is 0. Therefore, a value of 0 implies we
	// successfully cleared the metric label combination.
	metric, err := engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	// If metrics are cleared then the value will be == 0
	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 0)
	}

	// Check using a replica count above 0
	replicaCnt = int32(1)
	deployment.Spec.Replicas = &replicaCnt
	_, err = RunValidations(request, deployment)
	if err != nil {
		t.Errorf("Error running validations: %v", err)
	}
	metric, err = engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	// If metrics exist then value will be non 0
	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue == 0 {
		t.Errorf("Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 1)
	}

	// Check to see metrics clear by setting replicas to 0
	replicaCnt = int32(0)
	deployment.Spec.Replicas = &replicaCnt
	_, err = RunValidations(request, deployment)
	if err != nil {
		t.Errorf("Error running validations: %v", err)
	}
	metric, err = engine.GetMetric(customCheck.Name).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	// If metrics are cleared then the value will be == 0
	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("Deployment test failed %#v: got %d want %d", customCheck.Name, metricValue, 0)
	}

}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// getAllBuiltInKubeLinterChecks returns every check built-into kube-linter (including checks that DVO disables)
func getAllBuiltInKubeLinterChecks() ([]string, error) {
	ve := validationEngine{
		config: newEngineConfigWithAllChecks(),
	}
	registry := checkregistry.New()
	if err := builtinchecks.LoadInto(registry); err != nil {
		return nil, fmt.Errorf("failed to load built-in validations: %s", err.Error())
	}

	enabledChecks, err := configresolver.GetEnabledChecksAndValidate(&ve.config, registry)
	if err != nil {
		return nil, fmt.Errorf("error finding enabled validations: %s", err.Error())
	}

	return enabledChecks, nil
}

func TestRequest(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		Object         client.Object
		NamespaceUID   string
		ExpectedLabels prometheus.Labels
	}{
		"without namespace UID": {
			Object: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
					UID:       "abcdefgh",
				},
			},
			ExpectedLabels: prometheus.Labels{
				"kind":          "Deployment",
				"name":          "test",
				"namespace":     "test-namespace",
				"namespace_uid": "",
				"uid":           "abcdefgh",
			},
		},
		"with namespace UID": {
			Object: &appsv1.Deployment{
				TypeMeta: metav1.TypeMeta{
					Kind: "Deployment",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test",
					Namespace: "test-namespace",
					UID:       "abcdefgh",
				},
			},
			NamespaceUID: "12345678",
			ExpectedLabels: prometheus.Labels{
				"kind":          "Deployment",
				"name":          "test",
				"namespace":     "test-namespace",
				"namespace_uid": "12345678",
				"uid":           "abcdefgh",
			},
		},
	} {
		tc := tc

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			req := NewRequestFromObject(tc.Object)
			req.NamespaceUID = tc.NamespaceUID

			assert.Equal(t, tc.ExpectedLabels, req.ToPromLabels())
		})
	}
}
