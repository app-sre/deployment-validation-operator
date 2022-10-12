package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	"go.uber.org/multierr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/util/retry"

	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	"github.com/app-sre/deployment-validation-operator/pkg/validations"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
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
	discovery             discovery.DiscoveryInterface
}

// NewGenericReconciler returns a GenericReconciler struct
func NewGenericReconciler(client client.Client, discovery discovery.DiscoveryInterface) (*GenericReconciler, error) {
	listLimit, err := getListLimit()
	if err != nil {
		return nil, err
	}

	return &GenericReconciler{
		client:                client,
		discovery:             discovery,
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
	return mgr.Add(gr)
}

// Start validating the given object kind every interval.
func (gr *GenericReconciler) Start(ctx context.Context) error {
	var log = logf.Log.WithName("controller")

	for {
		select {
		case <-ctx.Done():
			// stop reconciling
			return nil
		default:
			if err := gr.reconcileEverything(ctx); err != nil && !errors.Is(err, context.Canceled) {
				// TODO: Improve error handling so that error can be returned to manager from here
				// this is done to make sure errors caused by skew in k8s version on server and
				// client-go version in the operator code does not create issues like
				// `batch/v1 CronJobs` failing in while `lookUpType()
				// nikthoma: oct 11, 2022
				log.Error(err, "error fetching and validating resource types")
			}
		}
	}
}

func (gr *GenericReconciler) reconcileEverything(ctx context.Context) error {
	var log = logf.Log.WithName("reconcileEverything")
	apiResources, err := reconcileResourceList(gr.discovery, gr.client.Scheme())
	if err != nil {
		return fmt.Errorf("retrieving resources to reconcile: %w", err)
	}

	for i, resource := range apiResources {
		log.Info("apiResource", "no:", i+1, "Group:", resource.Group,
			"Version:", resource.Version,
			"Kind:", resource.Kind)
	}

	gr.watchNamespaces.resetCache()

	if err := gr.processAllResources(ctx, apiResources); err != nil {
		return fmt.Errorf("processing all resources: %w", err)
	}

	gr.handleResourceDeletions()

	return nil
}

func (gr *GenericReconciler) processAllResources(ctx context.Context, resources []metav1.APIResource) error {
	var finalErr error

	for _, resource := range resources {
		gvk := gvkFromMetav1APIResource(resource)

		if resource.Namespaced {
			if err := gr.processNamespacedResource(ctx, gvk); err != nil {
				multierr.AppendInto(
					&finalErr,
					fmt.Errorf("processing namespace scoped resources of type %q: %w", gvk, err),
				)
			}
		} else {
			if err := gr.processClusterscopedResources(ctx, gvk); err != nil {
				multierr.AppendInto(
					&finalErr,
					fmt.Errorf("processing cluster scoped resources of type %q: %w", gvk, err),
				)
			}
		}
	}

	return finalErr
}

func (gr *GenericReconciler) processNamespacedResource(ctx context.Context, gvk schema.GroupVersionKind) error {
	var finalErr error

	namespaces, err := gr.watchNamespaces.getWatchNamespaces(ctx, gr.client)
	if err != nil {
		return fmt.Errorf("getting watched namespaces: %w", err)
	}

	for _, ns := range *namespaces {
		if err := gr.processObjectInstances(ctx, gvk, ns.name); err != nil {
			multierr.AppendInto(&finalErr, fmt.Errorf("processing resources: %w", err))
		}
	}

	return finalErr
}

func (gr *GenericReconciler) processClusterscopedResources(ctx context.Context, gvk schema.GroupVersionKind) error {
	return gr.processObjectInstances(ctx, gvk, "")
}

func (gr *GenericReconciler) processObjectInstances(ctx context.Context,
	gvk schema.GroupVersionKind, namespace string) error {
	gvk.Kind = gvk.Kind + "List"

	do := func() (done bool, err error) {
		return true, gr.paginatedList(ctx, gvk, namespace)
	}

	if err := wait.ExponentialBackoffWithContext(ctx, retry.DefaultBackoff, do); err != nil {
		return fmt.Errorf("processing list: %w", err)
	}

	return nil
}

func (gr *GenericReconciler) reconcile(ctx context.Context, obj *unstructured.Unstructured) error {
	gr.currentObjects.store(obj, "")
	if gr.objectValidationCache.objectAlreadyValidated(obj) {
		return nil
	}

	var log = logf.Log.WithName(fmt.Sprintf("%s Validation", obj.GetObjectKind().GroupVersionKind()))

	request := utils.NewRequestFromObj(obj)
	if len(request.Namespace) > 0 {
		namespaceUID := gr.watchNamespaces.getNamespaceUID(request.Namespace)
		if len(namespaceUID) == 0 {
			log.V(2).Info("Namespace UID not found", request.Namespace)
		}
		request.NamespaceUID = namespaceUID
	}

	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconcile", "Kind", obj.GetObjectKind().GroupVersionKind())

	typedClientObject, err := gr.unstructuredToTyped(obj)
	if err != nil {
		return fmt.Errorf("instantiating typed object: %w", err)
	}

	outcome, err := validations.RunValidations(request, typedClientObject)
	if err != nil {
		return fmt.Errorf("running validations: %w", err)
	}

	gr.objectValidationCache.store(obj, outcome)

	return nil
}

func (gr *GenericReconciler) unstructuredToTyped(obj *unstructured.Unstructured) (client.Object, error) {
	typedResource, err := gr.lookUpType(obj)
	if err != nil {
		return nil, fmt.Errorf("looking up object type: %w", err)
	}

	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.Object, typedResource); err != nil {
		return nil, fmt.Errorf("converting unstructured to typed object: %w", err)
	}

	return typedResource.(client.Object), nil
}

func (gr *GenericReconciler) lookUpType(obj *unstructured.Unstructured) (runtime.Object, error) {
	gvk := obj.GetObjectKind().GroupVersionKind()

	typedObj, err := gr.client.Scheme().New(gvk)
	if err != nil {
		return nil, fmt.Errorf("creating new object of type %s: %w", gvk, err)
	}

	return typedObj, nil
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

		if err := gr.client.List(ctx, &list, listOptions); err != nil {
			return fmt.Errorf("listing %s: %w", gvk.String(), err)
		}

		for i := range list.Items {
			obj := list.Items[i]

			if err := gr.reconcile(ctx, &obj); err != nil {
				return fmt.Errorf(
					"reconciling object '%s/%s': %w", obj.GetNamespace(), obj.GetName(), err,
				)
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
