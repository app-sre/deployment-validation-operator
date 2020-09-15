package validations

import (
	"context"
	"fmt"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	validation, err := newReplicaValidation()
	if err != nil {
		fmt.Printf("failed to add ReplicaValidation: %+v\n", err)
	} else {
		AddValidation(validation)
	}
}

type ReplicaValidation struct {
	ctx    context.Context
	metric *prometheus.GaugeVec
}

func newReplicaValidation() (*ReplicaValidation, error) {
	m, err := newGaugeVecMetric(
		"replica_validation",
		"resource has less than 3 replicas.",
		[]string{"namespace", "name", "kind"})
	if err != nil {
		return nil, err
	}
	metrics.Registry.MustRegister(m)

	return &ReplicaValidation{ctx: context.TODO(), metric: m}, nil
}

func (r *ReplicaValidation) AppliesTo() map[string]struct{} {
	return map[string]struct{}{
		"Deployment": {},
		"ReplicaSet": {},
	}
}

func (r *ReplicaValidation) Validate(request reconcile.Request, kind string, obj interface{}, isDeleted bool) {
	logger := log.WithValues(
		"Request.Namespace", request.Namespace,
		"Request.Name", request.Name,
		"Kind", kind)
	logger.V(2).Info("Validating replicas")

	minReplicas := int64(3)
	promLabels := getPromLabels(request.Name, request.Namespace, kind)

	replicaCnt := reflect.ValueOf(obj).FieldByName("Spec").FieldByName("Replicas").Elem().Int()
	if replicaCnt > 0 {
		if isDeleted {
			r.metric.Delete(promLabels)
		} else if replicaCnt < minReplicas {
			r.metric.With(promLabels).Set(1)
			logger.Info("has too few replicas", "current replicas", replicaCnt, "minimum replicas",
				minReplicas)
		} else {
			r.metric.With(promLabels).Set(0)
		}
	}
}
