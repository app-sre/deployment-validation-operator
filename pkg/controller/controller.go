package controller

import (
	appsv1 "k8s.io/api/apps/v1"

	"k8s.io/apimachinery/pkg/runtime"

	"sigs.k8s.io/controller-runtime/pkg/manager"
)

var desiredControllerKinds []runtime.Object = []runtime.Object{
	&appsv1.Deployment{},
	&appsv1.ReplicaSet{},
}

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager) error {
	for _, obj := range desiredControllerKinds {
		c := NewGenericReconciler(obj)
		err := c.AddToManager(m)
		if err != nil {
			return err
		}
	}
	return nil
}
