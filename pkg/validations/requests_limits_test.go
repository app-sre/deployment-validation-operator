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

func TestRequestLimitAppliesTo(t *testing.T) {
	tests := []struct {
		kind     string
		expected bool
	}{
		{"Deployment", true},
		{"ReplicaSet", true},
		{"Pod", false},
		{"StatefulSet", false},
	}

	rl := newRequestLimitValidation()

	for _, test := range tests {
		if _, ok := rl.AppliesTo()[test.kind]; ok != test.expected {
			t.Errorf("%s kind: got %t want %t", test.kind, ok, test.expected)
		}
	}
}

func TestDeploymentRequestLimitValidation(t *testing.T) {
	tests := []struct {
		resourceRequests bool
		resourceLimits   bool
		isDeleted        bool
		expected         int
	}{
		{true, true, false, 0},
		{true, false, false, 1},
		{false, true, false, 1},
		{false, false, false, 1},
		{false, false, true, 0}, // a bit weird, but this is what prom_tu.ToFloat64 returns
	}

	rl := newRequestLimitValidation()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}

	for _, test := range tests {
		args := dv_tu.NewTemplateArgs()
		args.ResourceRequests = test.resourceRequests
		args.ResourceLimits = test.resourceLimits
		deployment, err := dv_tu.CreateDeploymentFromTemplate(args)
		if err != nil {
			t.Errorf("Error creating deployment from template %v", err)
		}

		rl.Validate(request, &deployment, testutils.ObjectKind(&deployment), test.isDeleted)
		metric, err := rl.metric.GetMetricWith(
			prometheus.Labels{"name": "foo", "namespace": "bar", "kind": "Deployment"})
		if err != nil {
			t.Errorf("Error getting prometheus metric: %v", err)
		}

		if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != test.expected {
			t.Errorf("Deployment test failed %#v: got %d want %d", test, metricValue, test.expected)
		}
	}
}

func TestReplicaSetRequestLimitValidation(t *testing.T) {
	rl := newRequestLimitValidation()
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}
	replicaSet, err := dv_tu.CreateReplicaSetFromTemplate(dv_tu.NewTemplateArgs())
	if err != nil {
		t.Errorf("Error creating ReplicaSet from template %v", err)
	}
	rl.Validate(request, &replicaSet, testutils.ObjectKind(&replicaSet), false)
	metric, err := rl.metric.GetMetricWith(
		prometheus.Labels{"name": "foo", "namespace": "bar", "kind": "ReplicaSet"})
	if err != nil {
		t.Errorf("Error getting prometheus metric: %v", err)
	}

	if metricValue := int(prom_tu.ToFloat64(metric)); metricValue != 0 {
		t.Errorf("ReplicaSet test failed: got %d, want %d", metricValue, 0)
	}
}
