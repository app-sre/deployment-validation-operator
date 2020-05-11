package validations

import (
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	deploymentValidations = append(deploymentValidations, deploymentReplicas)
	replicaSetValidations = append(replicaSetValidations, replicaSetReplicas)
}

func deploymentReplicas(request reconcile.Request, instance *appsv1.Deployment, deleted bool) {
	log.Debugf("Replicas: deployment/%s [%s]", request.Name, request.Namespace)
	promLabels := getPromLabels(request.Name, request.Namespace, instance)

	if !deleted && *instance.Spec.Replicas < 3 {
		metricReplicas.With(promLabels).Set(1)
	} else {
		metricReplicas.Delete(promLabels)
	}
}

func replicaSetReplicas(request reconcile.Request, instance *appsv1.ReplicaSet, deleted bool) {
	log.Debugf("Replicas: replicaSet/%s [%s]", request.Name, request.Namespace)
	promLabels := getPromLabels(request.Name, request.Namespace, instance)

	if !deleted && *instance.Spec.Replicas < 3 {
		metricReplicas.With(promLabels).Set(1)
	} else {
		metricReplicas.Delete(promLabels)
	}
}
