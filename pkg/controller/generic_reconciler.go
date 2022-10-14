package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"k8s.io/client-go/util/workqueue"

	"go.uber.org/multierr"
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
	_                    manager.Runnable = &GenericReconciler{}
	ctxKeyCurrentObjects                  = struct{}{}
)

// GenericReconciler watches a defined object
type GenericReconciler struct {
	listLimit             int64
	watchNamespaces       *watchNamespacesCache
	objectValidationCache *cacheSet
	client                client.Client
	discovery             discovery.DiscoveryInterface
	workqueue             workqueue.RateLimitingInterface
	workers               int
	enqueueInterval       time.Duration
}

// NewGenericReconciler returns a GenericReconciler struct
func NewGenericReconciler(client client.Client, discovery discovery.DiscoveryInterface) (*GenericReconciler, error) {
	lLimit, err := listLimit()
	if err != nil {
		return nil, err
	}
	wCount, err := workerCount()
	if err != nil {
		return nil, err
	}
	eInterval, err := enqueueInterval()
	if err != nil {
		return nil, err
	}
	return &GenericReconciler{
		client:                client,
		discovery:             discovery,
		listLimit:             lLimit,
		watchNamespaces:       newWatchNamespacesCache(),
		objectValidationCache: newCacheSet(),
		workqueue:             workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter()),
		workers:               wCount,
		enqueueInterval:       eInterval,
	}, nil
}

func enqueueInterval() (time.Duration, error) {
	val, err := defaultOrEnv(EnvValidationCheckInterval, defaultValidationCheckInterval)
	return time.Duration(val) * time.Second, err
}

func workerCount() (int, error) {
	return defaultOrEnv(EnvNumberOfWorkers, defaultNumberOfWorkers)
}

func listLimit() (int64, error) {
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
	ticker := time.NewTicker(gr.enqueueInterval)
	defer gr.workqueue.ShutDown()

	// start workers
	for i := 0; i < gr.workers; i++ {
		go gr.runWorker(i + 1)
	}

	if err := gr.enqueueAllResources(); err != nil && !errors.Is(err, context.Canceled) {
		log.Error(err, "error enqueueing all resource types for validation")
		return err
	}

	for {
		select {
		case <-ctx.Done():
			// stop reconciling
			return nil
		case <-ticker.C:
			if err := gr.enqueueAllResources(); err != nil && !errors.Is(err, context.Canceled) {
				log.Error(err, "error enqueueing all resource types for validation")
				return err
			}
		}
	}
}

func (gr *GenericReconciler) enqueueAllResources() error {
	var log = logf.Log.WithName("enqueueAllResources")
	apiResources, err := reconcileResourceList(gr.discovery, gr.client.Scheme())
	if err != nil {
		return fmt.Errorf("retrieving resources to reconcile: %w", err)
	}

	for i, resource := range apiResources {
		log.Info("apiResource", "no:", i+1, "Group:", resource.Group,
			"Version:", resource.Version,
			"Kind:", resource.Kind)
		gr.workqueue.Add(resource)
	}
	gr.watchNamespaces.resetCache()
	return nil
}

func (gr *GenericReconciler) runWorker(i int) {
	log := logf.Log.WithName("runWorker")
	log.Info("started", "worker", i)
	for gr.attendNextWorkItem() {
	}
}

func (gr *GenericReconciler) attendNextWorkItem() bool {
	var log = logf.Log.WithName("attendNextWorkItem")
	item, shutdown := gr.workqueue.Get()
	if shutdown {
		return false
	}
	apiResource := item.(hashableAPIResource)
	err := gr.processWorkItem(context.Background(), apiResource)
	log.Info("processing API Resource",
		"Group", apiResource.Group,
		"Version", apiResource.Version,
		"Kind", apiResource.Kind)
	if err != nil {
		log.Error(err, "error processing API Resource",
			"Group", apiResource.Group,
			"Version", apiResource.Version,
			"Kind", apiResource.Kind)
		if gr.workqueue.NumRequeues(item) > 5 {
			gr.workqueue.Forget(item)
			gr.workqueue.Done(item)
		}
		gr.workqueue.AddRateLimited(item)
	}
	gr.workqueue.Done(item)
	return true
}

func (gr *GenericReconciler) processWorkItem(ctx context.Context, resource hashableAPIResource) error {
	currentObjects := newValidationCache()
	ctx = context.WithValue(ctx, ctxKeyCurrentObjects, currentObjects)
	gvk := resource.gvk()
	var err error
	if resource.Namespaced {
		err = gr.processNamespacedResource(ctx, gvk)

	} else {
		err = gr.processClusterscopedResource(ctx, gvk)
	}
	if err != nil {
		return err
	}
	gr.handleResourceDeletions(ctx, gvk)
	currentObjects.drain()
	return err
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

func (gr *GenericReconciler) processClusterscopedResource(ctx context.Context, gvk schema.GroupVersionKind) error {
	return gr.processObjectInstances(ctx, gvk, "")
}

func (gr *GenericReconciler) processObjectInstances(ctx context.Context,
	gvk schema.GroupVersionKind, namespace string) error {
	do := func() (done bool, err error) {
		return true, gr.paginatedList(ctx, gvk, namespace)
	}

	if err := wait.ExponentialBackoffWithContext(ctx, retry.DefaultBackoff, do); err != nil {
		return fmt.Errorf("processing list: %w", err)
	}
	return nil
}

func (gr *GenericReconciler) reconcile(ctx context.Context, obj *unstructured.Unstructured) error {
	currentObjects := ctx.Value(ctxKeyCurrentObjects).(*validationCache)
	currentObjects.store(obj, validations.ValidationStatus{})
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

	vStatus, err := validations.RunValidations(request, typedClientObject)
	if err != nil {
		return fmt.Errorf("running validations: %w", err)
	}

	gr.objectValidationCache.store(obj, vStatus)

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

func (gr *GenericReconciler) handleResourceDeletions(ctx context.Context, gvk schema.GroupVersionKind) {
	currentObjects := ctx.Value(ctxKeyCurrentObjects).(*validationCache)
	objectHistory, ok := (*gr.objectValidationCache)[cacheSetKey(gvk)]
	if !ok {
		return
	}
	for k, vStatus := range *objectHistory {
		if currentObjects.has(k) {
			continue
		}
		promLabels := vStatus.status.PromLabels
		validations.DeleteMetrics(promLabels)
		objectHistory.removeKey(k)
	}
}

func (gr *GenericReconciler) paginatedList(ctx context.Context, gvk schema.GroupVersionKind, namespace string) error {
	list := unstructured.UnstructuredList{}
	listOptions := &client.ListOptions{
		Limit:     gr.listLimit,
		Namespace: namespace,
	}
	gvk.Kind = gvk.Kind + "List"
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
