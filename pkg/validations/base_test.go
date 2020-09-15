package validations

import (
	"testing"

	"github.com/app-sre/dv-operator/pkg/testutils"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestRunValidations(t *testing.T) {
	request := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "foo", Namespace: "bar"},
	}

	deployment, err := testutils.CreateDeploymentFromTemplate(
		testutils.NewTemplateArgs())
	if err != nil {
		t.Errorf("Error creating deployment from template %v", err)
	}

	RunValidations(request, &deployment, "Deployment", false)
}
