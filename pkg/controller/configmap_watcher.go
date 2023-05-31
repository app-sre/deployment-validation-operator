package controller

import (
	"context"
	"fmt"
	"strings"
	"time"

	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type ConfigMapWatcher struct {
	clientset kubernetes.Interface
	ch        chan struct{} // TODO - TBD configmap struct
}

var configMapName = "deployment-validation-operator-config"
var configMapNamespace = "deployment-validation-operator"

// NewConfigMapWatcher returns a watcher that can be used both:
// basic: with GetStaticDisabledChecks method, it returns an existent ConfigMap data's disabled check
// dynamic: with StartInformer it sets an Informer that will be triggered on ConfigMap update
func NewConfigMapWatcher(cfg *rest.Config) (ConfigMapWatcher, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return ConfigMapWatcher{}, fmt.Errorf("initializing clientset: %w", err)
	}

	return ConfigMapWatcher{
		clientset: clientset,
	}, nil
}

// GetStaticDisabledChecks returns an existent ConfigMap data's disabled checks, if they exist
func (cmw *ConfigMapWatcher) GetStaticDisabledChecks(ctx context.Context) ([]string, error) {
	cm, err := cmw.clientset.CoreV1().
		ConfigMaps(configMapNamespace).Get(ctx, configMapName, v1.GetOptions{})
	if err != nil {
		return []string{}, fmt.Errorf("gathering starting configmap: %w", err)
	}

	// TODO - Fix dummy return based on data being check1,check2,check3...
	return strings.Split(cm.Data["disabled-checks"], ","), nil
}

// StartInformer will update the channel structure with new configuration data from ConfigMap update event
func (cmw *ConfigMapWatcher) StartInformer(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		cmw.clientset, time.Second*30, informers.WithNamespace(configMapNamespace),
	)
	informer := factory.Core().V1().ConfigMaps().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{ // nolint:errcheck
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldCm := oldObj.(*apicorev1.ConfigMap)
			newCm := newObj.(*apicorev1.ConfigMap)

			fmt.Printf("oldCm: %v\n", oldCm)
			fmt.Printf("ConfigMap updated: %s/%s\n", newCm.Namespace, newCm.Name)

			// TODO - Validate new configmap
			cmw.ch <- struct{}{}
		},
	})

	factory.Start(ctx.Done())

	return nil
}

// ConfigChanged receives push notifications when the configuration is updated
func (cmw *ConfigMapWatcher) ConfigChanged() <-chan struct{} {
	return cmw.ch
}
