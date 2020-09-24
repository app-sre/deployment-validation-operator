package validations

import (
	"testing"

	"github.com/app-sre/deployment-validation-operator/pkg/testutils"
	dv_tu "github.com/app-sre/deployment-validation-operator/pkg/testutils"
	"github.com/prometheus/client_golang/prometheus"
	prom_tu "github.com/prometheus/client_golang/prometheus/testutil"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestReplicaValidationAppliesTo(t *testing.T) {
	tests := []struct {
		kind     string
		expected bool
	}{
		{"Deployment", true},
		{"ReplicaSet", true},
		{"Pod", false},
		{"StatefulSet", false},
	}

	rv := newReplicaValidation()

	for _, test := range tests {
		if _, ok := rv.AppliesTo()[test.kind]; ok != test.expected {
			t.Errorf("%s kind: got %t want %t", test.kind, ok, test.expected)
		}
	}
}

func TestDeploymentReplicaValidation(t *testing.T) {
	tests := []struct {
		replicas  int
		isDeleted bool
		expected  int
	}{
		{3, false, 0},
		{1, false, 1},
		{1, true, 0}, // a bit weird, but this is what prom_tu.ToFloat64 returns
	}

	rv := newReplicaValidation()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}

	for _, test := range tests {
		args := dv_tu.NewTemplateArgs()
		args.Replicas = test.replicas
		deployment, err := dv_tu.CreateDeploymentFromTemplate(args)
		if err != nil {
			t.Errorf("Error creating deployment from template %v", err)
		}

		rv.Validate(request, &deployment, testutils.ObjectKind(&deployment), test.isDeleted)
		metric, err := rv.metric.GetMetricWith(
			prometheus.Labels{"name": "foo", "namespace": "bar", "kind": "Deployment"})
		if err != nil {
			t.Errorf("Error getting prometheus metric: %v", err)
		}

		if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != test.expected {
			t.Errorf("Deployment test failed %#v: got %d want %d", test, metricValue, test.expected)
		}
	}

}

func TestReplicaSetReplicaValidation(t *testing.T) {
	rv := newReplicaValidation()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}
	replicaSet, err := dv_tu.CreateReplicaSetFromTemplate(dv_tu.NewTemplateArgs())
	if err != nil {
		t.Errorf("Error creating ReplicaSet from template %v", err)
	}
	rv.Validate(request, &replicaSet, testutils.ObjectKind(&replicaSet), false)
	metric, err := rv.metric.GetMetricWith(
		prometheus.Labels{"name": "foo", "namespace": "bar", "kind": "ReplicaSet"})
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("ReplicaSet test failed: got %d, want %d", metricValue, 0)
	}
}
