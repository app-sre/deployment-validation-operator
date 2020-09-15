package validations

import (
	"context"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var requestLimitValidationMetric = newGaugeVecMetric(
	"request_limit_validation",
	"resource does not have requests or limits.",
	[]string{"namespace", "name", "kind"})

func init() {
	metrics.Registry.MustRegister(requestLimitValidationMetric)
	AddValidation(newRequestLimitValidation())
}

type RequestLimitValidation struct {
	ctx    context.Context
	metric *prometheus.GaugeVec
}

func newRequestLimitValidation() *RequestLimitValidation {
	return &RequestLimitValidation{ctx: context.TODO(), metric: requestLimitValidationMetric}
}

func (r *RequestLimitValidation) AppliesTo() map[string]struct{} {
	return map[string]struct{}{
		"Deployment": {},
		"ReplicaSet": {},
	}
}

func (r *RequestLimitValidation) Validate(request reconcile.Request, obj interface{}, kind string, isDeleted bool) {
	logger := log.WithValues(
		"Request.Namespace", request.Namespace,
		"Request.Name", request.Name,
		"Kind", kind)
	logger.V(2).Info("Validating limits")

	promLabels := getPromLabels(request.Name, request.Namespace, kind)

	if isDeleted {
		r.metric.Delete(promLabels)
		return
	}

	replicaCnt := reflect.ValueOf(obj).Elem().FieldByName("Spec").FieldByName("Replicas").Elem().Int()
	if replicaCnt > 0 {
		podTemplateSpec := reflect.
			ValueOf(obj).
			Elem().
			FieldByName("Spec").
			FieldByName("Template").
			Interface().(core_v1.PodTemplateSpec)
		for _, c := range podTemplateSpec.Spec.Containers {
			if c.Resources.Requests.Memory().IsZero() || c.Resources.Requests.Cpu().IsZero() ||
				c.Resources.Limits.Memory().IsZero() || c.Resources.Limits.Cpu().IsZero() {
				logger.Info("does not have requests or limits set")
				r.metric.With(promLabels).Set(1)
				return
			}

			r.metric.With(promLabels).Set(0)
		}
	}
}
