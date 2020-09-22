package controller

import (
	"testing"

	"github.com/app-sre/dv-operator/pkg/testutils"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestGenericReconciler(t *testing.T) {
	deployment, err := testutils.CreateDeploymentFromTemplate(
		testutils.NewTemplateArgs())
	if err != nil {
		t.Errorf("Error creating deployment from template %v", err)
	}

	// Objects to track in the fake client.
	objs := []runtime.Object{&deployment}

	// Create a fake client to mock API calls.
	client := fake.NewFakeClient(objs...)

	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}

	gr := &GenericReconciler{
		client:         client,
		reconciledKind: testutils.ObjectKind(&deployment),
		reconciledObj:  &deployment}

	_, err = gr.Reconcile(request)
	if err != nil {
		t.Errorf("Error reconciling %v", err)
	}

	// since we're not modifying k8s objects, not much to see here except for
	// checking metrics registered, but that is done in the validation tests
}
