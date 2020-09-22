package validations

import (
	"context"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var replicaValidationMetric = newGaugeVecMetric(
	"replica_validation",
	"resource has less than 3 replicas.",
	[]string{"namespace", "name", "kind"})

func init() {
	metrics.Registry.MustRegister(replicaValidationMetric)
	AddValidation(newReplicaValidation())
}

type ReplicaValidation struct {
	ctx    context.Context
	metric *prometheus.GaugeVec
}

func newReplicaValidation() *ReplicaValidation {
	return &ReplicaValidation{ctx: context.TODO(), metric: replicaValidationMetric}
}

func (r *ReplicaValidation) AppliesTo() map[string]struct{} {
	return map[string]struct{}{
		"Deployment": {},
		"ReplicaSet": {},
	}
}

func (r *ReplicaValidation) Validate(request reconcile.Request, obj interface{}, kind string, isDeleted bool) {
	logger := log.WithValues(
		"Request.Namespace", request.Namespace,
		"Request.Name", request.Name,
		"Kind", kind)
	logger.V(2).Info("Validating replicas")

	minReplicas := int64(3)
	promLabels := getPromLabels(request.Name, request.Namespace, kind)

	replicaCnt := reflect.ValueOf(obj).Elem().FieldByName("Spec").FieldByName("Replicas").Elem().Int()
	if replicaCnt > 0 {
		if isDeleted {
			r.metric.Delete(promLabels)
		} else if replicaCnt < minReplicas {
			r.metric.With(promLabels).Set(1)
			logger.Info("has too few replicas", "current replicas", replicaCnt,
				"minimum replicas", minReplicas)
		} else {
			r.metric.With(promLabels).Set(0)
		}
	}
}
