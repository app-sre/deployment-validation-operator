package controller

import (
	"github.com/app-sre/dv-operator/pkg/controller/replicaset"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, replicaset.Add)
}
