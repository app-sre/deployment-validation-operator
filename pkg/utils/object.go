package utils

import (
	"golang.stackrox.io/kube-linter/pkg/objectkinds"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var log = logf.Log.WithName("DeploymentValidation")

var deploymentLikeMatcher, _ = objectkinds.ConstructMatcher(objectkinds.DeploymentLike)

// IsController returns true if the object passed in does
// not have a controller associated with it
func IsController(obj metav1.Object) bool {
	controller := metav1.GetControllerOf(obj)
	return controller == nil
}

// IsOwner returns false if the object is deployment-like resource and owned by a deployment-like resource
func IsOwner(obj client.Object) bool {
	gvk := obj.GetObjectKind().GroupVersionKind()
	// no need to check owner for none deployment-like resource
	if deploymentLikeMatcher.Matches(gvk) {
		for _, ref := range obj.GetOwnerReferences() {
			refGvk := schema.FromAPIVersionAndKind(ref.APIVersion, ref.Kind)
			// get a deployment-like owner
			if deploymentLikeMatcher.Matches(refGvk) {
				return false
			}
		}
	}
	return true
}

//IsOpenshift identify environment and returns true if its openshift else false
func IsOpenshift(osKind map[string]bool) (bool, error) {
	log.Info("Checking User Environment in IsOpenshift.")
	config, err := rest.InClusterConfig()
	if err != nil {
		return false, err
	}
	discoveryclient, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	errs := []error{}
	lists, err := discoveryclient.ServerPreferredResources()
	if err != nil {
		errs = append(errs, err)
	}

	for _, list := range lists {
		if len(list.APIResources) == 0 {
			continue
		}
		// Check for acceptable kinds in current Env
		for _, resource := range list.APIResources {
			if len(resource.Verbs) == 0 {
				continue
			}
			if osKind[resource.Kind] {
				return true, nil
			}
		}
	}

	return false, err
}
