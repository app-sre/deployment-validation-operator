package configmap

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"golang.stackrox.io/kube-linter/pkg/config"

	"github.com/app-sre/deployment-validation-operator/pkg/validations"
	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	apicorev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Watcher struct {
	clientset kubernetes.Interface
	cfg       config.Config
	ch        chan struct{}
	logger    logr.Logger
	namespace string
}

var configMapName = "deployment-validation-operator-config"
var configMapDataAccess = "deployment-validation-operator-config.yaml"

// NewWatcher creates a new Watcher instance for observing changes to a ConfigMap.
//
// Parameters:
//   - cfg: A pointer to a rest.Config representing the Kubernetes client configuration.
//
// Returns:
//   - A pointer to a Watcher instance for monitoring changes to DVO ConfigMap resource.
//   - An error if there's an issue while initializing the Kubernetes clientset.
func NewWatcher(cfg *rest.Config) (*Watcher, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("initializing clientset: %w", err)
	}

	// the Informer will use this to monitor the namespace for the ConfigMap.
	namespace, err := getPodNamespace()
	if err != nil {
		return nil, fmt.Errorf("getting namespace: %w", err)
	}

	return &Watcher{
		clientset: clientset,
		logger:    log.Log.WithName("ConfigMapWatcher"),
		ch:        make(chan struct{}),
		namespace: namespace,
	}, nil
}

// Start will update the channel structure with new configuration data from ConfigMap update event
func (cmw *Watcher) Start(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		cmw.clientset, time.Second*30, informers.WithNamespace(cmw.namespace),
	)
	informer := factory.Core().V1().ConfigMaps().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{ // nolint:errcheck
		AddFunc: func(obj interface{}) {
			newCm := obj.(*apicorev1.ConfigMap)

			if configMapName != newCm.GetName() {
				return
			}

			cmw.logger.Info(
				"a ConfigMap has been created under watched namespace",
				"name", newCm.GetName(),
				"namespace", newCm.GetNamespace(),
			)

			cfg, err := readConfig(newCm.Data[configMapDataAccess])
			if err != nil {
				cmw.logger.Error(err, "ConfigMap data format")
				return
			}

			cmw.cfg = cfg

			cmw.ch <- struct{}{}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			newCm := newObj.(*apicorev1.ConfigMap)

			// This is sometimes triggered even if no change was due to the ConfigMap
			if configMapName != newCm.GetName() || reflect.DeepEqual(oldObj, newObj) {
				return
			}

			cmw.logger.Info(
				"a ConfigMap has been updated under watched namespace",
				"name", newCm.GetName(),
				"namespace", newCm.GetNamespace(),
			)

			cfg, err := readConfig(newCm.Data[configMapDataAccess])
			if err != nil {
				cmw.logger.Error(err, "ConfigMap data format")
				return
			}

			cmw.cfg = cfg

			cmw.ch <- struct{}{}
		},
		DeleteFunc: func(oldObj interface{}) {
			cm := oldObj.(*apicorev1.ConfigMap)

			cmw.logger.Info(
				"a ConfigMap has been deleted under watched namespace",
				"name", cm.GetName(),
				"namespace", cm.GetNamespace(),
			)

			cmw.cfg = config.Config{
				Checks: validations.GetDefaultChecks(),
			}

			cmw.ch <- struct{}{}
		},
	})

	factory.Start(ctx.Done())

	return nil
}

// ConfigChanged receives push notifications when the configuration is updated
func (cmw *Watcher) ConfigChanged() <-chan struct{} {
	return cmw.ch
}

// GetConfig returns a previously saved kube-linter Config structure
func (cmw *Watcher) GetConfig() config.Config {
	return cmw.cfg
}

// readConfig returns a valid Kube-linter Config structure
// based on the checks received by the string
func readConfig(data string) (config.Config, error) {
	var cfg config.Config

	err := yaml.Unmarshal([]byte(data), &cfg, yaml.DisallowUnknownFields)
	if err != nil {
		return cfg, fmt.Errorf("unmarshalling configmap data: %w", err)
	}

	return cfg, nil
}

func getPodNamespace() (string, error) {
	namespace, exists := os.LookupEnv("POD_NAMESPACE")
	if !exists {
		return "", fmt.Errorf("could not find DVO pod")
	}

	return namespace, nil
}
