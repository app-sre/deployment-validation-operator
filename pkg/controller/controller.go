package controller

import (
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddControllersToManager adds all Controllers to the Manager
func AddControllersToManager(m manager.Manager) error {
	c, err := NewGenericReconciler()
	if err != nil {
		return err
	}

	err = c.AddToManager(m)
	if err != nil {
		return err
	}

	return nil
}
