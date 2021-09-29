package validations

import (
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/testutils"

	prom_tu "github.com/prometheus/client_golang/prometheus/testutil"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/types"

	"golang.stackrox.io/kube-linter/pkg/builtinchecks"
	"golang.stackrox.io/kube-linter/pkg/checkregistry"
	"golang.stackrox.io/kube-linter/pkg/config"
	"golang.stackrox.io/kube-linter/pkg/configresolver"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	checkName = "test-minimum-replicas"
)

func newEngine(c config.Config) (validationEngine, error) {
	ve := validationEngine{
		config: c,
	}
	loadErr := ve.InitRegistry()
	if loadErr != nil {
		return validationEngine{}, loadErr
	}
	return ve, nil
}

func newEngineConfigWithCustomCheck() config.Config {
	return config.Config{
		CustomChecks: []config.Check{
			{
				Name:     checkName,
				Template: "minimum-replicas",
				Scope: &config.ObjectKindsDesc{
					ObjectKinds: []string{"DeploymentLike"},
				},
				Params: map[string]interface{}{"minReplicas": 3},
			},
		},
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

func TestRunValidationsIssueCorrection(t *testing.T) {
	e, err := newEngine(newEngineConfigWithCustomCheck())
	if err != nil {
		t.Errorf("Error creating validation engine %v", err)
	}
	engine = e

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}

	replicaCnt := int32(1)
	deployment, err := createTestDeployment(replicaCnt)
	if err != nil {
		t.Errorf("Error creating deployment from template %v", err)
	}

	RunValidations(request, deployment, testutils.ObjectKind(deployment), false)

	labels := getPromLabels(request.Name, request.Namespace, "Deployment")
	metric, err := engine.GetMetric(checkName).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 1 {
		t.Errorf("Deployment test failed %#v: got %d want %d", checkName, metricValue, 1)
	}

	// Problem resolved
	replicaCnt = int32(3)
	deployment.Spec.Replicas = &replicaCnt
	RunValidations(request, deployment, testutils.ObjectKind(deployment), false)

	metric, err = engine.GetMetric(checkName).GetMetricWith(labels)
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("Deployment test failed %#v: got %d want %d", checkName, metricValue, 0)
	}
}

func TestIncompatibleChecksAreDisabled(t *testing.T) {
	e, err := newEngine(newEngineConfigWithAllChecks())
	if err != nil {
		t.Errorf("Error creating validation engine %v", err)
	}

	badChecks := getIncompatibleChecks()
	allKubeLinterChecks, err := getAllBuiltInKubeLinterChecks()
	if err != nil {
		t.Fatalf("Got unexpected error while getting all checks built-into kube-linter: %v", err)
	}
	expectedNumChecks := len(allKubeLinterChecks) - len(badChecks)

	enabledChecks := e.EnabledChecks()
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
		log.Error(err, "failed to load built-in validations")
		return nil, err
	}

	enabledChecks, err := configresolver.GetEnabledChecksAndValidate(&ve.config, registry)
	if err != nil {
		log.Error(err, "error finding enabled validations")
		return nil, err
	}

	return enabledChecks, nil
}
