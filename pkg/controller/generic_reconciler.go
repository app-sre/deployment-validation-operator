package controller

import (
	"context"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var namespaceIgnore *regexp.Regexp

func init() {
	if os.Getenv("NAMESPACE_IGNORE_PATTERN") != "" {
		namespaceIgnore = regexp.MustCompile(os.Getenv("NAMESPACE_IGNORE_PATTERN"))
	}
}

var (
	_ manager.Runnable = &GenericReconciler{}
)

// default interval to run validations in.
// A 10 percent jitter will be added to the reconcile interval between reconcilers,
// so that not all reconcilers will not send list requests simultaneously.
const defaultReconcileInterval = 1 * time.Minute

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

// AddToManager will add the reconciler for the configured obj to a manager.
func (gr *GenericReconciler) AddToManager(mgr manager.Manager) error {
	gr.client = mgr.GetClient()
	return mgr.Add(gr)
}

// Start validating the given object kind every interval.
func (gr *GenericReconciler) Start(ctx context.Context) error {
	t := time.NewTicker(resyncPeriod(defaultReconcileInterval)())
	defer t.Stop()

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
	// todo: Get List GVK somewhere, this doesn't work till "generateObjects" is refactored.
	list := &unstructured.UnstructuredList{}
	if err := gr.client.List(ctx, list); err != nil {
		return fmt.Errorf("listing %s: %w", gr.reconciledKind, err)
	}

	for i := range list.Items {
		if err := gr.reconcile(ctx, &list.Items[i]); err != nil {
			return err
		}
	}
	return nil
}

func (gr *GenericReconciler) reconcile(ctx context.Context, obj client.Object) error {
	request := reconcile.Request{
		NamespacedName: client.ObjectKeyFromObject(obj),
	}

	var log = logf.Log.WithName(fmt.Sprintf("%sController", gr.reconciledKind))
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.V(2).Info("Reconcile", "Kind", gr.reconciledKind)

	if namespaceIgnore != nil && namespaceIgnore.Match([]byte(request.Namespace)) {
		reqLogger.Info("Ignoring object as it matches namespace ignore pattern")
		return nil
	}

	// todo: figure out how to delete metrics,
	// e.g. keep an index of UIDs for every GVK to check if something was deleted between runs.
	var deleted bool
	validations.RunValidations(request, obj, gr.reconciledKind, deleted)
	return nil
}
