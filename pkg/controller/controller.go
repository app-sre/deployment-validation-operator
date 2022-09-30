package controller

import (
    "math/rand"
    "time"

    logf "sigs.k8s.io/controller-runtime/pkg/log"
    "sigs.k8s.io/controller-runtime/pkg/manager"
)

var log = logf.Log.WithName("DeploymentValidationRun")

// AddControllersToManager adds all Controllers to the Manager
func AddControllersToManager(m manager.Manager) error {
    kubeconfig := m.GetConfig()
    c, err := NewGenericReconciler(kubeconfig)
    if err != nil {
        return err
    }

    err = c.AddToManager(m)
    if err != nil {
        return err
    }

    return nil
}

// resyncPeriod returns a function which generates a duration each time it is
// invoked; this is so that multiple controllers don't get into lock-step and all
// hammer the apiserver with list requests simultaneously.
func resyncPeriod(resync time.Duration) func() time.Duration {
    return func() time.Duration {
        // the factor will fall into [0.9, 1.1)
        factor := rand.Float64()/5.0 + 0.9 //nolint:gosec
        return time.Duration(float64(resync.Nanoseconds()) * factor)
    }
}
