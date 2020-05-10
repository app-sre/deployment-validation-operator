package replicaset

import (
	"context"

	"github.com/jmelis/dv-operator/pkg/validations"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_replicaset")

// Add creates a new ReplicaSet Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileReplicaSet{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("replicaset-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource ReplicaSet
	err = c.Watch(&source.Kind{Type: &appsv1.ReplicaSet{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileReplicaSet{}

// ReconcileReplicaSet watches a ReplicaSet object
type ReconcileReplicaSet struct {
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile watches replicasets and reports validation errors
func (r *ReconcileReplicaSet) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Processing replicaset")

	instance := &appsv1.ReplicaSet{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	deleted := err != nil && errors.IsNotFound(err)

	validations.ReplicaSetValidations(request, instance, deleted)

	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}
