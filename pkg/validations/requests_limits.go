package validations

import (
	"context"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"

	appsv1 "k8s.io/api/apps/v1"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	AddValidation(newRequestLimitValidation())
}

type RequestLimitValidation struct {
	ctx    context.Context
	metric *prometheus.GaugeVec
}

func newRequestLimitValidation() *RequestLimitValidation {
	m := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "RequestLimitValidation",
		Help: "resource does not have requests or limits.",
	}, []string{"namespace", "name", "kind"})
	metrics.Registry.MustRegister(m)

	return &RequestLimitValidation{ctx: context.TODO(), metric: m}
}

func (r *RequestLimitValidation) AppliesTo() map[string]struct{} {
	return map[string]struct{}{
		"Deployment": struct{}{},
		"ReplicaSet": struct{}{},
	}
}

func (r *RequestLimitValidation) Validate(request reconcile.Request, kind string, obj interface{}, isDeleted bool) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "Kind", kind)
	logger.V(2).Info("Validating limits")

	promLabels := getPromLabels(request.Name, request.Namespace, kind)

	if isDeleted {
		r.metric.Delete(promLabels)
		return
	}

	replica_cnt := reflect.ValueOf(obj).FieldByName("Spec").FieldByName("Replicas").Elem().Int()
	if replica_cnt > 0 {
		podTemplateSpec := reflect.ValueOf(obj).FieldByName("Spec").FieldByName("Template").Interface().(v1.PodTemplateSpec)
		for _, c := range podTemplateSpec.Spec.Containers {
			if c.Resources.Requests.Memory().IsZero() || c.Resources.Requests.Cpu().IsZero() ||
				c.Resources.Limits.Memory().IsZero() || c.Resources.Limits.Cpu().IsZero() {
				logger.Info("does not have requests or limits set")
				r.metric.With(promLabels).Set(1)
				return
			}
		}
	}
}

func (r *RequestLimitValidation) ValidateWithClient(kubeClient client.Client) {
	listObjs := []runtime.Object{&appsv1.DeploymentList{}, &appsv1.ReplicaSetList{}}
	for _, listObj := range listObjs {
		err := kubeClient.List(r.ctx, listObj, client.InNamespace(metav1.NamespaceAll))
		if err != nil {
			log.Info("unable to list object", "error", err)
		}
		items := reflect.ValueOf(listObj).Elem().FieldByName("Items")
		for i := 0; i < items.Len(); i++ {
			obj := items.Index(i)
			objInterface := obj.Interface()
			kind := reflect.TypeOf(objInterface).String()
			req := reconcile.Request{}
			req.Namespace = obj.FieldByName("Namespace").String()
			req.Name = obj.FieldByName("Name").String()
			r.Validate(req, kind, objInterface, false)
		}
	}
}
