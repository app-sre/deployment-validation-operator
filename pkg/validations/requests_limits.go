package validations

import (
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	deploymentValidations = append(deploymentValidations, deploymentRequestsLimits)
	replicaSetValidations = append(replicaSetValidations, replicaSetRequestsLimits)
}

func deploymentRequestsLimits(request reconcile.Request, instance *appsv1.Deployment, deleted bool) {
	log.Debugf("RequestsLimits: deployment/%s [%s]", request.Name, request.Namespace)
	promLabels := getPromLabels(request.Name, request.Namespace, instance)
	validateRequestsLimits(instance.Spec.Template, promLabels, deleted)
}

func replicaSetRequestsLimits(request reconcile.Request, instance *appsv1.ReplicaSet, deleted bool) {
	log.Debugf("RequestsLimits: replicaSet/%s [%s]", request.Name, request.Namespace)
	promLabels := getPromLabels(request.Name, request.Namespace, instance)
	validateRequestsLimits(instance.Spec.Template, promLabels, deleted)
}

func validateRequestsLimits(spec v1.PodTemplateSpec, promLabels prometheus.Labels, deleted bool) {
	if deleted {
		log.Infof("is deleted validateRequestsLimits %v", promLabels)
		metricRequestsLimits.Delete(promLabels)
		return
	}

	for _, c := range spec.Spec.Containers {
		if c.Resources.Requests.Memory().IsZero() || c.Resources.Requests.Cpu().IsZero() ||
			c.Resources.Limits.Memory().IsZero() || c.Resources.Limits.Cpu().IsZero() {

			metricRequestsLimits.With(promLabels).Set(1)
			return
		}
	}

	metricRequestsLimits.Delete(promLabels)
}
