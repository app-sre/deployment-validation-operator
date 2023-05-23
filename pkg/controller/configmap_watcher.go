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
	disabledChecks []string
}

// Basic logic (getting info from ConfigMap before the manager runs)
// This should be later overriden by controller logic
// TODO - document properly
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

func (cmw *configMapWatcher) withoutInformer(client corev1.CoreV1Interface) error {

	cm, err := client.ConfigMaps("deployment-validation-operator").
		Get(context.Background(), "deployment-validation-operator-config", v1.GetOptions{})
	if err != nil {
		return fmt.Errorf("gathering starting configmap: %w", err)
	}

	cmw.disabledChecks = strings.Split(cm.Data["disabled-checks"], ",")

	return nil
}

func (cmw configMapWatcher) GetDisabledChecks() []string {
	return cmw.disabledChecks
}

// Controller logic (setting up a informer to watch over updates of the configmap)
// This should be the behaviour once validation engine cares about configuration updates
// TODO - document properly
func NewConfigMapWatcher(cc client.Client, mc managerCache.Cache) configMapWatcher {
	return configMapWatcher{
		client: cc,
		cache:  mc,
	}
}

func (c configMapWatcher) Start(ctx context.Context) error {
	var configMap apicorev1.ConfigMap
	var cmKey = client.ObjectKey{
		Name:      "deployment-validation-operator-config",
		Namespace: "deployment-validation-operator",
	}

	err := c.client.Get(ctx, cmKey, &configMap)
	if err != nil {
		return err
	}

	inf, err := c.cache.GetInformer(ctx, &configMap)
	if err != nil {
		return err
	}

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			fmt.Println("warning // new configmap detected")
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			obj, err := meta.Accessor(newObj)
			if err != nil {
				fmt.Println("error // retrieving updated configmap", err)
			}

			if obj.GetName() == "deployment-validation-operator-config" {
				fmt.Println("debug // OLD ", oldObj)
				fmt.Println("debug // NEW ", newObj)
			}
		},
	})
	return nil
}
