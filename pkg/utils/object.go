package utils

import (
	"golang.stackrox.io/kube-linter/pkg/objectkinds"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var deploymentLikeMatcher, _ = objectkinds.ConstructMatcher(objectkinds.DeploymentLike)

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
