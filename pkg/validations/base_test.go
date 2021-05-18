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
	loadOnce sync.Once
	ve       validationEngine
	loadErr  error
)

func createEngine() (validationEngine, error) {
	loadOnce.Do(func() {
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
	e, err := createEngine()
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
