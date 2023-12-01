package validations

import (
	"fmt"
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/testutils"
	"github.com/prometheus/client_golang/prometheus"
	promUtils "github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"golang.stackrox.io/kube-linter/pkg/config"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	customCheckName        = "test-minimum-replicas"
	customCheckDescription = "some description"
	customCheckRemediation = "some remediation"
	customCheckTemplate    = "minimum-replicas"
	testNamespaceUID       = "1234-6789-1011-testUID"
)

func TestGetValidChecks(t *testing.T) {
	testCases := []struct {
		name     string
		included []string
		expected []string
	}{
		{
			name:     "it returns only included checks",
			included: []string{"host-network", "host-pid"},
			expected: []string{"host-network", "host-pid"},
		},
		{
			name:     "it returns only validated checks, and not the misspelled",
			included: []string{"host-network", "host-pid", "misspelled", "wrong_format"},
			expected: []string{"host-network", "host-pid"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			mock := validationEngine{
				config: config.Config{
					Checks: config.ChecksConfig{
						DoNotAutoAddDefaults: true,
						Include:              testCase.included,
					},
				},
			}
			registry, _ := GetKubeLinterRegistry()

			// When
			test, err := mock.getValidChecks(registry)

			// Assert
			assert.Equal(t, testCase.expected, test)
			assert.NoError(t, err)
		})
	}
}

func TestRemoveCheckFromConfig(t *testing.T) {
	testCases := []struct {
		name     string
		check    string
		cfg      config.ChecksConfig
		expected config.ChecksConfig
	}{
		{
			name:  "function removes misspelled check from Include list",
			check: "unset-something",
			cfg: config.ChecksConfig{
				Include: []string{"host-network", "host-pid", "unset-something"},
			},
			expected: config.ChecksConfig{
				Include: []string{"host-network", "host-pid"},
			},
		},
		{
			name:  "function removes misspelled check from Exclude list",
			check: "unset-something",
			cfg: config.ChecksConfig{
				Exclude: []string{"host-network", "host-pid", "unset-something"},
			},
			expected: config.ChecksConfig{
				Exclude: []string{"host-network", "host-pid"},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			// Given
			mock := validationEngine{config: config.Config{Checks: testCase.cfg}}

			// When
			mock.removeCheckFromConfig(testCase.check)

			// Assert
			assert.Equal(t, mock.config.Checks, testCase.expected)
		})
	}
}

func newValidationEngine(configPath string, metrics map[string]*prometheus.GaugeVec) (*validationEngine, error) {
	config, err := loadConfig(configPath)
	if err != nil {
		return nil, err
	}

	ve := &validationEngine{
		config:  config,
		metrics: metrics,
	}
	loadErr := ve.InitRegistry()
	if loadErr != nil {
		return nil, loadErr
	}

	// checks now are preloaded, adding them after Registry init
	//ve.metrics = make(map[string]*prometheus.GaugeVec)
	for _, checkName := range ve.enabledChecks {
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

func createTestDeployment(args testutils.TemplateArgs) (*appsv1.Deployment, error) {
	d, err := testutils.CreateDeploymentFromTemplate(
		&args)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func TestUpdateConfig(t *testing.T) {
	tests := []struct {
		name          string
		initialConfig config.Config
		updatedConfig config.Config
	}{
		{
			name: "basic update test",
			initialConfig: config.Config{
				Checks: GetDefaultChecks(),
			},

			updatedConfig: config.Config{
				CustomChecks: []config.Check{
					newCustomCheck(),
				},
				Checks: config.ChecksConfig{
					Include: []string{"foo-a", "foo-b"},
					Exclude: []string{"exclude-1", "exclude-2"},
				},
			},
		},
		{
			name: "setting empty config",
			initialConfig: config.Config{
				Checks: GetDefaultChecks(),
			},

			updatedConfig: config.Config{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ve, err := newValidationEngine("", make(map[string]*prometheus.GaugeVec))
			assert.NoError(t, err, "failed to create a new validation engine")
			assert.Equal(t, tt.initialConfig, ve.config)
			ve.SetConfig(tt.updatedConfig)
			assert.Equal(t, tt.updatedConfig, ve.config)

		})
	}

}

func TestRunValidationsForObjects(t *testing.T) {
	tests := []struct {
		name                       string
		initialReplicaCount        int32
		initialExpectedMetricValue int
		updatedReplicaCount        int32
		updatedExpectedMetricValue int
		runAdditionalValidation    bool
	}{
		{
			name:                       "Custom check (min 3 replicas) is active(failing) and then it's fixed", // nolint: lll
			initialReplicaCount:        1,
			initialExpectedMetricValue: 1,
			updatedReplicaCount:        3,
			updatedExpectedMetricValue: 0,
			runAdditionalValidation:    false,
		},
		{
			name:                       "Custom check (min 3 replicas) is not active, then it's failing and it's fixed again", // nolint: lll
			initialReplicaCount:        0,
			initialExpectedMetricValue: 0,
			updatedReplicaCount:        1,
			updatedExpectedMetricValue: 1,
			runAdditionalValidation:    true,
		},
	}

	for _, testCase := range tests {
		tt := testCase
		t.Run(tt.name, func(t *testing.T) {
			metrics := make(map[string]*prometheus.GaugeVec)
			ve, err := newValidationEngine("test-resources/config-with-custom-check.yaml", metrics)
			assert.NoError(t, err, "Error creating a new validation engine")

			deployment, err := createTestDeployment(
				testutils.TemplateArgs{Replicas: int(tt.initialReplicaCount)})
			assert.NoError(t, err, "Error creating deployment from template")
			request := NewRequestFromObject(deployment)
			request.NamespaceUID = testNamespaceUID

			// run validations with "broken" (replica=1) deployment object
			_, err = ve.RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
			assert.NoError(t, err, "Error running validations")

			labels := request.ToPromLabels()
			metric, err := ve.getMetric(customCheckName).GetMetricWith(labels)
			assert.NoError(t, err, "Error getting prometheus metric")

			expectedConstLabelSubString := fmt.Sprintf(""+
				"constLabels: {check_description=\"%s\",check_remediation=\"%s\"}",
				customCheckDescription,
				customCheckRemediation,
			)
			assert.Contains(t, metric.Desc().String(), expectedConstLabelSubString,
				"Metric is missing expected constant labels! Expected:\n%s\nGot:\n%s",
				expectedConstLabelSubString,
				metric.Desc().String())
			metricValue := int(promUtils.ToFloat64(metric))
			assert.Equal(t, tt.initialExpectedMetricValue, metricValue,
				"Deployment test failed %#v: got %d want %d",
				customCheckName, metricValue, tt.initialExpectedMetricValue)

			// Problem resolved
			deployment.Spec.Replicas = &tt.updatedReplicaCount // nolint:gosec
			_, err = ve.RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
			assert.NoError(t, err, "Error running validations")
			// Metric with label combination should be successfully cleared because problem was resolved.
			// The 'GetMetricWith()' function will create a new metric with provided labels if it
			// does not exist. The default value of a metric is 0. Therefore, a value of 0 implies we
			// successfully cleared the metric label combination.
			customCheckMetricVal, err := getMetricValue(ve, customCheckName, labels)
			assert.NoError(t, err, "Error getting prometheus metric")
			assert.Equal(t, tt.updatedExpectedMetricValue, customCheckMetricVal,
				"Deployment test failed %#v: got %d want %d", "test-minimum-replicas",
				customCheckMetricVal, tt.updatedExpectedMetricValue)

			if tt.runAdditionalValidation {
				deployment.Spec.Replicas = &tt.initialReplicaCount // nolint:gosec
				_, err = ve.RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
				assert.NoError(t, err, "Error running validations")

				customCheckMetricVal, err := getMetricValue(ve, customCheckName, labels)
				assert.NoError(t, err, "Error getting prometheus metric")
				assert.Equal(t, tt.initialExpectedMetricValue, customCheckMetricVal,
					"Deployment test failed %#v: got %d want %d", "test-minimum-replicas",
					customCheckMetricVal, tt.initialExpectedMetricValue)
			}
		})
	}
}

func TestRunValidationsForObjectsAndResetMetrics(t *testing.T) {
	metrics := make(map[string]*prometheus.GaugeVec)
	ve, err := newValidationEngine("test-resources/config-with-custom-check.yaml", metrics)
	assert.NoError(t, err, "Error creating a new validation engine")

	deployment, err := createTestDeployment(
		testutils.TemplateArgs{Replicas: 1, ResourceLimits: false, ResourceRequests: false})
	assert.NoError(t, err, "Error creating deployment from template")
	request := NewRequestFromObject(deployment)
	request.NamespaceUID = testNamespaceUID

	// run validations with "broken" (replica=1) deployment object
	_, err = ve.RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
	assert.NoError(t, err, "Error running validations")

	labels := request.ToPromLabels()
	unsetCPUReqMetricVal, err := getMetricValue(ve, "unset-cpu-requirements", labels)
	assert.NoError(t, err, "Error getting prometheus metric")
	assert.Equal(t, 1, unsetCPUReqMetricVal,
		"Deployment test failed unset-cpu-requirements: got %d want %d",
		unsetCPUReqMetricVal, 1)

	customCheckMetricVal, err := getMetricValue(ve, customCheckName, labels)
	assert.NoError(t, err, "Error getting prometheus metric")
	assert.Equal(t, 1, customCheckMetricVal,
		"Deployment test failed %#v: got %d want %d",
		customCheckName, customCheckMetricVal, 1)

	ve.ResetMetrics()
	// metrics have value 0 when reset
	unsetCPUReqMetricVal, err = getMetricValue(ve, "unset-cpu-requirements", labels)
	assert.NoError(t, err)
	assert.Equal(t, 0, unsetCPUReqMetricVal,
		"Deployment test failed unset-cpu-requirements: got %d want %d",
		unsetCPUReqMetricVal, 0)

	customCheckMetricVal, err = getMetricValue(ve, customCheckName, labels)
	assert.NoError(t, err, "Error getting prometheus metric")
	assert.Equal(t, 0, customCheckMetricVal,
		"Deployment test failed %#v: got %d want %d",
		customCheckName, customCheckMetricVal, 0)
}

func getMetricValue(v *validationEngine, checkName string, labels prometheus.Labels) (int, error) {
	gauge := v.getMetric(checkName)
	if gauge == nil {
		return 0, fmt.Errorf("gauge vector %s not found ", checkName)
	}
	metric, err := gauge.GetMetricWith(labels)
	if err != nil {
		return 0, err
	}
	return int(promUtils.ToFloat64(metric)), nil
}

func TestExcludedChecksAreNotActive(t *testing.T) {
	ve, err := newValidationEngine("test-resources/config-with-some-excluded-checks.yaml",
		make(map[string]*prometheus.GaugeVec))
	assert.NoError(t, err, "Error initializing engine")

	deployment, err := createTestDeployment(
		testutils.TemplateArgs{Replicas: 1, ResourceLimits: false, ResourceRequests: false})
	assert.NoError(t, err, "Error creating deployment from template")
	request := NewRequestFromObject(deployment)
	request.NamespaceUID = testNamespaceUID

	_, err = ve.RunValidationsForObjects([]client.Object{deployment}, request.NamespaceUID)
	assert.NoError(t, err)

	// following two checks are excluded in the corresponding config file
	labels := request.ToPromLabels()
	_, err = getMetricValue(ve, "unset-cpu-requirements", labels)
	assert.Error(t, err, "gauge vector unset-cpu-requirements not found")

	_, err = getMetricValue(ve, "unset-memory-requirements", labels)
	assert.Error(t, err, "gauge vector unset-memory-requirements not found")
}
