package controller

import (
	"context"
	"fmt"
	"time"

	"golang.stackrox.io/kube-linter/pkg/config"
	"gopkg.in/yaml.v3"

	apicorev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
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

type ConfigMapWatcher struct {
	clientset kubernetes.Interface
	checks    KubeLinterChecks
	ch        chan config.Config
}

var configMapName = "deployment-validation-operator-custom-config"
var configMapNamespace = "deployment-validation-operator"
var configMapDataAccess = "deployment-validation-operator-config.yaml"

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

// GetStaticKubelinterConfig returns the ConfigMap's checks configuration
func (cmw *ConfigMapWatcher) GetStaticKubelinterConfig(ctx context.Context) (config.Config, error) {
	var cfg config.Config

	cm, err := cmw.clientset.CoreV1().
		ConfigMaps(configMapNamespace).Get(ctx, configMapName, v1.GetOptions{})
	if err != nil {
		return cfg, fmt.Errorf("gathering starting configmap: %w", err)
	}

	return cmw.getKubeLinterConfig(cm.Data[configMapDataAccess])
}

// StartInformer will update the channel structure with new configuration data from ConfigMap update event
func (cmw *ConfigMapWatcher) StartInformer(ctx context.Context) error {
	factory := informers.NewSharedInformerFactoryWithOptions(
		cmw.clientset, time.Second*30, informers.WithNamespace(configMapNamespace),
	)
	informer := factory.Core().V1().ConfigMaps().Informer()

	informer.AddEventHandler(cache.ResourceEventHandlerFuncs{ // nolint:errcheck
		UpdateFunc: func(oldObj, newObj interface{}) {
			newCm := newObj.(*apicorev1.ConfigMap)

			fmt.Printf("ConfigMap updated: %s/%s\n", newCm.Namespace, newCm.Name)

			cfg, err := cmw.getKubeLinterConfig(newCm.Data[configMapDataAccess])
			if err != nil {
				fmt.Printf("Error: unmarshalling configmap data: %s", err.Error())
				return
			}

			cmw.ch <- cfg
		},
	})

	factory.Start(ctx.Done())

	return nil
}

// ConfigChanged receives push notifications when the configuration is updated
func (cmw *ConfigMapWatcher) ConfigChanged() <-chan config.Config {
	return cmw.ch
}

// getKubeLinterConfig returns a valid Kube-linter Config structure
// based on the checks received by the string
func (cmw *ConfigMapWatcher) getKubeLinterConfig(data string) (config.Config, error) {
	var cfg config.Config

	err := yaml.Unmarshal([]byte(data), &cmw.checks)
	if err != nil {
		return cfg, fmt.Errorf("unmarshalling configmap data: %w", err)
	}

	cfg.Checks = config.ChecksConfig(cmw.checks.Checks)

	return cfg, nil
}
