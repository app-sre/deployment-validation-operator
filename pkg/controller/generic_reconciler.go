package controller

import (
    "context"
    "errors"
    "fmt"
    "github.com/app-sre/deployment-validation-operator/pkg/validations"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
    "k8s.io/client-go/discovery"
    "k8s.io/client-go/rest"
    "os"
    "strconv"
    "time"

    "k8s.io/apimachinery/pkg/util/wait"
    "k8s.io/client-go/util/retry"

    "sigs.k8s.io/controller-runtime/pkg/client"
    logf "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/manager"
    "sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const (
    // default interval to run validations in.
    // A 10 percent jitter will be added to the reconcile interval between reconcilers,
    // so that not all reconcilers will not send list requests simultaneously.
    defaultReconcileInterval = 5 * time.Minute

    // default number of resources retrieved from the api server per list request
    // the usage of list-continue mechanism ensures that the memory consumption
    // by this operator always stays under a desired threshold irrespective of the
    // number of resource instances for any kubernetes resource
    defaultListLimit = 5
)

var (
    _ manager.Runnable = &GenericReconciler{}
)

// GenericReconciler watches a defined object
type GenericReconciler struct {
    listLimit       int64
    watchNamespaces *watchNamespacesCache
    client          client.Client
    discoveryClient *discovery.DiscoveryClient
}

// NewGenericReconciler returns a GenericReconciler struct
func NewGenericReconciler(kc *rest.Config) (*GenericReconciler, error) {
    return &GenericReconciler{
        listLimit:       getListLimit(),
        watchNamespaces: newWatchNamespacesCache(),
    }, nil
}

func getListLimit() int64 {
    listLimit := defaultListLimit
    listLimitEnvVal := os.Getenv("LIST_LIMIT_PER_QUERY")
    if listLimitEnvVal != "" {
        var err error
        listLimit, err = strconv.Atoi(listLimitEnvVal)
        if err != nil {
            panic(err.Error())
        }
    }
    return int64(listLimit)
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
    t := time.NewTicker(resyncPeriod(defaultReconcileInterval)())
    defer t.Stop()

    err := gr.reconcileEverything(ctx)
    if err != nil {
        return err
    }
    for {
        select {
        case <-t.C:
            // Try to reconcile with default exponential backoff until success.
            err := wait.ExponentialBackoffWithContext(
                ctx, retry.DefaultBackoff,
                func() (done bool, err error) {
                    return true, gr.reconcileEverything(ctx)
                })
            if err != nil && !errors.Is(err, context.Canceled) {
                // could not recover
                return err
            }

        case <-ctx.Done():
            // stop reconciling
            return nil
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

func (gr *GenericReconciler) processObjectInstances(ctx context.Context, gvk schema.GroupVersionKind, namespace string) error {
    gvk.Kind = gvk.Kind + "List"
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

func (gr *GenericReconciler) reconcile(ctx context.Context, obj client.Object) error {
    request := reconcile.Request{
        NamespacedName: client.ObjectKeyFromObject(obj),
    }

    var log = logf.Log.WithName(fmt.Sprintf("%s Validation", obj.GetObjectKind().GroupVersionKind()))
    reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
    reqLogger.V(2).Info("Reconcile", "Kind", obj.GetObjectKind().GroupVersionKind())
    reqLogger.Info("Reconcile", "Kind", obj.GetObjectKind().GroupVersionKind())

    // TODO: figure out how to delete metrics when a resource instance is deleted from etcd,
    // as we are not using informers anymore the current reconciler will not know when a resource deleted and
    // which instance is deleted. Need to add a inmemory cache and identify deletions (as a first solution)
    // e.g. keep an index of UIDs for every GVK to check if something was deleted between runs.
    //var deleted bool
    validations.RunValidations(request, obj)
    return nil
}
