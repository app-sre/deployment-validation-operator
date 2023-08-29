package configmap

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"time"

	"golang.stackrox.io/kube-linter/pkg/config"
	"gopkg.in/yaml.v3"

	"github.com/go-logr/logr"
	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// this structure mirrors Kube-Linter configuration structure
// it is used as a bridge to unmarshall ConfigMap data
// doc: https://pkg.go.dev/golang.stackrox.io/kube-linter/pkg/config#Config
type KubeLinterChecks struct {
	Checks struct {
		AddAllBuiltIn        bool     `yaml:"addAllBuiltIn,omitempty"`
		DoNotAutoAddDefaults bool     `yaml:"doNotAutoAddDefaults,omitempty"`
		Exclude              []string `yaml:"exclude,omitempty"`
		Include              []string `yaml:"include,omitempty"`
		IgnorePaths          []string `yaml:"ignorePaths,omitempty"`
	} `yaml:"checks"`
}

type Watcher struct {
	clientset kubernetes.Interface
	checks    KubeLinterChecks
	ch        chan config.Config
	logger    logr.Logger
	namespace string
}

var configMapName = "deployment-validation-operator-config"
var configMapNamespace = "deployment-validation-operator"
var configMapDataAccess = "deployment-validation-operator-config.yaml"

// NewWatcher creates a new Watcher instance for observing changes to a ConfigMap.
//
// Parameters:
//   - cfg: A pointer to a rest.Config representing the Kubernetes client configuration.
//
// Returns:
//   - A Watcher instance for monitoring changes to DVO ConfigMap resource if the initialization is successful.
//   - An error if there's an issue while initializing the Kubernetes clientset.
func NewWatcher(cfg *rest.Config) (Watcher, error) {
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return Watcher{}, fmt.Errorf("initializing clientset: %w", err)
	}

	// the Informer will use this to monitor the namespace for the ConfigMap.
	namespace, err := getPodNamespace()
	if err != nil {
		return Watcher{}, fmt.Errorf("getting namespace: %w", err)
	}

	return Watcher{
		clientset: clientset,
		logger:    log.Log.WithName("ConfigMapWatcher"),
		ch:        make(chan config.Config),
		namespace: namespace,
	}, nil
}

// GetStaticKubelinterConfig returns the ConfigMap's checks configuration
func (cmw *Watcher) GetStaticKubelinterConfig(ctx context.Context) (config.Config, error) {
	cm, err := cmw.clientset.CoreV1().
		ConfigMaps(cmw.namespace).Get(ctx, configMapName, v1.GetOptions{})
	if err != nil {
		return config.Config{}, fmt.Errorf("getting initial configuration: %w", err)
	}

	return cmw.getKubeLinterConfig(cm.Data[configMapDataAccess])
}

// Start will update the channel structure with new configuration data from ConfigMap update event
func (cmw Watcher) Start(ctx context.Context) error {
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

			cfg, err := cmw.getKubeLinterConfig(newCm.Data[configMapDataAccess])
			if err != nil {
				cmw.logger.Error(err, "ConfigMap data format")
				return
			}

			cmw.ch <- cfg
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

			cfg, err := cmw.getKubeLinterConfig(newCm.Data[configMapDataAccess])
			if err != nil {
				cmw.logger.Error(err, "ConfigMap data format")
				return
			}

			cmw.ch <- cfg
		},
	})

	factory.Start(ctx.Done())

	return nil
}

// ConfigChanged receives push notifications when the configuration is updated
func (cmw *Watcher) ConfigChanged() <-chan config.Config {
	return cmw.ch
}

// getKubeLinterConfig returns a valid Kube-linter Config structure
// based on the checks received by the string
func (cmw *Watcher) getKubeLinterConfig(data string) (config.Config, error) {
	var cfg config.Config

	err := yaml.Unmarshal([]byte(data), &cmw.checks)
	if err != nil {
		return cfg, fmt.Errorf("unmarshalling configmap data: %w", err)
	}

	cfg.Checks = config.ChecksConfig(cmw.checks.Checks)

	return cfg, nil
}

func getPodNamespace() (string, error) {
	namespace, exists := os.LookupEnv("POD_NAMESPACE")
	if !exists {
		return "", fmt.Errorf("could not find DVO pod")
	}

	return namespace, nil
}
