package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/go-logr/logr"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	logger                logr.Logger
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
		logger:                ctrl.Log.WithName("reconcile"),
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
				gr.logger.Error(err, "error fetching and validating resource types")
			}
		}
	}
}

func (gr *GenericReconciler) reconcileEverything(ctx context.Context) error {
	apiResources, err := reconcileResourceList(gr.discovery, gr.client.Scheme())
	if err != nil {
		return fmt.Errorf("retrieving resources to reconcile: %w", err)
	}

	for i, resource := range apiResources {
		gr.logger.Info("apiResource", "no", i+1, "Group", resource.Group,
			"Version", resource.Version,
			"Kind", resource.Kind)
	}

	gr.watchNamespaces.resetCache()
	namespaces, err := gr.watchNamespaces.getWatchNamespaces(ctx, gr.client)
	if err != nil {
		return fmt.Errorf("getting watched namespaces: %w", err)
	}

	gvkResources := gr.getNamespacedResourcesGVK(apiResources)
	errNR := gr.processNamespacedResources(ctx, gvkResources, namespaces)
	if errNR != nil {
		return fmt.Errorf("processing namespace scoped resources: %w", errNR)
	}

	gr.handleResourceDeletions()

	return nil
}

// getAppLabel tries to read "app" label from the provided unstructured object
func getAppLabel(object *unstructured.Unstructured) (string, error) {
	appLabel, found, err := unstructured.NestedString(object.Object, "metadata", "labels", "app")
	// if not found try another path - e.g for PDB resource
	if !found {
		appLabel, found, err = unstructured.NestedString(object.Object,
			"spec", "selector", "matchLabels", "app")
		if err != nil {
			return "", err
		}
		if !found {
			return "", fmt.Errorf("can't find any 'app' label for %s resource from %s namespace",
				object.GetName(), object.GetNamespace())
		}
	}
	if err != nil {
		return "", err
	}
	return appLabel, nil
}

// groupAppObjects iterates over provided GroupVersionKind in given namespace
// and returns map of objects grouped by their "app" label
func (gr *GenericReconciler) groupAppObjects(ctx context.Context,
	namespace string, gvks []schema.GroupVersionKind) (map[string][]*unstructured.Unstructured, error) {
	relatedObjects := make(map[string][]*unstructured.Unstructured)
	for _, gvk := range gvks {
		list := unstructured.UnstructuredList{}
		listOptions := &client.ListOptions{
			Limit:     gr.listLimit,
			Namespace: namespace,
		}
		for {
			list.SetGroupVersionKind(gvk)

			if err := gr.client.List(ctx, &list, listOptions); err != nil {
				return nil, fmt.Errorf("listing %s: %w", gvk.String(), err)
			}

			for i := range list.Items {
				obj := &list.Items[i]
				appLabel, err := getAppLabel(obj)
				if err != nil {
					// swallow the error here. it will be too noisy to log
					continue
				}
				relatedObjects[appLabel] = append(relatedObjects[appLabel], obj)
			}

			listContinue := list.GetContinue()
			if listContinue == "" {
				break
			}
			listOptions.Continue = listContinue
		}

	}
	return relatedObjects, nil
}

func (gr *GenericReconciler) processNamespacedResources(
	ctx context.Context, gvks []schema.GroupVersionKind, namespaces *[]namespace) error {

	for _, ns := range *namespaces {
		relatedObjects, err := gr.groupAppObjects(ctx, ns.name, gvks)
		if err != nil {
			return err
		}
		for label, objects := range relatedObjects {
			gr.logger.Info("reconcileNamespaceResources",
				"Reconciling group of", len(objects), "objects with app label", label)
			err := gr.reconcileGroupOfObjects(ctx, objects, ns.name)
			if err != nil {
				return fmt.Errorf(
					"reconciling related objects with 'app' label value '%s': %w", label, err,
				)
			}
		}
	}

	return nil
}

func (gr *GenericReconciler) reconcileGroupOfObjects(ctx context.Context,
	objs []*unstructured.Unstructured, namespace string) error {

	if gr.allObjectsValidated(objs) {
		gr.logger.Info("reconcileGroupOfObjects", "All objects are validated", "Nothing to do")
		return nil
	}

	namespaceUID := gr.watchNamespaces.getNamespaceUID(namespace)
	cliObjects := make([]client.Object, 0, len(objs))
	for _, o := range objs {
		typedClientObject, err := gr.unstructuredToTyped(o)
		if err != nil {
			return fmt.Errorf("instantiating typed object: %w", err)
		}
		cliObjects = append(cliObjects, typedClientObject)
	}

	outcome, err := validations.RunValidationsForObjects(cliObjects, namespaceUID)
	if err != nil {
		return fmt.Errorf("running validations: %w", err)
	}
	for _, o := range objs {
		gr.objectValidationCache.store(o, outcome)
	}

	return nil
}

// allObjectsValidated checks whether all unstructured objects passed as argument are validated
// and thus present in the cache
func (gr *GenericReconciler) allObjectsValidated(objs []*unstructured.Unstructured) bool {
	allObjectsValidated := true
	// we must be sure that all objects in the given group are cached (validated)
	// see DVO-103
	for _, o := range objs {
		gr.currentObjects.store(o, "")
		if !gr.objectValidationCache.objectAlreadyValidated(o) {
			allObjectsValidated = false
		}
	}
	return allObjectsValidated
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
	for k, v := range *gr.objectValidationCache {
		if gr.currentObjects.has(k) {
			continue
		}

		req := validations.Request{
			Kind:         k.kind,
			Name:         k.name,
			Namespace:    k.namespace,
			NamespaceUID: gr.watchNamespaces.getNamespaceUID(k.namespace),
			UID:          v.uid,
		}

		validations.DeleteMetrics(req.ToPromLabels())

		gr.objectValidationCache.removeKey(k)

	}
	gr.currentObjects.drain()
}

// getNamespacedResourcesGVK filters APIResources and returns the ones within a namespace
func (gr GenericReconciler) getNamespacedResourcesGVK(resources []metav1.APIResource) []schema.GroupVersionKind {
	namespacedResources := make([]schema.GroupVersionKind, 0)
	for _, resource := range resources {
		if resource.Namespaced {
			gvk := gvkFromMetav1APIResource(resource)
			namespacedResources = append(namespacedResources, gvk)
		}
	}

	return namespacedResources
}
