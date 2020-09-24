package controller

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"

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

var _ reconcile.Reconciler = &GenericReconciler{}

// GenericReconciler watches a defined object
type GenericReconciler struct {
	client         client.Client
	reconciledKind string
	reconciledObj  runtime.Object
}

// NewGenericReconciler returns a GenericReconciler struct
func NewGenericReconciler(obj runtime.Object) *GenericReconciler {
	kind := reflect.TypeOf(obj).String()
	kind = strings.SplitN(kind, ".", 2)[1]
	return &GenericReconciler{reconciledObj: obj, reconciledKind: kind}
}

func (gr *GenericReconciler) AddToManager(mgr manager.Manager) error {
	gr.client = mgr.GetClient()

	// Create a new controller
	c, err := controller.New(
		fmt.Sprintf("%sController", gr.reconciledKind),
		mgr,
		controller.Options{Reconciler: gr})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource
	err = c.Watch(&source.Kind{Type: gr.reconciledObj}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

// Reconcile watches an object kind and reports validation errors
func (gr *GenericReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	var log = logf.Log.WithName(fmt.Sprintf("%sController", gr.reconciledKind))
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconcile", "Kind", gr.reconciledKind)

	instance := gr.reconciledObj.DeepCopyObject()
	err := gr.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil && !errors.IsNotFound(err) {
		return reconcile.Result{Requeue: true}, err
	}

	deleted := err != nil && errors.IsNotFound(err)
	gvk := instance.GetObjectKind().GroupVersionKind()
	validations.RunValidations(request, instance, gvk.Kind, deleted)

	return reconcile.Result{}, nil
}
