package validations

import (
	"context"
	"reflect"

	"github.com/prometheus/client_golang/prometheus"

	appsv1 "k8s.io/api/apps/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func init() {
	AddValidation(newReplicaValidation())
}

type ReplicaValidation struct {
	ctx    context.Context
	metric *prometheus.GaugeVec
}

func newReplicaValidation() *ReplicaValidation {
	m := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ReplicaValidation",
		Help: "resource has less than 3 replicas.",
	}, []string{"namespace", "name", "kind"})
	metrics.Registry.MustRegister(m)

	return &ReplicaValidation{ctx: context.TODO(), metric: m}
}

func (r *ReplicaValidation) AppliesTo() map[string]struct{} {
	return map[string]struct{}{
		"Deployment": struct{}{},
		"ReplicaSet": struct{}{},
	}
}

func (r *ReplicaValidation) Validate(request reconcile.Request, kind string, obj interface{}, isDeleted bool) {
	logger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name, "Kind", kind)
	logger.V(2).Info("Validating replicas")

	minReplicas := int64(3)
	promLabels := getPromLabels(request.Name, request.Namespace, kind)

	replica_cnt := reflect.ValueOf(obj).FieldByName("Spec").FieldByName("Replicas").Elem().Int()
	if replica_cnt > 0 {
		if !isDeleted && replica_cnt < minReplicas {
			r.metric.With(promLabels).Set(1)
			logger.Info("has too few replicas", "current replicas", replica_cnt, "minimum replicas", minReplicas)
		} else {
			r.metric.Delete(promLabels)
		}
	}
}

func (r *ReplicaValidation) ValidateWithClient(kubeClient client.Client) {
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
