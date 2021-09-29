package validations

import (
	"sync"
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/testutils"

	prom_tu "github.com/prometheus/client_golang/prometheus/testutil"

	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/types"

	"golang.stackrox.io/kube-linter/pkg/config"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
	checkName = "test-minimum-replicas"
)

var (
	loadOnceWithCustomCheck sync.Once
	loadOnceWithAllChecks 	sync.Once
	ve       				validationEngine
	loadErr  				error
)

func createEngineWithCustomCheck() (validationEngine, error) {
	loadOnceWithCustomCheck.Do(func() {
		ve = validationEngine{
			config: config.Config{
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
			},
		}
		loadErr = ve.InitRegistry()
	})
	if loadErr != nil {
		return validationEngine{}, loadErr
	}
	return ve, nil
}

func createEngineWithAllChecks() (validationEngine, error) {
	loadOnceWithAllChecks.Do(func() {
		ve = validationEngine{
			config: config.Config{
				CustomChecks: []config.Check{},
				Checks: config.ChecksConfig{
					AddAllBuiltIn:        true,
					DoNotAutoAddDefaults: false,
				},
			},
		}
		loadErr = ve.InitRegistry()
	})
	if loadErr != nil {
		return validationEngine{}, loadErr
	}
	return ve, nil
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
	e, err := createEngineWithCustomCheck()
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
	e, err := createEngineWithAllChecks()
	if err != nil {
		t.Errorf("Error creating validation engine %v", err)
	}

	enabledChecks := e.EnabledChecks()
	if len(enabledChecks) <= 10 {
		t.Errorf("Expected more than 10 checks to be enabled, but got '%v' from '%v'", len(enabledChecks), enabledChecks)
	}

	badChecks := getIncompatibleChecks()
	for _, badCheck := range badChecks {
		if stringInSlice(badCheck, enabledChecks) {
			t.Errorf("Expected incompatible kube-linter check '%v' to not be enabled, but it was in the enabled list '%v'", badCheck, enabledChecks)
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
