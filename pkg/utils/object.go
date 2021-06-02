package utils

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IsController returns true if the object passed in does
// not have a controller associated with it
func IsController(obj metav1.Object) bool {
	controller := metav1.GetControllerOf(obj)
	return controller == nil
}

// IsOwner returns true if the object passed has no owner references
func IsOwner(obj metav1.Object) bool {
	refs := obj.GetOwnerReferences()
	return len(refs) == 0
}
