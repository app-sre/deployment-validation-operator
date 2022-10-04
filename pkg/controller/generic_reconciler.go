package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	osappsscheme "github.com/openshift/client-go/apps/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	_ manager.Runnable = &GenericReconciler{}
)

// GenericReconciler watches a defined object
type GenericReconciler struct {
	listLimit             int64
	watchNamespaces       *watchNamespacesCache
	objectValidationCache *validationCache
	currentObjects        *validationCache
	client                client.Client
	discoveryClient       *discovery.DiscoveryClient
}

// NewGenericReconciler returns a GenericReconciler struct
func NewGenericReconciler() (*GenericReconciler, error) {
	listLimit, err := getListLimit()
	if err != nil {
		return nil, err
	}
	return &GenericReconciler{
		listLimit:             listLimit,
		watchNamespaces:       newWatchNamespacesCache(),
		objectValidationCache: newValidationCache(),
		currentObjects:        newValidationCache(),
	}, nil
}

func getListLimit() (int64, error) {
	intVal, err := defaultOrEnv(EnvResorucesPerListQuery, defaultListLimit)
	return int64(intVal), err
}

func defaultOrEnv(envName string, defaultIntVal int) (int, error) {
	envVal, ok, err := intFromEnv(envName)
	if err != nil {
		return 0, err
	}
	if ok {
		return envVal, nil
	}
	return defaultIntVal, nil
}

func intFromEnv(envName string) (int, bool, error) {
	strVal, ok := os.LookupEnv(envName)
	if !ok || strVal == "" {
		return 0, false, nil
	}

	intVal, err := strconv.Atoi(strVal)
	if err != nil {
		return 0, false, err
	}

	return intVal, true, nil
}

// AddToManager will add the reconciler for the configured obj to a manager.
func (gr *GenericReconciler) AddToManager(mgr manager.Manager) error {
	kubeconfig := mgr.GetConfig()
	dc, err := discovery.NewDiscoveryClientForConfig(kubeconfig)
	if err != nil {
		return err
	}
	gr.discoveryClient = dc
	gr.client = mgr.GetClient()
	return mgr.Add(gr)
}

// Start validating the given object kind every interval.
func (gr *GenericReconciler) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			// stop reconciling
			return nil
		default:
			err := gr.reconcileEverything(ctx)
			if err != nil && !errors.Is(err, context.Canceled) {
				// could not recover
				return err
			}
		}
	}
}

func (gr *GenericReconciler) reconcileEverything(ctx context.Context) error {
	apiResources, err := reconcileResourceList(gr.discoveryClient)
	if err != nil {
		return err
	}
	gr.watchNamespaces.resetCache()
	err = gr.processAllResources(ctx, apiResources)
	if err != nil {
		return err
	}

	gr.handleResourceDeletions()
	return nil
}

func (gr *GenericReconciler) processAllResources(ctx context.Context, resources []metav1.APIResource) error {
	for _, resource := range resources {
		gvk := gvkFromMetav1APIResource(resource)
		var err error
		if resource.Namespaced {
			err = gr.processNamespacedResource(ctx, gvk)
		} else {
			err = gr.processClusterscopedObjects(ctx, gvk)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (gr *GenericReconciler) processNamespacedResource(ctx context.Context, gvk schema.GroupVersionKind) error {
	namespaces, err := gr.watchNamespaces.getWatchNamespaces(ctx, gr.client)
	if err != nil {
		return err
	}
	for _, ns := range *namespaces {
		err := gr.processObjectInstances(ctx, gvk, ns)
		if err != nil {
			return err
		}
	}
	return nil
}

func (gr *GenericReconciler) processClusterscopedObjects(ctx context.Context, gvk schema.GroupVersionKind) error {
	return gr.processObjectInstances(ctx, gvk, "")
}

func (gr *GenericReconciler) processObjectInstances(ctx context.Context,
	gvk schema.GroupVersionKind, namespace string) error {
	gvk.Kind = gvk.Kind + "List"

	err := wait.ExponentialBackoffWithContext(
		ctx, retry.DefaultBackoff,
		func() (done bool, err error) {
			return true, gr.paginatedList(ctx, gvk, namespace)
		})

	return err
}

func (gr *GenericReconciler) reconcile(ctx context.Context, obj *unstructured.Unstructured) error {
	gr.currentObjects.store(obj, "")
	if gr.objectValidationCache.objectAlreadyValidated(obj) {
		return nil
	}
	request := reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(obj),
	}

	var log = logf.Log.WithName(fmt.Sprintf("%s Validation", obj.GetObjectKind().GroupVersionKind()))
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconcile", "Kind", obj.GetObjectKind().GroupVersionKind())
	typedClientObject, err := unstructuredToTyped(obj)
	if err != nil {
		return err
	}
	outcome, err := validations.RunValidations(request, typedClientObject)
	if err != nil {
		return err
	}

	gr.objectValidationCache.store(obj, outcome)

	return nil
}

func unstructuredToTyped(obj *unstructured.Unstructured) (client.Object, error) {
	typedResource, err := lookUpType(obj)
	if err != nil {
		return nil, err
	}
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, typedResource)
	if err != nil {
		return nil, err
	}
	return typedResource.(client.Object), err

}

func lookUpType(obj *unstructured.Unstructured) (runtime.Object, error) {
	kubeScheme := scheme.Scheme
	openshiftScheme := osappsscheme.Scheme

	gvk := obj.GetObjectKind().GroupVersionKind()
	typedObj, err := kubeScheme.New(gvk)
	if err == nil {
		return typedObj, nil
	}
	if !runtime.IsNotRegisteredError(err) {
		return nil, err
	}
	typedObj, err = openshiftScheme.New(gvk)
	if err == nil {
		return typedObj, nil
	}
	return nil, err
}

func (gr *GenericReconciler) handleResourceDeletions() {
	for k := range *gr.objectValidationCache {
		if gr.currentObjects.has(k) {
			continue
		}
		validations.DeleteMetrics(k.namespace, k.name, k.kind)
		gr.objectValidationCache.removeKey(k)

	}
	gr.currentObjects.drain()
}

func (gr *GenericReconciler) paginatedList(ctx context.Context, gvk schema.GroupVersionKind, namespace string) error {
	list := unstructured.UnstructuredList{}
	listOptions := &client.ListOptions{
		Limit:     gr.listLimit,
		Namespace: namespace,
	}
	for {
		list.SetGroupVersionKind(gvk)
		err := gr.client.List(ctx, &list, listOptions)
		if err != nil {
			return fmt.Errorf("listing %s: %w", gvk.String(), err)
		}

		for i := range list.Items {
			err := gr.reconcile(ctx, &list.Items[i])
			if err != nil {
				return err
			}
		}
		listContinue := list.GetContinue()
		if listContinue == "" {
			break
		}
		listOptions.Continue = listContinue
	}
	return nil
}
