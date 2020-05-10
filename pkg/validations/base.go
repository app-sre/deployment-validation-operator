package validations

import (
	"reflect"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var deploymentValidations []func(reconcile.Request, *appsv1.Deployment, bool)
var replicaSetValidations []func(reconcile.Request, *appsv1.ReplicaSet, bool)

func getPromLabels(name, namespace string, instance interface{}) prometheus.Labels {

	// this gets the kind which will evaluate to something like `*v1.Deployment`
	kind := reflect.TypeOf(instance).String()

	// remove the `*` prefix from the kind
	kind = strings.TrimPrefix(kind, "*")

	return prometheus.Labels{"namespace": namespace, "name": name, "kind": kind}
}

// DeploymentValidations runs validations on Deployments
func DeploymentValidations(request reconcile.Request, instance *appsv1.Deployment, deleted bool) {
	for _, validateFunc := range deploymentValidations {
		validateFunc(request, instance, deleted)
	}
}

// ReplicaSetValidations runs validations on ReplicaSets
func ReplicaSetValidations(request reconcile.Request, instance *appsv1.ReplicaSet, deleted bool) {
	for _, validateFunc := range replicaSetValidations {
		validateFunc(request, instance, deleted)
	}
}
