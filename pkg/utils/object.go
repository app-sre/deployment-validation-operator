package utils

import (
	//	"golang.stackrox.io/kube-linter/pkg/k8sutil"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func IsController(obj metav1.Object) bool {
	controller := metav1.GetControllerOf(obj)
	return controller == nil
}
