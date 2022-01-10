package controller

import (
	"fmt"

	"github.com/app-sre/deployment-validation-operator/pkg/utils"
	osappsscheme "github.com/openshift/client-go/apps/clientset/versioned/scheme"

	"golang.stackrox.io/kube-linter/pkg/objectkinds"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"k8s.io/client-go/kubernetes/scheme"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// Openshift Kinds
var osKinds = map[string]bool{}

var log = logf.Log.WithName("DeploymentValidation")

// AddControllersToManager adds all Controllers to the Manager
func AddControllersToManager(m manager.Manager) error {
	for _, obj := range generateObjects() {
		c := NewGenericReconciler(obj)
		err := c.AddToManager(m)
		if err != nil {
			return err
		}
	}
	return nil
}

func generateObjects() []runtime.Object {
	// Construct the gvks for objects to watch.  Remove the Any
	// kind or else all objects kinds will be watched.  That would
	// be bad.
	log.Info("Generating Objects.")
	kubeLinterKinds := objectkinds.AllObjectKinds()
	for i := range kubeLinterKinds {
		if kubeLinterKinds[i] == objectkinds.Any {
			kubeLinterKinds = append(kubeLinterKinds[:i], kubeLinterKinds[i+1:]...)
			break
		}
	}

	kubeLinterMatcher, err := objectkinds.ConstructMatcher(kubeLinterKinds...)
	if err != nil {
		// TODO log error or exit?
		return []runtime.Object{}
	}

	kubeScheme := scheme.Scheme

	// Create the set of api group/versions with priorities.  This provides ability
	// to choose the highest priority version for an api group.
	preferredGV := make(map[schema.GroupVersion]int)
	currentGroup := ""
	priority := 0
	for _, gv := range kubeScheme.PrioritizedVersionsAllGroups() {
		if currentGroup != gv.Group {
			currentGroup = gv.Group
			priority = 0
		}
		preferredGV[gv] = priority
		priority++
	}

	gvks := make(map[schema.GroupKind]schema.GroupVersionKind)
	for gvk := range kubeScheme.AllKnownTypes() {
		if kubeLinterMatcher.Matches(gvk) {
			current, ok := gvks[gvk.GroupKind()]
			if !ok || preferredGV[gvk.GroupVersion()] < preferredGV[current.GroupVersion()] {
				gvks[gvk.GroupKind()] = gvk
			}
		}
	}

	osAppsScheme := osappsscheme.Scheme
	for gvk := range osAppsScheme.AllKnownTypes() {
		osKinds[gvk.Kind] = true
		if kubeLinterMatcher.Matches(gvk) {
			if _, ok := gvks[gvk.GroupKind()]; !ok {
				gvks[gvk.GroupKind()] = gvk
			}
		}
	}

	objs := []runtime.Object{}

	allowOpenshiftKinds, _ := utils.IsOpenshift(osKinds)

	for gk := range gvks {
		if osKinds[gk.Kind] && !allowOpenshiftKinds {
			log.Info(fmt.Sprintf("Non-Openshift Environment. Skipping registering kind: %s", gk.Kind))
			continue
		}
		obj, err := kubeScheme.New(gvks[gk])
		if err == nil {
			objs = append(objs, obj)
		} else {
			// Try this as an openshift object
			obj, err := osAppsScheme.New(gvks[gk])
			if err == nil {
				objs = append(objs, obj)
			}
		}
	}

	return objs
}
