package controller

import (
	"context"
	"fmt"
	"strings"

	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	managerCache "sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type configMapWatcher struct {
	client         client.Client
	cache          managerCache.Cache
	disabledChecks []string      // TODO - TBD disable checks
	ch             chan struct{} // TODO - TBD configmap struct
}

var configMapName = "deployment-validation-operator-config"
var configMapNamespace = "deployment-validation-operator"

// NewBasicConfigMapWatcher provides a basic way to retrieve an existing configuration ConfigMap
// This way of initiating the watcher provides the configuration with pull functionality
// through the GetDisabledChecks method.
// To Be Deprecated
func NewBasicConfigMapWatcher(cfg *rest.Config) (configMapWatcher, error) {
	var cmw configMapWatcher

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return configMapWatcher{}, fmt.Errorf("initializing clientset: %w", err)
	}

	err = cmw.withoutInformer(clientset.CoreV1())
	if err != nil {
		return configMapWatcher{}, fmt.Errorf("gathering starting configmap: %w", err)
	}

	return cmw, nil
}

// only used with basic functionality
func (cmw *configMapWatcher) withoutInformer(client corev1.CoreV1Interface) error {

	cm, err := client.ConfigMaps(configMapNamespace).Get(context.Background(), configMapName, v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("gathering starting configmap: %w", err)
	}

	cmw.disabledChecks = strings.Split(cm.Data["disabled-checks"], ",")

	return nil
}

// GetDisabledChecks returns disable checks from an existing ConfigMap only if the watcher was initiated
// with the NewBasicConfigMapWatcher method
func (cmw configMapWatcher) GetDisabledChecks() []string {
	return cmw.disabledChecks
}

// NewConfigMapWatcher provides a watcher that runs in a manager
// and sends push notifications to the ConfigChanged method when the configuration is updated.
// constraint - The current validation engine cannot handle these notifications.
func NewConfigMapWatcher(cc client.Client, mc managerCache.Cache) configMapWatcher {
	ch := make(chan struct{})
	return configMapWatcher{
		client: cc,
		cache:  mc,
		ch:     ch,
	}
}

// Start method is used by a Manager
func (cmw configMapWatcher) Start(ctx context.Context) error {
	var configMap apicorev1.ConfigMap
	var cmKey = client.ObjectKey{
		Name:      configMapName,
		Namespace: configMapNamespace,
	}

	err := cmw.client.Get(ctx, cmKey, &configMap)
	if err != nil {
		return err
	}

	inf, err := cmw.cache.GetInformer(ctx, &configMap)
	if err != nil {
		return err
	}

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			// TODO - Validate new configmap
			fmt.Println("new configmap detected")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			obj, err := meta.Accessor(newObj)
			if err != nil {
				fmt.Println("error // retrieving updated configmap", err)
			}

			if obj.GetName() == configMapName {
				// TODO - Validate new configmap
				fmt.Println("configmap has been updated")
				cmw.ch <- struct{}{}
			}
		},
	})
	return nil
}

// ConfigChanged receives push notifications when the configuration is updated
func (cmw *configMapWatcher) ConfigChanged() <-chan struct{} {
	return cmw.ch
}
